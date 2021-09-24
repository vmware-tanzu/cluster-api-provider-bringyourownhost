package cloudinit

import (
	"bytes"
	"text/template"
)

//counterfeiter:generate . ITemplateParser
type ITemplateParser interface {
	ParseTemplate(string) (string, error)
}

type TemplateParser struct {
	Template interface{}
}

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
