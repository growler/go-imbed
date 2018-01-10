// Copyright 2017 Alexey Naidyonov. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

//+build !bootstrap

package imbed

import (
	"text/template"

	"github.com/growler/go-imbed/imbed/internal/templates"
)

func iMustHazAsmList() []string {
	return nil
}

func iMustHazFile(name string) string {
	return templates.Must(name).String()
}

func iMustHazTemplate(name string) *template.Template {
	return template.Must(template.New("").Parse(iMustHazFile(name)))
}
