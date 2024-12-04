package linuxhost_client

import "strings"

type Parser[T any] struct {
	Items       []T
	CurrentItem *T
	Handlers    []func(*Parser[T], string)
}

func NewParser[T any]() *Parser[T] {
	return &Parser[T]{}
}

func (p *Parser[T]) AddHandler(handler func(*Parser[T], string)) {
	p.Handlers = append(p.Handlers, handler)
}

func (p *Parser[T]) ParseLine(line string) {
	for _, handler := range p.Handlers {
		handler(p, line)
	}
}

func (p *Parser[T]) AddItem(newItem T) {
	p.Items = append(p.Items, newItem)
	p.CurrentItem = &p.Items[len(p.Items)-1]
}

func (p *Parser[T]) Parse(text string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		p.ParseLine(line)
	}
}
