// Code generated by go-imbed. DO NOT EDIT.

// Package {{.Pkg}} holds binary resources embedded into Go executable
package {{.Pkg}}

import (
	"os"
	"io"
	"bytes"
	"path/filepath"
{{- if .Params.BuildHttpHandlerAPI }}
	"strconv"
{{- end }}
{{- if .Params.BuildFsAPI }}
	"sort"
{{- end }}
{{- if or .Params.BuildHttpFsAPI .Params.BuildHttpHandlerAPI }}
	"net/http"
{{- end }}
{{- if or .Params.BuildHttpHandlerAPI .Params.BuildFsAPI }}
	"path"
{{- end }}
{{- if .Params.BuildHttpHandlerAPI }}
	"strings"
{{- end }}
{{- if .Params.CompressAssets }}
	"compress/gzip"
{{- end }}
{{- if or .Params.BuildFsAPI .Params.CompressAssets }}
	"io/ioutil"
{{- end }}
{{- if .Params.BuildMain }}
	"flag"
{{- end }}
	"time"
)

func blob_bytes(uint32) []byte
func blob_string(uint32) string

// Asset represents binary resource stored within Go executable. Asset implements
// fmt.Stringer and io.WriterTo interfaces, decompressing binary data if necessary.
type Asset struct {
	name         string // File name
	size         int32  // File size (uncompressed)
	blob         []byte // Resource blob []byte
	str_blob     string // Resource blob as a string
{{- if .Params.CompressAssets }}
	isCompressed bool   // true if resources was compressed with gzip
{{- end}}
	mime         string // MIME Type
	tag          string // Tag is essentially a Tag of resource content and can be used as a value for "Etag" HTTP header
}

// Name returns the base name of the asset
func (a *Asset) Name() string       { return a.name }
// MimeType returns MIME Type of the asset
func (a *Asset) MimeType() string   { return a.mime }
// Tag returns a string which can serve as an unique version identifier for the asset (i.e., "Etag")
func (a *Asset) Tag() string        { return a.tag  }
{{- if .Params.CompressAssets }}
// IsCompressed returns true of asset has been compressed
func (a *Asset) IsCompressed() bool { return a.isCompressed }
{{- end }}
// String returns (uncompressed, if necessary) content of asset as a string
func (a *Asset) String() string {
{{- if .Params.CompressAssets }}
	if a.isCompressed {
		ungzip, _ := gzip.NewReader(bytes.NewReader(a.blob))
		ret, _ := ioutil.ReadAll(ungzip)
		ungzip.Close()
		return string(ret)
	}
{{- end }}
	return a.str_blob
}

// Bytes returns (uncompressed) content of asset as a []byte
func (a *Asset) Bytes() []byte {
{{- if .Params.CompressAssets }}
	if a.isCompressed {
		ungzip, _ := gzip.NewReader(bytes.NewReader(a.blob))
		ret, _ := ioutil.ReadAll(ungzip)
		ungzip.Close()
		return ret
	}
{{- end }}
	ret := make([]byte, len(a.blob))
	copy(ret, a.blob)
	return ret
}
{{- if .Params.BuildRawBytesAPI }}
// RawBytes returns a raw byte slice of the asset. Changing content of slice will result into segfault.
func (a *Asset) RawBytes() []byte {
	return a.blob
}
{{- end }}

// Size implements os.FileInfo and returns the size of the asset (uncompressed, if asset has been compressed)
func (a *Asset) Size() int64        { return int64(a.size) }
// Mode implements os.FileInfo and always returns 0444
func (a *Asset) Mode() os.FileMode  { return 0444 }
// ModTime implements os.FileInfo and returns the time stamp when this package has been produced (the same value for all the assets)
func (a *Asset) ModTime() time.Time { return stamp }
// IsDir implements os.FileInfo and returns false
func (a *Asset) IsDir() bool        { return false }
// Sys implements os.FileInfo and returns nil
func (a *Asset) Sys() interface{}   { return a }

// WriteTo implements io.WriterTo interface and writes content of the asset to w
func (a *Asset) WriteTo(w io.Writer) (int64, error) {
{{- if .Params.CompressAssets }}
	if a.isCompressed {
		ungzip, _ := gzip.NewReader(bytes.NewReader(a.blob))
		n, err := io.Copy(w, ungzip)
		ungzip.Close()
		return n, err
	}
{{- end }}
	n, err := w.Write(a.blob)
	return int64(n), err
}

type assetReader struct {
	bytes.Reader
}

func (r *assetReader) Close() error {
	r.Reset(nil)
	return nil
}

// Returns content of the asset as io.ReaderCloser.
func (a *Asset) Reader() io.ReadCloser {
{{- if .Params.CompressAssets }}
	if a.isCompressed {
		ungzip, _ := gzip.NewReader(bytes.NewReader(a.blob))
		return ungzip
	} else {
{{- end }}
		ret := &assetReader{}
		ret.Reset(a.blob)
		return ret
{{- if .Params.CompressAssets }}
	}
{{- end }}
}

func cleanPath(path string) string {
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		path = path[len(filepath.VolumeName(path)):]
		if len(path) > 0 || os.IsPathSeparator(path[0]) {
			path = path[1:]
		}
	} else if path == "." {
		return ""
	}
	return filepath.ToSlash(path)
}

// Opens asset as an io.ReadCloser. Returns os.ErrNotExist if no asset is found.
{{- if .Params.BuildFsAPI }}
func Open(name string) (File, error) {
	return FS().Open(name)
}
{{- else }}
func Open(name string) (io.ReadCloser, error) {
	name = cleanPath(name)
	if asset, ok := fidx[name]; !ok {
		return nil, os.ErrNotExist
	} else {
		return asset.Reader(), nil
	}
}
{{- end }}

// Gets asset by name. Returns nil if no asset found.
func Get(name string) *Asset {
	if entry, ok := fidx[name]; ok {
		return entry
	} else {
		return nil
	}
}

// Get asset by name. Panics if no asset found.
func Must(name string) *Asset {
	if entry, ok := fidx[name]; ok {
		return entry
	} else {
		panic("asset " + name + " not found")
	}
}

type directoryAsset struct {
	name  string
	dirs  []directoryAsset
	files []Asset
}

var root *directoryAsset

{{- if or .Params.BuildFsAPI }}

// A simple FileSystem abstraction
type FileSystem interface {
	Open(name string) (File, error)
	Stat(name string) (os.FileInfo, error)
	// As in filepath.Walk
	Walk(root string, walkFunc filepath.WalkFunc) error
{{- if .Params.BuildHttpFsAPI }}
    // Returns http.FileSystem interface to use with http.Server
	HttpFileSystem() http.FileSystem
{{- end }}
}

// The CopyTo method extracts all mentioned files
// to a specified location, keeping directory structure.
// If supplied file is a directory, than it will be extracted
// recursively. CopyTo with no file mentioned will extract
// the whole content of the embedded filesystem.
// CopyTo returns error if there is a file with the same name
// at the target location, unless overwrite is set to true, or
// file has the same size and modification file as the extracted
// file.
// {{.Pkg}}.CopyTo(".", mode, false) will effectively
// extract content of the filesystem to the current directory (which
// makes it the most space-wise inefficient self-extracting archive
// ever).
func CopyTo(target string, mode os.FileMode, overwrite bool, files ...string) error {
	mode    =  mode&0777
	dirmode := os.ModeDir|((mode&0444)>>2)|mode
	if len(files) == 0 {
		files = []string{""}
	}
	for _, file := range files {
		file = cleanPath(file)
		err := FS().Walk(file, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			targetPath := filepath.Join(target, path)
			fi, err := os.Stat(targetPath)
			if err == nil {
				if info.IsDir() && fi.IsDir() {
					return nil
				} else if info.IsDir() != fi.IsDir() {
					return os.ErrExist
				} else if !overwrite {
					if info.Size() == fi.Size() && info.ModTime().Equal(fi.ModTime()) {
						return nil
					} else {
						return os.ErrExist
					}
				}
			}
			if info.IsDir() {
				return os.MkdirAll(targetPath, dirmode)
			}
			asset := Get(path)
			if asset == nil {
				return os.ErrNotExist
			}
			targetPathDir := filepath.Dir(targetPath)
			if err = os.MkdirAll(targetPathDir, dirmode); err != nil {
				return err
			}
			dst, err := ioutil.TempFile(targetPathDir, ".imbed")
			if err != nil {
				return err
			}
			defer func() {
				dst.Close()
				os.Remove(dst.Name())
			}()
			_, err = asset.WriteTo(dst)
			if err != nil {
				return err
			}
			dst.Close()
			os.Chtimes(dst.Name(), info.ModTime(), info.ModTime())
			os.Chmod(dst.Name(), mode)
			return os.Rename(dst.Name(), targetPath)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type fileInfoSlice []os.FileInfo
func (fis *fileInfoSlice) Len() int           { return len(*fis) }
func (fis *fileInfoSlice) Less(i, j int) bool { return (*fis)[i].Name() < (*fis)[j].Name() }
func (fis *fileInfoSlice) Swap(i, j int) {
	s := (*fis)[i]
	(*fis)[i] = (*fis)[j]
	(*fis)[j] = s
}

func walkRec(fs FileSystem, info os.FileInfo, p string, walkFn filepath.WalkFunc) error {
	var (
		dir File
		fis fileInfoSlice
		err error
	)
	err = walkFn(p, info, nil)
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	dir, err = fs.Open(p)
	if err != nil {
		return walkFn(p, info, err)
	}
	fis, err = dir.Readdir(-1)
	if err != nil {
		return walkFn(p, info, err)
	}
	sort.Sort(&fis)
	for i := range fis {
		fn := path.Join(p, fis[i].Name())
		err = walkRec(fs, fis[i], fn, walkFn)
		if err != nil {
			if !fis[i].IsDir() || err != filepath.SkipDir {
				return err
			}
		}
	}
	return nil
}

func walk(fs FileSystem, name string, walkFunc filepath.WalkFunc) error {
	var r os.FileInfo
	var err error
	name = cleanPath(name)
	r, err = fs.Stat(name)
	if err != nil {
		return err
	}
	return walkRec(fs, r, name, walkFunc)
}

type assetFs struct{}

// Returns embedded FileSystem
func FS() FileSystem {
	return &assetFs{}
}

func (fs *assetFs) Walk(root string, walkFunc filepath.WalkFunc) error {
	return walk(fs, root, walkFunc)
}

func (fs *assetFs) Stat(name string) (os.FileInfo, error) {
	name = cleanPath(name)
	if name == "" {
		return root, nil
	}
	if dir, ok := didx[name]; ok {
		return dir, nil
	}
	if asset, ok := fidx[name]; ok {
		return asset, nil
	}
	return nil, os.ErrNotExist
}

func (fs *assetFs) Open(name string) (File, error) {
	name = cleanPath(name)
	if name == "" {
		return root.open(""), nil
	}
	if dir, ok := didx[name]; ok {
		return dir.open(name), nil
	}
	if asset, ok := fidx[name]; ok {
		return asset.open(name), nil
	}
	return nil, os.ErrNotExist
}

{{- if .Params.BuildHttpFsAPI }}
func (fs *assetFs) HttpFileSystem() http.FileSystem {
	return &httpFileSystem{fs: fs}
}
{{- end }}


// A File is returned by virtual FileSystem's Open method.
// The methods should behave the same as those on an *os.File.
type File interface {
	io.Closer
	io.Reader
	io.Seeker
	Readdir(count int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
}

func (a *Asset) open(name string) File {
{{- if .Params.CompressAssets }}
	if a.isCompressed {
		ret := &assetCompressedFile{
			asset: a,
			name:  name,
		}
		ret.Reset(bytes.NewReader(a.blob))
		return ret
	} else {
{{- end }}
		ret := &assetFile{
			asset: a,
			name:  name,
		}
		ret.Reset(a.blob)
		return ret
{{- if .Params.CompressAssets }}
	}
{{- end }}
}

func (d *directoryAsset) open(name string) File {
	return &directoryAssetFile{
		dir:  d,
		name: name,
		pos:  0,
	}
}

type directoryAssetFile struct {
	dir  *directoryAsset
	name string
	pos  int
}

func (d *directoryAssetFile) Name() string {
	return d.name
}

func (d *directoryAssetFile) checkClosed() error {
	if d.pos < 0 {
		return os.ErrClosed
	}
	return nil
}

func (d *directoryAssetFile) Close() error {
	if err := d.checkClosed(); err != nil {
		return err
	}
	d.pos = -1
	return nil
}

func (d *directoryAssetFile) Read([]byte) (int, error) {
	if err := d.checkClosed(); err != nil {
		return 0, err
	}
	return 0, io.EOF
}

func (d *directoryAssetFile) Stat() (os.FileInfo, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}
	return d.dir, nil
}

func (d *directoryAssetFile) Seek(pos int64, whence int) (int64, error) {
	if err := d.checkClosed(); err != nil {
		return 0, err
	}
	return 0, os.ErrInvalid
}

func (d *directoryAssetFile) Readdir(count int) ([]os.FileInfo, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}
	var (
		last int
		total = len(d.dir.dirs) + len(d.dir.files)
	)
	if d.pos > total {
		if count > 0 {
			return nil, io.EOF
		} else {
			return nil, nil
		}
	}
	if count <= 0 || (d.pos + count) <= total {
		last = total
	} else {
		last = d.pos + count
	}
	ret := make([]os.FileInfo, 0, last - d.pos)
	if d.pos < len(d.dir.dirs) {
		var stop int
		if last > len(d.dir.dirs) {
			stop = len(d.dir.dirs)
		} else {
			stop = last
		}
		for i := d.pos; i < stop; i++ {
			ret = append(ret, &d.dir.dirs[i])
		}
		d.pos = stop
	}
	var start, stop int
	start = d.pos - len(d.dir.dirs)
	stop = last - len(d.dir.dirs)
	for i := start; i < stop; i++ {
		ret = append(ret, &d.dir.files[i])
	}
	d.pos = last
	return ret, nil
}

func (d *directoryAsset) Name() string       { return d.name }
func (d *directoryAsset) Size() int64        { return 0 }
func (d *directoryAsset) Mode() os.FileMode  { return os.ModeDir | 0555 }
func (d *directoryAsset) ModTime() time.Time { return stamp }
func (d *directoryAsset) IsDir() bool        { return true }
func (d *directoryAsset) Sys() interface{}   { return d }

type assetFile struct {
	assetReader
	name string
	asset *Asset
}

func (a *assetFile) Name() string {
	return a.name
}

func (a *assetFile) Stat() (os.FileInfo, error) {
	return a.asset, nil
}

func (a *assetFile) Readdir(int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

{{- if .Params.CompressAssets }}
type assetCompressedFile struct {
	gzip.Reader
	name  string
	asset *Asset
}

func (a *assetCompressedFile) Name() string {
	return a.name
}

func (a *assetCompressedFile) Stat() (os.FileInfo, error) {
	return a.asset, nil
}

func (a *assetCompressedFile) Seek(int64, int) (int64, error) {
	return 0, os.ErrInvalid
}

func (a *assetCompressedFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

{{- end }}
{{- end }}

{{- if .Params.BuildUnionFsAPI }}

type unionFs struct {
	root string
}

func NewUnionFS(src string) (FileSystem, error) {
	abs, err := filepath.Abs(src)
	if err != nil {
		return nil, err
	}
	return &unionFs{
		root: abs,
	}, nil
}

func (fs *unionFs) Stat(name string) (os.FileInfo, error) {
	name = cleanPath(name)
	fname := filepath.Join(fs.root, filepath.FromSlash(name))
	fi, err := os.Stat(fname)
	if err == nil {
		return fi, nil
	}
	return FS().Stat(name)
}

func (fs *unionFs) Open(name string) (File, error) {
	name = cleanPath(name)
	fname := filepath.Join(fs.root, filepath.FromSlash(name))
	fi, err := os.Stat(fname)
	if err == nil {
		file, err := os.OpenFile(fname, os.O_RDONLY, 0)
		if err == nil {
			if !fi.IsDir() {
				return &unionFsFile{
					name: name,
					file: file,
				}, nil
			} else {
				dir, _ := didx[name]
				return &unionFsDirectoryFile{
					name:  name,
					dir:   dir,
					fsDir: file,
					pos:   0,
				}, nil
			}
		}
	}
	return FS().Open(name)
}

func (fs *unionFs) Walk(root string, walkFunc filepath.WalkFunc) error {
	return walk(fs, root, walkFunc)
}

{{- if .Params.BuildHttpFsAPI }}
func (fs *unionFs) HttpFileSystem() http.FileSystem {
	return &httpFileSystem{fs: fs}
}
{{- end }}

type unionFsFile struct {
	name string
	file *os.File
}

func (f *unionFsFile) Name() string { return f.name }
func (f *unionFsFile) Close() error { return f.file.Close() }
func (f *unionFsFile) Read(d []byte) (int, error) { return f.file.Read(d) }
func (f *unionFsFile) Stat() (os.FileInfo, error) { return f.file.Stat() }
func (f *unionFsFile) Seek(pos int64, whence int) (int64, error) { return f.file.Seek(pos, whence) }
func (f *unionFsFile) Readdir(count int) ([]os.FileInfo, error) { return f.file.Readdir(count) }

type unionFsDirectoryFile struct {
	name  string
	dir   *directoryAsset
	fsDir *os.File
	pos   int
}

func (d *unionFsDirectoryFile) Name() string { return d.name }
func (d *unionFsDirectoryFile) Close() error {
	if d.fsDir == nil {
		return os.ErrClosed
	}
	err := d.fsDir.Close()
	d.fsDir = nil
	return err
}

func (d *unionFsDirectoryFile) Read([]byte) (int, error) {
	if d.fsDir == nil {
		return 0, os.ErrClosed
	}
	return 0, io.EOF
}

func (d *unionFsDirectoryFile) Stat() (os.FileInfo, error) {
	if d.fsDir == nil {
		return nil, os.ErrClosed
	}
	return d.fsDir.Stat()
}

func (d *unionFsDirectoryFile) Seek(pos int64, whence int) (int64, error) {
	if d.fsDir == nil {
		return 0, os.ErrClosed
	}
	return 0, os.ErrInvalid
}
func (d *unionFsDirectoryFile) Readdir(count int) ([]os.FileInfo, error) {
	if d.fsDir == nil {
		return nil, os.ErrClosed
	}
	if d.pos < 0 {
		if count > 0 {
			return nil, io.EOF
		} else {
			return nil, nil
		}
	}
	if d.dir == nil {
		return d.fsDir.Readdir(count)
	}
	ret, err := d.fsDir.Readdir(count)
	if count > 0 && err == nil {
		return ret, err
	}
	embedded := make([]os.FileInfo, 0, len(d.dir.dirs) + len(d.dir.files))
	for i := range d.dir.dirs {
		embedded = append(embedded, &d.dir.dirs[i])
	}
	for i := range d.dir.files {
		embedded = append(embedded, &d.dir.files[i])
	}
	for _, fi := range embedded[d.pos:] {
		if count > 0 && len(ret) >= count {
			return ret, nil
		}
		d.pos++
		if _, err := os.Stat(filepath.Join(d.fsDir.Name(), fi.Name())); err == nil {
			continue
		}
		ret = append(ret, fi)
	}
	d.pos = -1
	return ret, nil
}

{{- end }}

{{- if .Params.BuildHttpFsAPI }}
type httpFileSystem struct {
	fs FileSystem
}
func (fs *httpFileSystem) Open(name string) (http.File, error) {
	return fs.fs.Open(name)
}
func HttpFileSystem() http.FileSystem {
	return FS().HttpFileSystem()
}
{{- end }}

var fidx = make(map[string]*Asset)
var didx = make(map[string]*directoryAsset)
var stamp time.Time

func init() {
	stamp = time.Unix({{.Date}}).UTC()
	bb := blob_bytes({{.Size}})
	bs := blob_string({{.Size}})
{{ .DirectoryCode -}}
{{ .IndexCode -}}
}

{{- if .Params.BuildHttpHandlerAPI }}
{{- if .Has404Asset }}
var http404Asset *Asset
{{- end }}
// ServeHTTP provides a convenience handler whenever embedded content should be served from the root URI.
var ServeHTTP = HTTPHandlerWithPrefix("")

// HTTPHandlerWithPrefix provides a simple way to serve embedded content via
// Go standard HTTP server and returns an http handler function. The "prefix"
// will be stripped from the request URL to serve embedded content from non-root URI
func HTTPHandlerWithPrefix(prefix string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" && req.Method != "HEAD" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(req.URL.Path, prefix) {
			http.NotFound(w, req)
			return
		}
		reqPath := req.URL.Path[len(prefix):]
		if strings.HasPrefix(reqPath, "/") {
			reqPath = reqPath[1:]
		}
		var status = http.StatusOK
		asset, ok := fidx[reqPath]
		if !ok {
			asset, ok = fidx[path.Join(reqPath, "index.html")]
		}
{{- if .Has404Asset }}
		if !ok {
			asset = http404Asset
			status = http.StatusNotFound
		}
{{- else }}
		if !ok {
			http.NotFound(w, req)
			return
		}
{{- end }}
		if tag := req.Header.Get("If-None-Match"); tag != "" {
			if strings.HasPrefix("W/", tag) || strings.HasPrefix("w/", tag) {
				tag = tag[2:]
			}
			if tag, err := strconv.Unquote(tag); err == nil && tag == asset.tag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		if mtime := req.Header.Get("If-Modified-Since"); mtime != "" {
			if ts, err := http.ParseTime(mtime); err == nil && !ts.Before(stamp) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
{{- if .Params.CompressAssets }}
		var deflate = asset.isCompressed
		if encs, ok := req.Header["Accept-Encoding"]; ok {
			for _, enc := range encs {
				if strings.Contains(enc, "gzip") {
					if deflate {
						w.Header().Set("Content-Encoding", "gzip")
					}
					deflate = false
					break
				}
			}
		}
		if !deflate {
			w.Header().Set("Content-Length", strconv.FormatInt(int64(len(asset.blob)), 10))
		}
{{- else }}
		w.Header().Set("Content-Length", strconv.FormatInt(int64(asset.size), 10))
{{- end }}
		w.Header().Set("Content-Type", asset.mime)
		w.Header().Set("Etag", strconv.Quote(asset.tag))
		w.Header().Set("Last-Modified", stamp.Format(http.TimeFormat))
		w.WriteHeader(status)
		if req.Method != "HEAD" {
{{- if .Params.CompressAssets }}
			if deflate {
				ungzip, _ := gzip.NewReader(bytes.NewReader(asset.blob))
				defer ungzip.Close()
				io.Copy(w, ungzip)
			} else {
{{- end }}
				w.Write(asset.blob)
{{- if .Params.CompressAssets }}
			}
{{- end }}
		}
	}
}
{{- end}}

{{- if .Params.BuildMain }}
var (
	listenAddr string
	cert       string
	key        string
	extract    string
	list       bool
	help       bool
)

func init() {
	flag.BoolVar(&help, "help", false, "prints help")
	flag.BoolVar(&list, "list", false, "list contents and exit")
	flag.StringVar(&extract, "extract", "", "extract contents to the target `directory` and exit")
	flag.StringVar(&listenAddr, "listen", ":8080", "socket `address` to listen")
	flag.StringVar(&cert, "tls-cert", "", "TLS certificate `file` to use")
	flag.StringVar(&key, "tls-key", "", "TLS key `file` to use")
}

func main() {
	var tls bool
	var err error
	flag.Parse()
	if help {
		flag.Usage()
		return
	}
	if list {
		FS().Walk("", func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			os.Stdout.WriteString(path)
			os.Stdout.WriteString("\n")
			return nil
		})
		return
	}
	if extract != "" {
		if err = CopyTo(extract, 0640, false); err != nil {
			os.Stderr.WriteString("error extracting content: ")
			os.Stderr.WriteString(err.Error())
			os.Stderr.WriteString("\n")
			os.Exit(1)
		}
		return
	}
	if cert != "" && key != "" {
		tls = true
	} else if cert != "" || key != "" {
		os.Stderr.WriteString("both cert and key must be supplied for HTTPS\n")
		os.Exit(1)
	}
	http.HandleFunc("/", HTTPHandlerWithPrefix("/"))
	if tls {
		err = http.ListenAndServeTLS(listenAddr, cert, key, nil)
	} else {
		err = http.ListenAndServe(listenAddr, nil)
	}
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString("\n")
		os.Exit(1)
	}
}
{{ end }}
