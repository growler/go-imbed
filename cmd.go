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
)

var usage = template.Must(template.New("").Parse(
	`A simple source generator to embed resources into Go executable

Usage:
    {{.Binary}} [options] <source-content-path> <target-package>

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

var disableCompression bool
var disableHTTPHelper bool
var enableFS bool
var enableHTTPFS bool
var enableBytes bool

func init() {

	cli = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	cli.BoolVar(&disableCompression, "no-compression", false, "disable compression even for compressible files")
	cli.BoolVar(&disableHTTPHelper, "no-http-helper", false, "disable http helper API")
	cli.BoolVar(&enableFS, "filesystem", false, "enable virtual filesystem API")
	cli.BoolVar(&enableHTTPFS, "http-filesystem", false, "enable http.FileSystem API (implies -filesystem")
	cli.BoolVar(&enableBytes, "raw-data", false, "enable raw data access API")
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
	params := imbed.ImbedParams{
		CompressAssets: !disableCompression,
		BuildHttpHelperAPI: !disableHTTPHelper,
		BuildFsAPI: enableFS,
		BuildHttpFsAPI: enableHTTPFS,
		BuildRawAccessAPI: enableBytes,
	}

	err = imbed.Imbed(source, target, params)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
