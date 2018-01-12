package main

import (
	"flag"
	"fmt"
	"os"
	"net/http"
)

var (
	listenAddr string
	cert       string
	key        string
	extract    string
)

func init() {
	flag.StringVar(&extract, "extract", "", "extract content to the target `directory` and exit")
	flag.StringVar(&listenAddr, "listen", ":8080", "socket `address` to listen")
	flag.StringVar(&cert, "tls-cert", "", "TLS certificate `file` to use")
	flag.StringVar(&key, "tls-key", "", "TLS key `file` to use")
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
	if extract != "" {
		if err = CopyTo(extract, 0640, false); err != nil {
			fmt.Fprintf(os.Stderr, "error extracting content: %s\n", err)
			os.Exit(1)
		}
		return
	}
	http.HandleFunc("/", HTTPHandlerWithPrefix("/"))
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