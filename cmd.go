// Copyright 2017 Alexey Naidyonov. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

package main

import (
	"flag"
	"fmt"
	"os"
	"mime"
	"bytes"
	"text/template"
	"github.com/growler/go-imbed/imbed"
	"path/filepath"
	"io/ioutil"
	"os/exec"
	"io"
)

var usage = template.Must(template.New("").Parse(
	`A simple source generator to embed resources into Go executable

Usage:
    {{.Binary}} [options] <source-content-path> <target-package-path>

Options:
{{.Options}}

Generator will build a golang assembly file along with assets access APIs.

All the generated sources will be placed into <target-package> relative to the current
working directory (so generator is convenient to use with go:generate). It is recommended to
use internal package (i.e., "internal/site")

The typical usage would be:

// go:generate go-imbed site-source internal/site
package main

import (
    "net/http"
    "fmt"
    "internal/site"
)

func main() {
    http.HandleFunc("/", site.ServeHTTP)
    if err := http.ListenAndServe(":8080", nil); err != nil{
        fmt.Println(err)
    }
}
`))

var cli *flag.FlagSet

var (
	disableCompression bool
	disableHTTPHandler bool
	enableFS           bool
	enableUnionFS      bool
	enableHTTPFS       bool
	enableRawBytes     bool
	pkgName            string
	makeBinary         bool
)

func init() {

	cli = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	cli.StringVar(&pkgName, "pkg", "", "package name (if not set, the basename of the <target-package-path> will be used)")
	cli.BoolVar(&disableCompression, "no-compression", false, "disable compression even for compressible files")
	cli.BoolVar(&disableHTTPHandler, "no-http-handler", false, "disable http handler API")
	cli.BoolVar(&enableFS, "fs", false, "enable embedded filesystem API")
	cli.BoolVar(&enableUnionFS, "union-fs", false, "enable union filesystem API (real fs over embedded, implies -fs)")
	cli.BoolVar(&enableHTTPFS, "http-fs", false, "enable http.FileSystem API (implies -fs")
	cli.BoolVar(&enableRawBytes, "raw-bytes", false, "enable raw bytes access API")
	cli.BoolVar(&makeBinary, "binary", false, "produce self-contained http server binary (<target-package-path> will become the binary name then)")
	mimeTypes := [][2]string{
		{".go", "text/x-golang"}, // Golang extension is due to get into apache /etc/mime.types
	}
	for i := range mimeTypes {
		mime.AddExtensionType(mimeTypes[i][0], mimeTypes[i][1])
	}
}

func main() {
	err := cli.Parse(os.Args[1:])
	if err != nil || cli.NArg() != 2 {
		var opts bytes.Buffer
		cli.SetOutput(&opts)
		cli.PrintDefaults()
		usage.Execute(os.Stdout, map[string]string{
			"Binary":  os.Args[0],
			"Options": opts.String(),
		})
		os.Exit(1)
	}
	source := cli.Arg(0)
	target := cli.Arg(1)
	if err = do(source, target); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func do(source, target string) error {
	var (
		targetDir string
		buildDir string
		flags imbed.ImbedFlag
		err error
	)
	if makeBinary {
		buildDir, err = ioutil.TempDir(os.TempDir(), ".go-imbed")
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		defer rmtree(buildDir)
		targetDir = filepath.Join(buildDir, "src", "main")
		pkgName = "main"
		flags = imbed.BuildFsAPI | imbed.BuildHttpHandlerAPI | imbed.CompressAssets
	} else {
		targetDir = target
		if pkgName == "" {
			pkgName = filepath.Base(target)
		}
		flags  = imbed.ImbedFlag(0).Set(imbed.CompressAssets, !disableCompression).
			Set(imbed.BuildHttpHandlerAPI, !disableHTTPHandler).
			Set(imbed.BuildFsAPI, enableFS).
			Set(imbed.BuildHttpFsAPI, enableHTTPFS).
			Set(imbed.BuildUnionFsAPI, enableUnionFS).
			Set(imbed.BuildRawBytesAPI, enableRawBytes)
	}
	err = imbed.Imbed(source, targetDir, pkgName, flags)
	if err != nil {
		return err
	}
	if makeBinary {
		if err = imbed.CopyBinaryTemplate(targetDir); err != nil {
			return err
		}
		cmd := exec.Command("go", "install", "main")
		cmd.Env = append(os.Environ(), "GOPATH="+buildDir)
		cmd.Dir = buildDir
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			return err
		}
		srcBin, err := os.Open(filepath.Join(buildDir, "bin", "main"))
		if err != nil {
			return err
		}
		srcBinStat, err := srcBin.Stat()
		if err != nil {
			return err
		}
		defer srcBin.Close()
		dstBin, err := os.OpenFile(target, os.O_CREATE | os.O_WRONLY, srcBinStat.Mode())
		if err != nil {
			return err
		}
		defer dstBin.Close()
		_, err = io.Copy(dstBin, srcBin)
		if err != nil {
			return err
		}
	}
	return nil
}

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