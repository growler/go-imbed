# go-imbed

go-imbed is a simple tool for embedding binary assets into Go executable.

## Why

`go-imbed` came up as a holiday side project for a very simple case of embedding 
a REST API documentation into the executable image. There are
[plenty of tools](#other-similar-tools) for embedding binary assets into executable 
(which clearly shows demand for something what Go lacks at the moment), but hey,
why not invent another wheel? Besides, stuffing binary asset into Go
source file seemed too unaesthetic to me.  

`go-imbed`:

- produces go-gettable go and go assembly sources with `go generate`,
- keeps data in read-only section of the binary,
- compress compressible files with `gzip`,
- provides [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc) helper
  (unless requested otherwise),
- provides [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem) API
  (if requested).

## License

The MIT License, see [LICENSE.md](LICENSE.md).

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
   ```go
   //go:generate go-imbed site internal/site
   ```
4. Run `go generate yourpackage`
5. Start using it:
   ```go
    package main

    import (
        "net/http"
        "fmt"
        "yourpackage/internal/site"
    )

    func main() {
        http.HandleFunc("/", site.ServeHTTP)
        if err := http.ListenAndServe(":9091", nil); err != nil{
            fmt.Println(err)
        }
    }

    ```

## Options

### `-no-compresssion`

`go-imbed` compresses all the text resources with [gzip](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Encoding#Directives).
Supplied HTTP helper function will decompress resource if HTTP client does not
support compression. `-no-compression` disables compression for all files.

### `-no-http-helper`

`-no-http-helper` disables generation of [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc) 
helper function.

### `-filesystem`

`-filesystem` generates a virtual filesystem API similar to [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem).
API includes [CopyTo](#filecopyto) method to extract single asset or a directory to a filesystem path (which makes `go-imbed`
the most space-wise inefficient self-extracting archiver ever).

### `-http-filesystem`

`-http-filesystem` generates [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem) 
interface (implies `-filesystem`)

### `-raw-data`

`-raw-data` enables direct access to stored binary asset as a []byte slice. Please note
that changing data will result in segmentation fault.

## Generated code API

### Asset

```go
type Asset struct {
    // contains filtered or unexported fields
}
```

Asset represents binary resource stored within Go executable. Asset implements
[fmt.Stringer](https://golang.org/pkg/fmt/#Stringer) and 
[io.WriterTo](https://golang.org/pkg/io/#WriterTo) interfaces, decompressing 
binary data if necessary.

### Asset.Name

```go
func (*Asset) Name() string
```

Returns base file name of the asset.

### Asset.MimeType

```go
func (*Asset) MimeType() string
```

Returns MIME Type (computed from the file extension during compilation) of the asset.

### Asset.IsCompressed

```go
func (*Asset) IsCompressed() bool
```

Returns true if resource has been compressed. Present only if compression was not disabled.

### Asset os.FileInfo interface

```go
func (*Asset) Size() int64
func (*Asset) ModTime() time.Time
func (*Asset) Mode() os.FileMode
func (*Asset) Sys() interface{}
func (*Asset) IsDir() bool
```

These functions implement [os.FileInfo](https://golang.org/pkg/os/#FileInfo) interface.
Please note that `Size()` returns real (uncompressed) size of the asset.

### Asset.String

```go
func (*Asset) String() string
```

Returns string representation of the asset. If asset was not compressed, then
string will hold a direct pointer to RO section of the binary, otherwise `String()`
will return uncompressed asset.

### Asset.WriteTo

```go
func (*Asset) WriteTo(io.Writer) (int64, error)
```

Writes full content of the asset to supplied `io.Writer`, decompressing asset content if 
necessary.  

### Asset.CopyTo

```go
func (a *Asset) CopyTo(target string, mode os.FileMode, overwrite bool) error
```

CopyTo method copies asset content to file in the target directory with
the `mode` permissions.  If file with the same name, size and modification 
time exists, it will not be overwritten, unless overwrite = true is specified.

#### Asset.Bytes

```go
func (*Asset) Bytes() []byte
```

Present only if `-raw-data` option was enabled and returns direct pointer to RO section
of the binary. Any write to the slice will result in segmentation fault.

### File

```go
type File interface {
	io.Closer
	io.Reader
	io.Seeker
	Readdir(count int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
	CopyTo(target string, mode os.FileMode, overwrite bool) error
}
```

Virtual filesystem File interface. Methods behave similar to *os.File methods

### File.CopyTo

```go
func (File) CopyTo(target string, mode os.FileMode, overwrite bool) error
```

The CopyTo method copies asset or assets directory content to the 
target path, creating files and directories if necessary with the `mode`
permissions. CopyTo will not overwrite files with the same name, size and 
modification time unless `overwrite` is set to true.

Following code

```go
pkg.Open("/").CopyTo(".", 0644, false)
```

will extract contents of the virtual embedded filesystem into the current directory.

### Get

```go
func Get(name string) *Asset
```

`Get` returns pointer to Asset structure, or nil if no asset found. Asset name
should not contain leading slash, i.e. `css/style.css`, not `/css/style.css`.

### Must

```go
func Must(name string) *Asset
```

`Must` returns pointer to Asset structure, or panics if no asset found.

### Open

```go
func Open(name string) (io.ReadCloser, error)

func Open(name string) (File, error)
```

`Open` returns [io.ReadCloser](https://golang.org/pkg/io/#ReadCloser) or [File](#file) if `-filesystem` option
was set, to read asset content from. If no asset was found, [os.ErrNotExist](https://golang.org/pkg/os/#ErrNotExist)
will be returned.

Note that with virtual filesystem enabled it is possible to open directories and list assets with Readdir. 

### HttpFileSystem

Present only if `-http-filesystem` option was enabled. Returns assets directory as [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem).

### HTTPHandlerWithPrefix

```go
func HTTPHandlerWithPrefix(prefix string) func(w http.ResponseWriter, req *http.Request)
```

Present only unless `-no-http-helper` option was set. 
`HTTPHandlerWithPrefix` provides a simple way to serve embedded content via
Go standard HTTP server and returns an http handler function. The `prefix`
will be stripped from the request URL to serve embedded content from non-root URI.
Please note that `HTTPHandlerWithPrefix` will send compressed content if client 
supports compression, so in most cases it will be more efficient than `http.FileSystem`
API.

```go
func main() {
     ...
     http.HandleFunc("/api/help/", site.HTTPHandlerWithPrefix("/api/help/"))
     ...
     http.ListenAndServe(address, nil)
     ...
}
```

If the source tree had `404.html` file, handler function will employ it in
case of absent resource, otherwise a standard Go `http.NotFound` response will
be used.

### ServeHTTP

```go
var ServeHTTP = HTTPHandlerWithPrefix("/")
```

ServeHTTP provides a convenience handler whenever embedded content should be served from the root URI.

## Caveats

- Tested well only for amd64 and 386. Other architectures should work, though.
- Once again, `asset.Bytes` points directly to data, located in read-only data section
  of the executable image, so any attempt to modify it will result in page
  protection fault. Hopefully, Go will have [read-only](https://docs.google.com/document/d/1-NzIYu0qnnsshMBpMPmuO21qd8unlimHgKjRD9qwp2A)
  [slices](https://github.com/golang/go/issues/20443) one day, so this will be no longer an issue.
- Even a minor change in one of the resources will result in totally new `data.s` file, which feels
  a bit inconvenient from VCS point of view.

## Other similar tools

- [go.rice](https://github.com/GeertJohan/go.rice)
- [go-bindata](https://github.com/jteeuwen/go-bindata)
- [statik](https://github.com/rakyll/statik)
- [esc](https://github.com/mjibson/esc)
- [packr](https://github.com/gobuffalo/packr)
