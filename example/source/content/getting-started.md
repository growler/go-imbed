---
title: "Getting Started"
anchor: "getting-started"
weight: 20
---
## Installation

```text
$ go get -u github.com/growler/go-imbed
```

## Usage

1. Install `go-imbed`:

    ```
    go get -u github.com/growler/go-imbed

    ```
2. Add a static content tree to target package:

    ```
    src
    └── yourpackage
        ├── code.go
        └── site
            ├── static
            │   └── style.css
            ├── index.html
            └── 404.html
    ```

3. Add a go-generate comment to `code.go` (or any other Go file in `yourpackage`):

    {{< highlight go >}}
    //go:generate go-imbed site internal/site
    
    package ...
    {{< /highlight >}}

4. Run `go generate yourpackage`

5. Start using it:

    {{< highlight go >}}
    package main 

    import (
        "net/http"
        "fmt"
        "yourpackage/internal/site"
    )

    func main() {
    	http.HandleFunc("/", site.ServeHTTP)
    	if err := http.ListenAndServe(":9091", nil); err != nil {
    		fmt.Println(err)   
 		}
    }
    {{< /highlight >}}
