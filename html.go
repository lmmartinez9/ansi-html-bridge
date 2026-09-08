package ansihtml

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

// basicPalette gives the standard terminal RGB values for SGR colors
// 30-37 and 90-97 (indices 8-15 are the bright variants). These match the
// classic xterm defaults, not any particular terminal emulator's theme.
var basicPalette = [16]string{
	"#000000", "#800000", "#008000", "#808000",
	"#000080", "#800080", "#008080", "#c0c0c0",
	"#808080", "#ff0000", "#00ff00", "#ffff00",
	"#0000ff", "#ff00ff", "#00ffff", "#ffffff",
}

// Hex returns the CSS hex color for c, and false if c is unset.
func (c Color) Hex() (string, bool) {
	switch c.Mode {
	case ColorBasic:
		return basicPalette[c.Index], true
	case Color256:
		return hex256(c.Index), true
	case ColorTrue:
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B), true
	default:
		return "", false
	}
}

// hex256 reproduces the xterm 256-color palette: 0-15 are the basic
// colors, 16-231 are a 6x6x6 RGB cube, and 232-255 are a grayscale ramp.
func hex256(i uint8) string {
	switch {
	case i < 16:
		return basicPalette[i]
	case i < 232:
		n := int(i) - 16
		r := cubeLevel(n / 36)
		g := cubeLevel((n / 6) % 6)
		b := cubeLevel(n % 6)
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	default:
		level := 8 + (int(i)-232)*10
		return fmt.Sprintf("#%02x%02x%02x", level, level, level)
	}
}

// cubeLevel converts one 0-5 coordinate of the 6x6x6 color cube into an
// 8-bit channel value, matching xterm's spacing.
func cubeLevel(n int) int {
	if n == 0 {
		return 0
	}
	return 55 + n*40
}

// ToHTML renders spans as HTML, one <span> per styled run with an inline
// style attribute. Plain (unstyled) spans are emitted as bare escaped
// text. Newlines become <br> so the result is ready to drop into a <pre>
// or <div> without further processing.
func ToHTML(spans []Span) string {
	var b strings.Builder
	for _, span := range spans {
		text := strings.ReplaceAll(html.EscapeString(span.Text), "\n", "<br>\n")
		css := cssFor(span.Style)
		if css == "" {
			b.WriteString(text)
			continue
		}
		b.WriteString(`<span style="`)
		b.WriteString(css)
		b.WriteString(`">`)
		b.WriteString(text)
		b.WriteString(`</span>`)
	}
	return b.String()
}

func cssFor(s Style) string {
	var parts []string
	if s.Bold {
		parts = append(parts, "font-weight:bold")
	}
	if s.Faint {
		parts = append(parts, "opacity:0.6")
	}
	if s.Italic {
		parts = append(parts, "font-style:italic")
	}

	var decorations []string
	if s.Underline {
		decorations = append(decorations, "underline")
	}
	if s.Strikethrough {
		decorations = append(decorations, "line-through")
	}
	if len(decorations) > 0 {
		parts = append(parts, "text-decoration:"+strings.Join(decorations, " "))
	}

	fg, bg := s.Foreground, s.Background
	if s.Reverse {
		fg, bg = bg, fg
	}
	if hex, ok := fg.Hex(); ok {
		parts = append(parts, "color:"+hex)
	}
	if hex, ok := bg.Hex(); ok {
		parts = append(parts, "background-color:"+hex)
	}

	return strings.Join(parts, ";")
}

// HTMLToSpans is the inverse of ToHTML: it parses HTML of the form ToHTML
// produces (bare text, "<br>" line breaks, and "<span style=\"...\">" runs)
// back into spans. It only understands that specific shape, not arbitrary
// HTML, since its job is to close the loop on ToHTML's own output rather
// than to be a general-purpose parser.
//
// Color is lossy in the round trip: CSS has no notion of ANSI's basic,
// 256-color, and truecolor spaces, so every parsed color comes back as
// ColorTrue with the hex value's RGB, even if the original span used
// ColorBasic or Color256.
func HTMLToSpans(s string) []Span {
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
	for i < len(s) {
		if s[i] != '<' {
			j := strings.IndexByte(s[i:], '<')
			if j == -1 {
				text.WriteString(html.UnescapeString(s[i:]))
				break
			}
			text.WriteString(html.UnescapeString(s[i : i+j]))
			i += j
			continue
		}

		end := strings.IndexByte(s[i:], '>')
		if end == -1 {
			text.WriteString(html.UnescapeString(s[i:]))
			break
		}
		tag := s[i : i+end+1]
		i += end + 1
		lower := strings.ToLower(tag)

		switch {
		case lower == "<br>" || lower == "<br/>" || lower == "<br />":
			text.WriteByte('\n')
			// ToHTML always follows "<br>" with a literal newline purely
			// for readability of the HTML source; that byte isn't part
			// of the original text, so swallow it here.
			if i < len(s) && s[i] == '\n' {
				i++
			}
		case strings.HasPrefix(lower, "<span"):
			flush()
			style = parseStyleAttr(tag)
		case lower == "</span>":
			flush()
			style = Style{}
		}
	}
	flush()
	return spans
}

// parseStyleAttr reads the style="..." attribute out of a "<span ...>"
// start tag and turns the declarations cssFor is known to emit back into a
// Style.
func parseStyleAttr(tag string) Style {
	var style Style

	const key = `style="`
	idx := strings.Index(tag, key)
	if idx == -1 {
		return style
	}
	rest := tag[idx+len(key):]
	end := strings.IndexByte(rest, '"')
	if end == -1 {
		return style
	}

	for _, decl := range strings.Split(rest[:end], ";") {
		prop, val, ok := strings.Cut(strings.TrimSpace(decl), ":")
		if !ok {
			continue
		}
		prop, val = strings.TrimSpace(prop), strings.TrimSpace(val)
		switch prop {
		case "font-weight":
			style.Bold = val == "bold"
		case "opacity":
			style.Faint = val == "0.6"
		case "font-style":
			style.Italic = val == "italic"
		case "text-decoration":
			for _, d := range strings.Fields(val) {
				switch d {
				case "underline":
					style.Underline = true
				case "line-through":
					style.Strikethrough = true
				}
			}
		case "color":
			style.Foreground = parseHexColor(val)
		case "background-color":
			style.Background = parseHexColor(val)
		}
	}
	return style
}

// parseHexColor turns a "#rrggbb" CSS color into a truecolor Color. It
// returns the zero Color for anything else, matching Color's "unset"
// convention.
func parseHexColor(s string) Color {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return Color{}
	}
	r, err1 := strconv.ParseUint(s[0:2], 16, 8)
	g, err2 := strconv.ParseUint(s[2:4], 16, 8)
	b, err3 := strconv.ParseUint(s[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return Color{}
	}
	return Color{Mode: ColorTrue, R: uint8(r), G: uint8(g), B: uint8(b)}
}
