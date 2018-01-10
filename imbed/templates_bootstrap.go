// Copyright 2017 Alexey Naidyonov. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

//+build bootstrap

package imbed

import (
	"text/template"
	"os"
	"io/ioutil"
	"path/filepath"
)

func iMustHazAsmList() []string {
	fis, err := ioutil.ReadDir("_templates")
	if err != nil {
		panic(err)
	}
	var list = make([]string, 0, len(fis))
	for i := range fis {
		if filepath.Ext(fis[i].Name()) == ".s" {
			list = append(list, fis[i].Name())
		}
	}
	return list
}

func iMustHazFile(name string) string {
	file, err := os.OpenFile(filepath.Join("_templates", name), os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func iMustHazTemplate(name string) *template.Template {
	return template.Must(template.New("").Parse(iMustHazFile(name)))
}
