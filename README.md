# go-imbed

[![Build Status](https://travis-ci.org/growler/go-imbed.svg?branch=master)](https://travis-ci.org/growler/go-imbed)

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
- provides [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc) handler
  (unless requested otherwise),
- provides [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem) API
  (if requested),
- provides a simple FileSystem abstraction (if requested),
- provides a union FileSystem abstraction with a real file system directory
  overlaid embedded (if requested),
- generates a test code as well to keep those having OCD regarding test coverage happy. 


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

```bash
go-imbed [options] <source-content-path> <target-package-path>
```

### `-pkg`

Sets the resulting package name. If not present, the base name (i.e. last item) of the `target-package-path` 
will be used. 

### `-no-compresssion`

`go-imbed` compresses all the text resources with [gzip](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Encoding#Directives).
Supplied HTTP helper function will decompress resource if HTTP client does not
support compression. `-no-compression` disables compression for all files.

### `-no-http-handler`

`-no-http-handler` disables generation of [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc) 
API.

### `-fs`

`-fs` generates a virtual filesystem API similar to [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem).

### `-union-fs`

`-union-fs` generates union filesystem API with a real file system directory overlaid
embedded filesystem (implies `-fs`)

### `-http-fs`

`-http-fs` generates [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem) 
interface (implies `-fs`).

### `-raw-bytes`

`-raw-bytes` enables direct access to stored binary asset as a []byte slice. Please note
that changing data will result in segmentation fault.

### `-binary`

`-binary` produces an executable image with embedded content instead of a source package. The image
could serve as a self-extracting archive or self-contained HTTP server:

```bash
$ ./site --help
Usage of ./site:
  -extract directory
    	extract content to the target directory and exit
  -listen address
    	socket address to listen (default ":8080")
  -tls-cert file
    	TLS certificate file to use
  -tls-key file
    	TLS key file to use
```

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

### Asset.Reader

```go
func (*Asset) Reader() io.ReaderCloser
```

Returns an io.ReaderCloser interface to read asset data.

### Asset os.FileInfo interface

```go
func (*Asset) Size() int64
func (*Asset) ModTime() time.Time
func (*Asset) Mode() os.FileMode
func (*Asset) Sys() interface{}
func (*Asset) IsDir() bool
```

These functions implement [os.FileInfo](https://golang.org/pkg/os/#FileInfo) interface.
Note that `Size()` returns real (uncompressed) size of the asset.

### Asset.String

```go
func (*Asset) String() string
```

Returns asset content as `string`. If asset was not compressed, then
string will hold a direct pointer to RO section of the binary, otherwise `String()`
will return uncompressed asset.

### Asset.Bytes

```go
func (*Asset) Bytes() []byte
```

Returns asset content as `[]byte`, uncompressing it if necessary. Note that
`Bytes()` will return a copy of the embedded content even if it was not compressed.
To get a direct reference to RO data, use [Asset.RawBytes](#asset.rawbytes)

### Asset.RawBytes

```go
func (*Asset) RawBytes() []byte
```

Present only if `-raw-bytes` option was enabled and returns direct pointer to RO section
of the binary. Any write to the slice will result in segmentation fault.

### Asset.WriteTo

```go
func (*Asset) WriteTo(io.Writer) (int64, error)
```

Writes full content of the asset to supplied `io.Writer`, decompressing asset content if 
necessary.  

### FileSystem

```go
type FileSystem interface {
	Open(name string) (File, error)
	Stat(name string) (os.FileInfo, error)
	Walk(root string, walkFunc filepath.WalkFunc) error
	HttpFileSystem() http.FileSystem
}
```

Virtual filesystem abstraction, present only if one of `-*fs` options were enabled. 
`Walk` methods behave the same way as [filepath.Walk](https://golang.org/pkg/path/filepath/#Walk).
`HttpFileSystem()` method present only if `-http-fs` option was enabled and returns [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem)
interface to serve content with standard http server (but take a look at builtin [http handler](#httphandlerwithprefix) first).

### File

```go
type File interface {
	io.Closer
	io.Reader
	io.Seeker
	Readdir(count int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
}
```

Virtual filesystem File interface. Methods behave similar to *os.File methods

### Open

```go
func Open(name string) (io.ReadCloser, error)

func Open(name string) (File, error)
```

`Open` returns [io.ReadCloser](https://golang.org/pkg/io/#ReadCloser) or [File](#file) if `-*fs` option
was set, to read asset content from. If no asset was found, [os.ErrNotExist](https://golang.org/pkg/os/#ErrNotExist)
will be returned.

Note that with virtual filesystem enabled it is possible to open directories and list assets with Readdir. 

### CopyTo

```go
func CopyTo(target string, mode os.FileMode, overwrite bool, files ...string) error 
```

The CopyTo method extracts all mentioned files to a specified location, keeping directory structure.
If supplied file is a directory, than it will be extracted recursively. CopyTo with no file mentioned 
will extract the whole content of the embedded filesystem. CopyTo returns error if there is a file with 
the same name at the target location, unless overwrite is set to true, or file has the same size and 
modification file as the extracted file.

Following code 

```go
pkg.CopyTo(".", 0640, false)
```

will effectively extract content of the filesystem to the current directory (which
makes it the most space-wise inefficient self-extracting archive ever).

### NewUnionFs

```go
func NewUnionFs(path string) (FileSystem, error)
```

Present only if `-union-fs` option was enabled and returns a union fs, a real file system 
directory starting `path`, which overlaid embedded filesystem.

### HttpFileSystem

```go
func HttpFileSystem() http.FileSystem
```

Present only if `-http-fs` option was enabled. Returns assets directory as [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem).
A convenience shortcut for `FS().HttpFileSystem()` 

### HTTPHandlerWithPrefix

```go
func HTTPHandlerWithPrefix(prefix string) func(w http.ResponseWriter, req *http.Request)
```

Present only unless `-no-http-handler` option was set. 
`HTTPHandlerWithPrefix` provides a simple way to serve embedded content via
Go standard HTTP server and returns an http handler function. The `prefix`
will be stripped from the request URL to serve embedded content from non-root URI.
Note that handler sends already compressed content if client supports compression, and 
also it sends `Etag` with precomputed asset hash and supports conditional requests
with `If-None-Match`, which makes it more efficient than `http.FileSystem`
API in most real life cases.

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
- Once again, `asset.RawBytes` points directly to data, located in read-only data section
  of the executable image, so any attempt to modify it will result in page
  protection fault. Hopefully, Go will have [read-only](https://docs.google.com/document/d/1-NzIYu0qnnsshMBpMPmuO21qd8unlimHgKjRD9qwp2A)
  [slices](https://github.com/golang/go/issues/20443) one day, so this will be no longer an issue.
- UnionFs abstraction do not allow to _delete_ file, only to add or replace content to the embedded
  filesystem.
- Even a minor change in one of the resources will result in totally new `data.s` file, which feels
  a bit inconvenient from VCS point of view.  

## License

The MIT License, see [LICENSE.md](LICENSE.md).

## Other similar tools

- [go.rice](https://github.com/GeertJohan/go.rice)
- [go-bindata](https://github.com/jteeuwen/go-bindata)
- [statik](https://github.com/rakyll/statik)
- [esc](https://github.com/mjibson/esc)
- [packr](https://github.com/gobuffalo/packr)
