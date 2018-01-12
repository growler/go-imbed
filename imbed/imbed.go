// Copyright 2017 Alexey Naidyonov. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

/*
   Package github.com/growler/go-imbed/imbed
*/
package imbed

//go:generate go run -tags bootstrap ../cmd.go --no-http-handler --fs _templates internal/templates

import (
	"bytes"
	"compress/gzip"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"go/format"
	"hash/crc64"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)


type directoryAsset struct {
	name  string
	dirs  []directoryAsset
	files []*fileAsset
}

type fileAsset struct {
	name         string
	mimeType     string
	tag          string
	size         int64
	isCompressed bool
	offStart     int
	offStop      int
}

func buildIndex(d *directoryAsset, flags ImbedFlag) (string, string, bool) {
	var dir bytes.Buffer
	var index bytes.Buffer
	addIndent(&dir, 1)
	dir.WriteString("root = &directoryAsset")
	addIndent(&index, 1)
	index.WriteString("didx[\"\"] = root\n")
	has404Asset := buildDirIndex(flags, &dir, &index, d, "", "root", 1)
	dir.WriteRune('\n')
	return dir.String(), index.String(), has404Asset
}

func addIndent(buf *bytes.Buffer, n int) {
	for i := 0; i < n; i++ {
		buf.WriteByte('\t')
	}
}

func buildDirIndex(flags ImbedFlag, dir *bytes.Buffer, index *bytes.Buffer, d *directoryAsset, p, indexPrefix string, indent int) bool {
	var has404Asset bool
	dir.WriteString("{\n")
	if d.name != "" {
		addIndent(dir, indent+1)
		fmt.Fprintf(dir, "name: \"%s\",\n", d.name)
	}
	if len(d.dirs) > 0 {
		addIndent(dir, indent+1)
		dir.WriteString("dirs: []directoryAsset{\n")
		for i := range d.dirs {
			addIndent(dir, indent+2)
			addIndent(index, 1)
			fmt.Fprintf(index, "didx[\"%s\"] = &%s.dirs[%d]\n", path.Join(p, d.dirs[i].name), indexPrefix, i)
			buildDirIndex(flags, dir, index, &d.dirs[i], path.Join(p, d.dirs[i].name), fmt.Sprintf("%s.dirs[%d]", indexPrefix, i), indent+2)
			dir.WriteString(",\n")
		}
		addIndent(dir, indent+1)
		dir.WriteString("},\n")
	}
	if len(d.files) > 0 {
		addIndent(dir, indent+1)
		dir.WriteString("files: []Asset{\n")
		for i := range d.files {
			fn := d.files[i].name
			addIndent(dir, indent+2)
			d.files[i].writeDefinition(dir, indent+2, flags)
			dir.WriteString(",\n")
			addIndent(index, 1)
			fmt.Fprintf(index, "fidx[\"%s\"] = &%s.files[%d]\n", path.Join(p, fn), indexPrefix, i)
			if flags.has(BuildHttpHandlerAPI) && p == "" && fn == "404.html" {
				addIndent(index, 1)
				fmt.Fprintf(index, "http404Asset = &%s.files[%d]\n", indexPrefix, i)
				has404Asset = true
			}
		}
		addIndent(dir, indent+1)
		dir.WriteString("},\n")
	}
	addIndent(dir, indent)
	dir.WriteString("}")
	return has404Asset
}

func (f *fileAsset) writeDefinition(w *bytes.Buffer, ind int, flags ImbedFlag) {
	fmt.Fprint(w, "{\n")
	addIndent(w, ind+1)
	fmt.Fprintf(w, "name:         \"%s\",\n", f.name)
	addIndent(w, ind+1)
	fmt.Fprintf(w, "blob:         bb[%d:%d],\n", f.offStart, f.offStop)
	addIndent(w, ind+1)
	fmt.Fprintf(w, "str_blob:     bs[%d:%d],\n", f.offStart, f.offStop)
	addIndent(w, ind+1)
	fmt.Fprintf(w, "mime:         \"%s\",\n", f.mimeType)
	addIndent(w, ind+1)
	fmt.Fprintf(w, "tag:          \"%s\",\n", f.tag)
	addIndent(w, ind+1)
	fmt.Fprintf(w, "size:         %d,\n", f.size)
	if flags.has(CompressAssets) {
		addIndent(w, ind+1)
		fmt.Fprintf(w, "isCompressed: %v,\n", f.isCompressed)
	}
	addIndent(w, ind)
	fmt.Fprint(w, "}")
}

func (d *directoryAsset) addDirectory(name string) {
	if name == "." {
		return
	}
	elts := strings.Split(name, "/")
	if len(elts) == 1 {
		d.dirs = append(d.dirs, directoryAsset{
			name: elts[0],
		})
	} else {
		for i := range d.dirs {
			if d.dirs[i].name == elts[0] {
				d.dirs[i].addDirectory(path.Join(elts[1:]...))
				return
			}
		}
		panic("directory not found")
	}
}

func (d *directoryAsset) addFile(name string, file *fileAsset) {
	elts := strings.Split(name, "/")
	if len(elts) == 1 {
		d.files = append(d.files, file)
	} else {
		for i := range d.dirs {
			if d.dirs[i].name == elts[0] {
				d.dirs[i].addFile(path.Join(elts[1:]...), file)
				return
			}
		}
		panic("directory not found")
	}
}

var b32Enc = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

const objectFileHeaderTemplate = `// Code generated by go-imbed. DO NOT EDIT.

#include "textflag.h"

`

const objectFileFooterTemplate = `GLOBL ·d(SB),RODATA,$%d
`

func writeObjectFileHeader(file *os.File) error {
	_, err := file.WriteString(objectFileHeaderTemplate)
	return err
}

func writeObjectFileFooter(file *os.File, size int) error {
	_, err := fmt.Fprintf(file, objectFileFooterTemplate, size)
	return err
}

func (a *fileAsset) writeObject(input *os.File, output *os.File, start int, flags ImbedFlag) (int, error) {
	var compressor *gzip.Writer
	var err error
	if _, err := input.Seek(0, 0); err != nil {
		return 0, err
	}
	crc := uint64(0)
	crcTable := crc64.MakeTable(crc64.ECMA)
	pipeIn, pipeOut := io.Pipe()
	if a.isCompressed {
		compressor, _ = gzip.NewWriterLevel(pipeOut, gzip.BestCompression)
	}
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := input.Read(buf)
			if err == io.EOF {
				if a.isCompressed {
					compressor.Close()
				}
				break
			} else if err != nil {
				pipeOut.CloseWithError(err)
				return
			}
			crc = crc64.Update(crc, crcTable, buf[:n])
			if a.isCompressed {
				_, err = compressor.Write(buf[:n])
			} else {
				_, err = pipeOut.Write(buf[:n])
			}
			if err != nil {
				pipeOut.CloseWithError(err)
				return
			}
		}
		pipeOut.Close()
	}()
	defer pipeIn.Close()
	var buf [8]byte
	var _sbuf [32]byte
	addr := start
	size := 0
	read := 0
	for {
		if read, err = io.ReadFull(pipeIn, buf[:]); err != nil {
			if err == io.EOF {
				break
			} else if err != io.ErrUnexpectedEOF {
				return 0, err
			}
		}
		for i := read; i < 8; i++ {
			buf[i] = 0
		}
		var sbuf = _sbuf[0:0]
		for i := range buf {
			sbuf = append(sbuf, []byte("\\x")...)
			if buf[i] < 0x10 {
				sbuf = append(sbuf, '0')
			}
			sbuf = strconv.AppendUint(sbuf, uint64(buf[i]), 16)
		}
		_, err = fmt.Fprintf(output, "DATA ·d+%d(SB)/8,$\"%s\"\n", addr, string(sbuf))
		if err != nil {
			return 0, err
		}
		size += read
		addr += 8
	}
	var crcBuf [8]byte
	binary.LittleEndian.PutUint64(crcBuf[:], crc)
	a.tag = b32Enc.EncodeToString(crcBuf[:])
	a.offStart = start
	a.offStop = start + size
	return addr, nil
}

func writeGoIndex(file io.Writer, testFile io.Writer, pkg string, root *directoryAsset, addr int, flags ImbedFlag) error {
	timestamp := time.Now()
	dir, index, has404Asset := buildIndex(root, flags)
	buf := bytes.Buffer{}
	params := map[string]interface{}{
		"Pkg":           pkg,
		"Size":          addr,
		"IndexCode":     index,
		"DirectoryCode": dir,
		"Date":          fmt.Sprintf("%d, %d", timestamp.Unix(), timestamp.Nanosecond()),
		"Params":        flags,
		"Has404Asset":   flags.BuildHttpHandlerAPI() && has404Asset,
	}
	err := iMustHazTemplate("index.go").Execute(&buf, params)
	if err != nil {
		return err
	}
	format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	_, err = file.Write(buf.Bytes())
	if err != nil {
		return err
	}
	buf.Reset()
	err = iMustHazTemplate("index_test.go").Execute(&buf, params)
	if err != nil {
		return err
	}
	_, err = testFile.Write(buf.Bytes())
	return err
}

func writeAsmIndex(target string) error {
	for _, file := range iMustHazAsmList() {
		src := iMustHazFile(file)
		targetFile, err := ioutil.TempFile(target, "indexasm")
		if err != nil {
			return err
		}
		if _, err = targetFile.WriteString(src); err != nil {
			targetFile.Close()
			os.Remove(targetFile.Name())
			return err
		}
		if err = targetFile.Close(); err != nil {
			os.Remove(targetFile.Name())
			return err
		}
		if err = os.Rename(targetFile.Name(), filepath.Join(target, file)); err != nil {
			os.Remove(targetFile.Name())
			return err
		}
	}
	return nil
}

func CopyBinaryTemplate(target string) error {
	content := iMustHazFile("main.go")
	dst, err := os.OpenFile(filepath.Join(target, "main.go"), os.O_CREATE | os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err = dst.WriteString(content); err != nil {
		return err
	}
	return nil
}

// Creates a Go package `pkgName` from `source` directory contents and puts code
// into `target` location.
func Imbed(source, target, pkgName string, flags ImbedFlag) error {
	if flags.has(BuildHttpFsAPI|BuildUnionFsAPI) {
		flags |= BuildFsAPI
	}
	err := os.MkdirAll(target, 0755)
	if err != nil {
		return err
	}
	dataFile, err := ioutil.TempFile(target, "data")
	if err != nil {
		return err
	}
	defer os.Remove(dataFile.Name())
	err = writeObjectFileHeader(dataFile)
	if err != nil {
		return err
	}
	indexFile, err := ioutil.TempFile(target, "index")
	if err != nil {
		return err
	}
	defer func() {
		indexFile.Close()
		os.Remove(indexFile.Name())
	}()
	testFile, err := ioutil.TempFile(target, "index_test")
	if err != nil {
		return err
	}
	defer func() {
		testFile.Close()
		os.Remove(testFile.Name())
	}()
	addr := 0
	root := &directoryAsset{}
	err = filepath.Walk(source, func(asset string, info os.FileInfo, err error) error {
		assetName, _ := filepath.Rel(source, asset)
		assetName = filepath.ToSlash(assetName)
		if err != nil {
			return err
		}
		if info.IsDir() {
			root.addDirectory(assetName)
			return nil
		}
		file, err := os.OpenFile(asset, os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		defer file.Close()
		fstat, _ := file.Stat()
		m := mime.TypeByExtension(path.Ext(strings.ToLower(asset)))
		if m == "" {
			m = "application/binary"
		}
		var compressed = false
		if flags.CompressAssets() && (strings.HasPrefix(m, "text/") || strings.HasSuffix(m, "+xml") ||
			strings.Contains(m, "javascript") || m == "application/xml") {
			compressed = true
		}
		var entry = fileAsset{
			name:         path.Base(assetName),
			mimeType:     m,
			size:         fstat.Size(),
			isCompressed: compressed,
		}
		addr, err = entry.writeObject(file, dataFile, addr, flags)
		if err != nil {
			return err
		}
		root.addFile(assetName, &entry)
		return nil
	})
	if err != nil {
		return err
	}
	err = writeObjectFileFooter(dataFile, addr)
	if err != nil {
		return err
	}
	err = writeGoIndex(indexFile, testFile, pkgName, root, addr, flags)
	if err != nil {
		return err
	}
	dataFile.Close()
	indexFile.Close()
	testFile.Close()
	err = os.Rename(dataFile.Name(), filepath.Join(target, "data.s"))
	if err != nil {
		return err
	}
	err = os.Rename(indexFile.Name(), filepath.Join(target, "index.go"))
	if err != nil {
		return err
	}
	err = os.Rename(testFile.Name(), filepath.Join(target, "index_test.go"))
	if err != nil {
		return err
	}
	return writeAsmIndex(target)
}