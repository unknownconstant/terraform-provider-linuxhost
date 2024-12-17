package linuxhost_client

import "strings"

type Parser[T any, C any] struct {
	Items          []T
	CurrentItem    *T
	Handlers       []func(*Parser[T, C], string)
	CurrentContext *C
}

func NewParser[T any, C any](CurrentContext *C) *Parser[T, C] {
	return &Parser[T, C]{
		CurrentContext: CurrentContext,
	}
}

func (p *Parser[T, C]) AddHandler(handler func(*Parser[T, C], string)) {
	p.Handlers = append(p.Handlers, handler)
}

func (p *Parser[T, C]) ParseLine(line string) {
	for _, handler := range p.Handlers {
		handler(p, line)
	}
}

func (p *Parser[T, C]) AddItem(newItem T) {
	p.Items = append(p.Items, newItem)
	p.CurrentItem = &p.Items[len(p.Items)-1]
}

func (p *Parser[T, C]) Parse(text string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		p.ParseLine(line)
	}
}
