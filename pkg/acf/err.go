package acf

var (
	ErrNotPointer = &ParseError{"expected a pointer"}
)

type ParseError struct {
	Msg string
}

func (e *ParseError) Error() string {
	return e.Msg
}
