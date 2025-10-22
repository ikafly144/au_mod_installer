package lexer

var whitespace = map[rune]struct{}{
	' ':  {},
	'\t': {},
	'\b': {},
	'\n': {},
	'\r': {},
}

var syntax = map[rune]struct{}{
	'{': {}, // 0x7B 123
	'}': {}, // 0x7D 125
}
