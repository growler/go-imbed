package imbed

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func rmtree(name string) {
	var files []string
	var dirs []string
	filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			dirs = append(dirs, path)
		} else {
			files = append(files, path)
		}
		return nil
	})
	for j := len(files) - 1; j >= 0; j-- {
		os.Remove(files[j])
	}
	for j := len(dirs) - 1; j >= 0; j-- {
		os.Remove(dirs[j])
	}
}

func TestGenerate(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "go-imbed-test")
	if err != nil {
		t.Fatal(err)
	}
	defer rmtree(tmp)
	pkgDir := filepath.Join(tmp, "src", "pkg")
	err = os.MkdirAll(filepath.Join(pkgDir, "internal"), 0700)
	if err != nil {
		t.Fatal(err)
	}
	mainf, err := os.OpenFile(filepath.Join(pkgDir, "main.go"), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.WriteString(mainf, `package main

import _ "pkg/internal/data"

func main() {
}
`)
	mainf.Close()
	if err != nil {
		t.Fatal(err)
	}
	targetPkg := filepath.Join(tmp, "src", "pkg", "internal", "data")
	for i := 0; i < (1 << maxFlag); i++ {
		cmd := exec.Command("go", "install", "pkg")
		cmd.Env = append(os.Environ(), "GOPATH="+tmp)
		cmd.Dir = tmp
		flags := ImbedFlag(i)
		err := Imbed("../example/site", targetPkg, "data", flags)
		if err != nil {
			t.Fatalf("error embedding with flags %s: %s", flags.String(), err)
		}
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			t.Fatalf("error compiling target with flags %s\n", flags.String())
		}
		cmd = exec.Command("go", "test", "-v", "pkg/internal/data")
		cmd.Env = append(os.Environ(), "GOPATH="+tmp)
		cmd.Dir = tmp
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			t.Fatalf("error testing target with flags %s\n", flags.String())
		}
	}
}
