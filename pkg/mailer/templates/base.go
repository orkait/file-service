package templates

import (
	"bytes"
	"file-service/pkg/mailer/registry"
	"html/template"
)

type TemplateContext map[string]interface{}

type EmailTemplate interface {
	GetName() string
	RenderAny(context any) (html string, text string, err error)
}

type Parser[T any] func(context T) (T, error)

type TypedTemplate[T any] struct {
	Name         string
	HTMLTemplate *template.Template
	TextTemplate *template.Template
	Parse        Parser[T]
}

func (t *TypedTemplate[T]) GetName() string {
	return t.Name
}

func (t *TypedTemplate[T]) Render(context T) (string, string, error) {
	if t.Parse != nil {
		parsed, err := t.Parse(context)
		if err != nil {
			return "", "", err
		}
		context = parsed
	}

	var htmlBuf bytes.Buffer
	if err := t.HTMLTemplate.Execute(&htmlBuf, context); err != nil {
		return "", "", err
	}

	var textBuf bytes.Buffer
	if t.TextTemplate != nil {
		if err := t.TextTemplate.Execute(&textBuf, context); err != nil {
			return "", "", err
		}
	}

	return htmlBuf.String(), textBuf.String(), nil
}

func (t *TypedTemplate[T]) RenderAny(context any) (string, string, error) {
	typedContext, ok := context.(T)
	if !ok {
		return "", "", registry.ErrInvalidTemplateContextType(t.Name)
	}

	return t.Render(typedContext)
}

func NewTemplate[T any](name string, htmlTmpl string, textTmpl string, parser Parser[T]) (*TypedTemplate[T], error) {
	htmlTemplate, err := template.New(name + "_html").Parse(htmlTmpl)
	if err != nil {
		return nil, err
	}

	var textTemplate *template.Template
	if textTmpl != "" {
		textTemplate, err = template.New(name + "_text").Parse(textTmpl)
		if err != nil {
			return nil, err
		}
	}

	return &TypedTemplate[T]{
		Name:         name,
		HTMLTemplate: htmlTemplate,
		TextTemplate: textTemplate,
		Parse:        parser,
	}, nil
}

func NewRawTemplate(name string, htmlTmpl string, textTmpl string) (*TypedTemplate[TemplateContext], error) {
	parser := func(context TemplateContext) (TemplateContext, error) {
		if context == nil {
			return nil, registry.ErrTemplateContextRequired
		}
		return context, nil
	}

	return NewTemplate(name, htmlTmpl, textTmpl, parser)
}
