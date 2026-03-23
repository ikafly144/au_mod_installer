package acf

import (
	"fmt"
	"strings"

	"github.com/ikafly144/au_mod_installer/pkg/acf/lexer"
)

type Parser struct {
	token []any
}

func newParser(token []any) *Parser {
	return &Parser{token: token}
}

func FromString(input string) (map[string]any, error) {
	tokens, err := lexer.Lex(input)
	if err != nil {
		return nil, err
	}
	p := newParser(tokens)
	return p.parseACF()
}

func (p *Parser) parseACF() (map[string]any, error) {
	ast := make(map[string]any)
	for {
		if len(p.token) == 0 {
			return ast, nil
		}
		id, ok := p.token[0].(string)
		if ok {
			p.token = p.token[1:]
			if p.token[0] == '{' {
				p.token = p.token[1:]
				child, err := p.parseACF()
				if err != nil {
					return nil, err
				}
				ast[id] = child
				continue
			}

			if p.token[0] == '}' {
				p.token = p.token[1:]
				return ast, nil
			}

			v, ok := p.token[0].(string)
			if !ok {
				return nil, ErrNotPointer
			}
			p.token = p.token[1:]
			ast[id] = v
		} else {
			if len(p.token) > 0 && p.token[0] == '}' {
				p.token = p.token[1:]
				return ast, nil
			}
			return ast, nil
		}
	}
}

func ToString(ast map[string]any) string {
	var result strings.Builder
	for k, v := range ast {
		result.WriteString(k + "\t\t")
		switch v := v.(type) {
		case string:
			result.WriteString(v)
		case map[string]any:
			result.WriteString(ToString(v))
		default:
			fmt.Fprintf(&result, "%v", v)
		}
		result.WriteString("\n")
	}
	return result.String()
}
