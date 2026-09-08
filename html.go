package ansihtml

import (
	"fmt"
	"html"
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
