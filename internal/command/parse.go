package command

import (
	"fmt"

	"github.com/dyl-03/shadow/internal/device"
	"github.com/dyl-03/shadow/internal/model"
)

// Parser converts raw command bytes into a structured command using the
// device type template.
type Parser struct {
	templates *device.TemplateStore
}

func NewParser(t *device.TemplateStore) *Parser {
	return &Parser{templates: t}
}

func (p *Parser) Parse(dev *model.Device, payload []byte) (map[string]string, error) {
	tmpl := p.templates.Resolve(dev.Template)
	if len(payload) == 0 {
		return nil, fmt.Errorf("command payload must not be empty")
	}
	out := make(map[string]string)
	for field := range tmpl.Schema {
		out[field] = string(payload)
	}
	return out, nil
}
