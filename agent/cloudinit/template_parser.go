// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"bytes"
	"text/template"
)

//counterfeiter:generate . ITemplateParser
type ITemplateParser interface {
	ParseTemplate(string) (string, error)
}

// TemplateParser cloudinit templates parsing using ITemplateParser
type TemplateParser struct {
	Template interface{}
}

// ParseTemplate parses and returns the parsed template content
func (tp TemplateParser) ParseTemplate(templateContent string) (string, error) {
	tmpl, err := template.New("byoh").Parse(templateContent)
	if err != nil {
		return templateContent, err
	}

	var parsedContent bytes.Buffer

	err = tmpl.Execute(&parsedContent, tp.Template)
	if err != nil {
		return templateContent, err
	}

	return parsedContent.String(), nil
}
