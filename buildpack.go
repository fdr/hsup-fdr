package hsup

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type recursiveTar struct {
	Tw     *tar.Writer
	SrcDir string
}

func (rt *recursiveTar) walker(path string, f os.FileInfo, err error) error {
	target, _ := os.Readlink(path)
	hdr, err := tar.FileInfoHeader(f, target)
	if err != nil {
		return err
	}

	log.Println(path, rt.SrcDir)
	if !strings.HasPrefix(path, rt.SrcDir) {
		log.Println("ignoring out-of-source tree file:", path)
		return nil
	} else if path == rt.SrcDir {
		hdr.Name = "src"
	} else {
		hdr.Name = filepath.Join("src", hdr.Name)
	}

	switch hdr.Typeflag {
	case tar.TypeReg:
		fallthrough
	case tar.TypeRegA:
		hdr.Size = f.Size()
		rt.Tw.WriteHeader(hdr)
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		io.Copy(rt.Tw, file)
	default:
		rt.Tw.WriteHeader(hdr)
	}

	fmt.Printf("Visited: %s\n", path)
	return nil
}

func (rt *recursiveTar) Run() error {
	err := filepath.Walk(rt.SrcDir, rt.walker)
	return err
}
