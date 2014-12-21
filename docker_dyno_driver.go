package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type DockerDynoDriver struct {
	d     *Docker
	state DynoState

	cmd       *exec.Cmd
	container *docker.Container
	waiting   chan error
}

func NewDockerDynoDriver() *DockerDynoDriver {
	return &DockerDynoDriver{}
}

func (dd *DockerDynoDriver) State() DynoState {
	return dd.state
}

func (dd *DockerDynoDriver) Start(b *Bundle) error {
	if dd.d == nil {
		dd.d = &Docker{}
		if err := dd.d.Connect(); err != nil {
			dd.d = nil
			return err
		}
	}

	si, err := dd.d.StackStat("cedar-14")
	if err != nil {
		return err
	}

	log.Printf("StackImage %+v", si)
	imageName, err := dd.d.BuildSlugImage(si, b)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Built image successfully")

	// Fill environment vector from Heroku configuration.
	env := make([]string, 0)
	for k, v := range b.config {
		env = append(env, k+"="+v)
	}

	dd.container, err = dd.d.c.CreateContainer(docker.CreateContainerOptions{
		Name: fmt.Sprintf("%v-%v", imageName, int32(time.Now().Unix())),
		Config: &docker.Config{
			Cmd:   b.argv,
			Env:   env,
			Image: imageName,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	err = dd.d.c.StartContainer(dd.container.ID, &docker.HostConfig{})
	if err != nil {
		log.Fatal(err)
	}

	dd.state = Started

	return nil
}

func (dd *DockerDynoDriver) Stop() error {
	// If we could never start the process, don't worry about stopping it. May
	// occur in cases like if Docker was down.
	if dd.cmd == nil {
		return nil
	}

	p := dd.cmd.Process

	group, err := os.FindProcess(-1 * p.Pid)
	if err != nil {
		return err
	}

	// Begin graceful shutdown via SIGTERM.
	group.Signal(syscall.SIGTERM)

	for {
		select {
		case <-time.After(10 * time.Second):
			log.Println("sigkill", group)
			group.Signal(syscall.SIGKILL)
		case err := <-dd.waiting:
			log.Println("waited", group)
			dd.state = Stopped
			return err
		}
		log.Println("spin", group)
		time.Sleep(1)
	}
}