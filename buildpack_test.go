package hsup

import (
	"archive/tar"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type testDir struct {
	T     *testing.T
	Where string
}

func (td *testDir) Init() string {
	name, err := ioutil.TempDir("", "hsuptest_")
	td.Where = name
	if err != nil {
		td.T.Fatalf("Could not create temporary directory for test: %v",
			err)
	}

	return name
}

func (td *testDir) Cleanup() {
	err := os.RemoveAll(td.Where)
	if err != nil {
		panic(err)
	}
}

func (td *testDir) mustWrite(dst, contents string) {
	td.mustWriteBytes(dst, []byte(contents))
}

func (td *testDir) mustWriteBytes(dst string, contents []byte) {
	place := filepath.Join(td.Where, dst)
	if err := ioutil.WriteFile(place, contents, 0666); err != nil {
		panic(err)
	}
}

func (td *testDir) RackApp() {
	td.mustWrite("config.ru", `run lambda do |env|
[200, {'Content-Type'=>'text/plain'},
StringIO.new(\"Hello World!\n\")]
}
`)
	td.mustWrite("Gemfile", "source 'https://rubygems.org'\ngem 'rack'\n")
	td.mustWrite("Gemfile.lock", `GEM
  remote: https://rubygems.org/
  specs:
    rack (1.6.0)

PLATFORMS
  ruby

DEPENDENCIES
  rack
`)
}

func memoryTar() (*tar.Writer, *bytes.Buffer) {
	tarBuf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(tarBuf)
	return tw, tarBuf
}

func TestEmptyDir(t *testing.T) {
	td := testDir{T: t}
	td.Init()
	defer td.Cleanup()
}

func TestRackApp(t *testing.T) {
	td := testDir{T: t}
	td.Init()
	// defer td.Cleanup()
	td.RackApp()

	tw, buf := memoryTar()
	rt := recursiveTar{Tw: tw, SrcDir: td.Where}
	rt.Run()
	tw.Close()
	td.mustWriteBytes("blah.tar.gz", buf.Bytes())
}
