package unityrichtext

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type richState struct {
	bold      bool
	italic    bool
	underline bool
	col       color.Color
	sizeScale float32
	upper     bool
	lower     bool
}

type textSegment struct {
	text     string
	style    fyne.TextStyle
	col      color.Color
	textSize float32
}

func (s *textSegment) Inline() bool { return true }

func (s *textSegment) Textual() string { return s.text }

func (s *textSegment) Visual() fyne.CanvasObject {
	col := s.col
	if col == nil {
		col = theme.Color(theme.ColorNameForeground)
	}
	obj := canvas.NewText(s.text, col)
	obj.Alignment = fyne.TextAlignLeading
	obj.TextStyle = s.style
	if s.textSize > 0 {
		obj.TextSize = s.textSize
	} else {
		obj.TextSize = theme.TextSize()
	}
	return obj
}

func (s *textSegment) Update(o fyne.CanvasObject) {
	obj := o.(*canvas.Text)
	obj.Text = s.text
	if s.col == nil {
		obj.Color = theme.Color(theme.ColorNameForeground)
	} else {
		obj.Color = s.col
	}
	obj.Alignment = fyne.TextAlignLeading
	obj.TextStyle = s.style
	if s.textSize > 0 {
		obj.TextSize = s.textSize
	} else {
		obj.TextSize = theme.TextSize()
	}
	obj.Refresh()
}

func (s *textSegment) Select(_, _ fyne.Position) {}

func (s *textSegment) SelectedText() string { return "" }

func (s *textSegment) Unselect() {}

// Parse converts a subset of Unity Rich Text tags into Fyne rich text segments.
func Parse(input string) []widget.RichTextSegment {
	if input == "" {
		return []widget.RichTextSegment{&textSegment{text: ""}}
	}
	segments := make([]widget.RichTextSegment, 0, 4)
	stack := make([]richState, 0, 8)
	state := richState{sizeScale: 1}
	var buf strings.Builder

	flush := func() {
		if buf.Len() == 0 {
			return
		}
		text := buf.String()
		if state.upper {
			text = strings.ToUpper(text)
		} else if state.lower {
			text = strings.ToLower(text)
		}
		size := theme.TextSize()
		if state.sizeScale > 0 {
			size *= state.sizeScale
		}
		segments = append(segments, &textSegment{
			text: text,
			style: fyne.TextStyle{
				Bold:      state.bold,
				Italic:    state.italic,
				Underline: state.underline,
			},
			col:      state.col,
			textSize: size,
		})
		buf.Reset()
	}

	for i := 0; i < len(input); {
		if input[i] != '<' {
			buf.WriteByte(input[i])
			i++
			continue
		}

		endRel := strings.IndexByte(input[i:], '>')
		if endRel <= 0 {
			buf.WriteByte(input[i])
			i++
			continue
		}
		tag := strings.TrimSpace(input[i+1 : i+endRel])
		if tag == "" {
			buf.WriteByte(input[i])
			i++
			continue
		}
		lowerTag := strings.ToLower(tag)

		flush()
		switch {
		case lowerTag == "br" || lowerTag == "br/":
			segments = append(segments, &textSegment{text: "\n"})
		case lowerTag == "b":
			stack = append(stack, state)
			state.bold = true
		case lowerTag == "/b":
			state, stack = popState(state, stack)
		case lowerTag == "i":
			stack = append(stack, state)
			state.italic = true
		case lowerTag == "/i":
			state, stack = popState(state, stack)
		case lowerTag == "u":
			stack = append(stack, state)
			state.underline = true
		case lowerTag == "/u":
			state, stack = popState(state, stack)
		case strings.HasPrefix(lowerTag, "color="):
			stack = append(stack, state)
			colText := strings.TrimSpace(tag[len("color="):])
			colText = strings.Trim(colText, "\"'")
			if col, ok := parseColor(colText); ok {
				state.col = col
			}
		case lowerTag == "/color":
			state, stack = popState(state, stack)
		case strings.HasPrefix(lowerTag, "size="):
			stack = append(stack, state)
			sizeText := strings.TrimSpace(tag[len("size="):])
			sizeText = strings.Trim(sizeText, "\"'")
			if scale, ok := parseSizeScale(sizeText); ok {
				state.sizeScale = scale
			}
		case lowerTag == "/size":
			state, stack = popState(state, stack)
		case lowerTag == "uppercase" || lowerTag == "allcaps":
			stack = append(stack, state)
			state.upper = true
			state.lower = false
		case lowerTag == "/uppercase" || lowerTag == "/allcaps":
			state, stack = popState(state, stack)
		case lowerTag == "lowercase" || lowerTag == "smallcaps":
			stack = append(stack, state)
			state.lower = true
			state.upper = false
		case lowerTag == "/lowercase" || lowerTag == "/smallcaps":
			state, stack = popState(state, stack)
		case strings.HasPrefix(lowerTag, "#"):
			stack = append(stack, state)
			if col, ok := parseColor(lowerTag); ok {
				state.col = col
			}
		case lowerTag == "/#":
			state, stack = popState(state, stack)
		default:
			// Unsupported tag is ignored.
		}
		i += endRel + 1
	}
	flush()

	if len(segments) == 0 {
		return []widget.RichTextSegment{&textSegment{text: ""}}
	}
	return segments
}

func popState(current richState, stack []richState) (richState, []richState) {
	if len(stack) == 0 {
		return current, stack
	}
	last := stack[len(stack)-1]
	return last, stack[:len(stack)-1]
}

func parseColor(v string) (color.Color, bool) {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "black":
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}, true
	case "blue":
		return color.NRGBA{R: 0, G: 0, B: 255, A: 255}, true
	case "cyan":
		return color.NRGBA{R: 0, G: 255, B: 255, A: 255}, true
	case "gray", "grey":
		return color.NRGBA{R: 128, G: 128, B: 128, A: 255}, true
	case "green":
		return color.NRGBA{R: 0, G: 128, B: 0, A: 255}, true
	case "magenta":
		return color.NRGBA{R: 255, G: 0, B: 255, A: 255}, true
	case "red":
		return color.NRGBA{R: 255, G: 0, B: 0, A: 255}, true
	case "white":
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, true
	case "yellow":
		return color.NRGBA{R: 255, G: 255, B: 0, A: 255}, true
	}

	if after, ok := strings.CutPrefix(s, "#"); ok {
		hex := after
		switch len(hex) {
		case 6:
			n, err := strconv.ParseUint(hex, 16, 32)
			if err != nil {
				return nil, false
			}
			return color.NRGBA{
				R: uint8((n >> 16) & 0xFF),
				G: uint8((n >> 8) & 0xFF),
				B: uint8(n & 0xFF),
				A: 255,
			}, true
		case 8:
			n, err := strconv.ParseUint(hex, 16, 32)
			if err != nil {
				return nil, false
			}
			return color.NRGBA{
				R: uint8((n >> 24) & 0xFF),
				G: uint8((n >> 16) & 0xFF),
				B: uint8((n >> 8) & 0xFF),
				A: uint8(n & 0xFF),
			}, true
		}
	}
	return nil, false
}

func parseSizeScale(v string) (float32, bool) {
	s := strings.ToLower(strings.TrimSpace(v))
	if s == "" {
		return 0, false
	}
	if before, ok := strings.CutSuffix(s, "%"); ok {
		n, err := strconv.ParseFloat(before, 64)
		if err != nil || n <= 0 {
			return 0, false
		}
		return float32(n / 100), true
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	if n > 10 {
		return float32(n) / theme.TextSize(), true
	}
	return float32(n), true
}
