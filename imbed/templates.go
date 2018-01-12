// Copyright 2017 Alexey Naidyonov. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

//+build !bootstrap

package imbed

import (
	"text/template"

	"github.com/growler/go-imbed/imbed/internal/templates"
	"path/filepath"
)

func iMustHazAsmList() []string {
	root, _ := templates.Open("")
	fis, _ := root.Readdir(-1)
	var list = make([]string, 0, len(fis))
	for i := range fis {
		if filepath.Ext(fis[i].Name()) == ".s" {
			list = append(list, fis[i].Name())
		}
	}
	return list
}

func iMustHazFile(name string) string {
	return templates.Must(name).String()
}

func iMustHazTemplate(name string) *template.Template {
	return template.Must(template.New("").Parse(iMustHazFile(name)))
}
