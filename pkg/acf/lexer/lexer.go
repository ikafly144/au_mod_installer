package lexer

import (
	"errors"
)

type Lexer struct {
	input  []rune
	tokens []any
}

func newLexer(input string) *Lexer {
	return &Lexer{input: []rune(input)}
}

func Lex(input string) ([]any, error) {
	l := newLexer(input)
	for len(l.input) > 0 {
		strings, err := l.lexString()
		if err != nil {
			return nil, err
		}
		if strings != "" {
			l.tokens = append(l.tokens, strings)
			continue
		}

		syntax, err := l.lexSyntax()
		if err != nil {
			return nil, err
		}
		if syntax != ' ' {
			l.tokens = append(l.tokens, syntax)
			continue
		}
	}
	return l.tokens, nil
}

func (l *Lexer) lexString() (string, error) {
	if l.input[0] == '"' {
		l.input = l.input[1:]
	} else {
		return "", nil
	}

	var strings []rune
	for i, char := range l.input {
		if char == '"' {
			l.input = l.input[i+1:]
			return string(strings), nil
		}
		strings = append(strings, char)
	}

	return "", errors.New("expected end-of-string quote")
}

func (l *Lexer) lexSyntax() (rune, error) {
	char := l.input[0]
	_, ok := whitespace[char]
	if ok {
		l.input = l.input[1:]
		return ' ', nil
	}
	_, ok = syntax[char]
	if ok {
		l.input = l.input[1:]
		return char, nil
	}
	return ' ', errors.New("expected syntax character")
}
