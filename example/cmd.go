//go:generate go run ../cmd.go --http-filesystem site internal/site

package main

import (
	"net/http"
	"fmt"
	"github.com/growler/go-imbed/example/internal/site"
	"flag"
	"os"
)

var (
	listenAddr string
	cert string
	key  string
)

func init() {
	flag.StringVar(&listenAddr, "listen", ":8080", "socket address to listen")
	flag.StringVar(&cert, "cert", "", "TLS certificate file to use")
	flag.StringVar(&key, "key", "", "TLS key file to use")
}

func main() {
	var tls bool
	var err error
	flag.Parse()
	if cert != "" && key != "" {
		tls = true
	} else if cert != "" || key != "" {
		fmt.Fprintln(os.Stderr, "both cert and key must be supplied for HTTPS")
		os.Exit(1)
	}
	http.Handle("/", http.FileServer(site.HttpFileSystem()))
	http.HandleFunc("/site/", site.HTTPHandlerWithPrefix("/site/"))
	if tls {
		err = http.ListenAndServeTLS(listenAddr, cert, key, nil)
	} else {
		err = http.ListenAndServe(listenAddr, nil)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}