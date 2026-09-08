// Package ansihtml converts between ANSI SGR escape sequences and HTML,
// using a slice of Span as the pure intermediate representation.
package ansihtml

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

const esc = 0x1b

// ColorMode identifies which of the three ANSI color spaces a Color uses.
type ColorMode int

const (
	ColorNone ColorMode = iota
	ColorBasic          // one of the 16 standard/bright colors (index 0-15)
	Color256            // xterm 256-color palette (index 0-255)
	ColorTrue           // 24-bit RGB
)

// Color is a single foreground or background color in one of the three
// color spaces a terminal can express. The zero value means "unset".
type Color struct {
	Mode  ColorMode
	Index uint8 // used by ColorBasic and Color256
	R, G, B uint8 // used by ColorTrue
}

// Style is the set of SGR attributes in effect for a run of text.
type Style struct {
	Bold          bool
	Faint         bool
	Italic        bool
	Underline     bool
	Blink         bool
	Reverse       bool
	Strikethrough bool
	Foreground    Color
	Background    Color
}

// Span is a run of text that shares a single Style. Decode produces spans,
// Encode and ToHTML consume them.
type Span struct {
	Text  string
	Style Style
}

// Decode parses a string containing ANSI SGR escape sequences into a slice
// of styled spans. Non-SGR CSI sequences (cursor movement, screen clearing,
// and so on) are recognized and discarded rather than leaking into the
// text, since they carry no style information the intermediate form can
// represent. Decode never returns an error: malformed or truncated escape
// sequences are treated as literal text so the function stays total.
func Decode(input string) []Span {
	var spans []Span
	style := Style{}
	var text strings.Builder

	flush := func() {
		if text.Len() > 0 {
			spans = append(spans, Span{Text: text.String(), Style: style})
			text.Reset()
		}
	}

	i := 0
	for i < len(input) {
		if input[i] == esc && i+1 < len(input) && input[i+1] == '[' {
			end := i + 2
			for end < len(input) && !isFinalByte(input[end]) {
				end++
			}
			if end >= len(input) {
				text.WriteString(input[i:])
				break
			}
			final := input[end]
			params := input[i+2 : end]
			if final == 'm' {
				flush()
				style = applySGR(style, params)
			}
			i = end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(input[i:])
		text.WriteRune(r)
		i += size
	}
	flush()
	return spans
}

func isFinalByte(b byte) bool {
	return b >= 0x40 && b <= 0x7e
}

// applySGR folds one "ESC[...m" parameter list into style, following the
// terminal convention that later codes override earlier ones and 0 resets
// everything.
func applySGR(style Style, params string) Style {
	codes := splitParams(params)
	for i := 0; i < len(codes); i++ {
		code := codes[i]
		switch {
		case code == 0:
			style = Style{}
		case code == 1:
			style.Bold = true
		case code == 2:
			style.Faint = true
		case code == 3:
			style.Italic = true
		case code == 4:
			style.Underline = true
		case code == 5:
			style.Blink = true
		case code == 7:
			style.Reverse = true
		case code == 9:
			style.Strikethrough = true
		case code == 22:
			style.Bold = false
			style.Faint = false
		case code == 23:
			style.Italic = false
		case code == 24:
			style.Underline = false
		case code == 25:
			style.Blink = false
		case code == 27:
			style.Reverse = false
		case code == 29:
			style.Strikethrough = false
		case code >= 30 && code <= 37:
			style.Foreground = Color{Mode: ColorBasic, Index: uint8(code - 30)}
		case code == 38:
			c, consumed := parseExtendedColor(codes[i+1:])
			style.Foreground = c
			i += consumed
		case code == 39:
			style.Foreground = Color{}
		case code >= 40 && code <= 47:
			style.Background = Color{Mode: ColorBasic, Index: uint8(code - 40)}
		case code == 48:
			c, consumed := parseExtendedColor(codes[i+1:])
			style.Background = c
			i += consumed
		case code == 49:
			style.Background = Color{}
		case code >= 90 && code <= 97:
			style.Foreground = Color{Mode: ColorBasic, Index: uint8(code-90) + 8}
		case code >= 100 && code <= 107:
			style.Background = Color{Mode: ColorBasic, Index: uint8(code-100) + 8}
		}
	}
	return style
}

// parseExtendedColor reads the "5;N" (256-color) or "2;R;G;B" (truecolor)
// tail that follows an SGR 38 or 48 code. It returns the parsed color and
// how many of the trailing codes it consumed, so the caller can skip past
// them.
func parseExtendedColor(rest []int) (Color, int) {
	if len(rest) == 0 {
		return Color{}, 0
	}
	switch rest[0] {
	case 5:
		if len(rest) < 2 {
			return Color{}, len(rest)
		}
		return Color{Mode: Color256, Index: uint8(rest[1])}, 2
	case 2:
		if len(rest) < 4 {
			return Color{}, len(rest)
		}
		return Color{Mode: ColorTrue, R: uint8(rest[1]), G: uint8(rest[2]), B: uint8(rest[3])}, 4
	}
	return Color{}, 0
}

// splitParams turns "1;38;5;9" into [1 38 5 9]. An empty string (bare
// "ESC[m") is treated as "0", matching how real terminals reset on it.
func splitParams(params string) []int {
	if params == "" {
		return []int{0}
	}
	parts := strings.Split(params, ";")
	codes := make([]int, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			codes = append(codes, 0)
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		codes = append(codes, n)
	}
	return codes
}

// Encode renders spans back into a string with ANSI SGR escape sequences.
// Each span is wrapped independently (full style, then a trailing reset)
// rather than emitting a minimal diff against the previous span's style;
// the output is correct but more verbose than a hand-written sequence.
func Encode(spans []Span) string {
	var b strings.Builder
	for _, span := range spans {
		codes := sgrCodes(span.Style)
		if len(codes) > 0 {
			b.WriteString("\x1b[")
			b.WriteString(strings.Join(codes, ";"))
			b.WriteString("m")
		}
		b.WriteString(span.Text)
		if len(codes) > 0 {
			b.WriteString("\x1b[0m")
		}
	}
	return b.String()
}

func sgrCodes(s Style) []string {
	var codes []string
	if s.Bold {
		codes = append(codes, "1")
	}
	if s.Faint {
		codes = append(codes, "2")
	}
	if s.Italic {
		codes = append(codes, "3")
	}
	if s.Underline {
		codes = append(codes, "4")
	}
	if s.Blink {
		codes = append(codes, "5")
	}
	if s.Reverse {
		codes = append(codes, "7")
	}
	if s.Strikethrough {
		codes = append(codes, "9")
	}
	codes = append(codes, colorCodes(s.Foreground, true)...)
	codes = append(codes, colorCodes(s.Background, false)...)
	return codes
}

func colorCodes(c Color, foreground bool) []string {
	switch c.Mode {
	case ColorBasic:
		if c.Index < 8 {
			base := 30
			if !foreground {
				base = 40
			}
			return []string{strconv.Itoa(base + int(c.Index))}
		}
		base := 90
		if !foreground {
			base = 100
		}
		return []string{strconv.Itoa(base + int(c.Index) - 8)}
	case Color256:
		prefix := "38"
		if !foreground {
			prefix = "48"
		}
		return []string{prefix, "5", strconv.Itoa(int(c.Index))}
	case ColorTrue:
		prefix := "38"
		if !foreground {
			prefix = "48"
		}
		return []string{prefix, "2", strconv.Itoa(int(c.R)), strconv.Itoa(int(c.G)), strconv.Itoa(int(c.B))}
	default:
		return nil
	}
}
