# ansi-html-bridge

Terminal programs color their output with ANSI SGR escape sequences
(`\x1b[31m`, `\x1b[1;38;5;208m`, and so on). That's fine as long as the
output stays in a terminal, but the moment you want to show a build log,
a test failure, or a REPL session on a web page, those bytes are either
stripped out (losing all the color) or dumped raw (showing garbage
characters). This library converts between the two: parse ANSI-colored
text into a plain data structure, then render that structure as HTML.

The parsing step and the rendering step are both pure functions with no
shared state, so a colored log line goes in and a `<span>`-wrapped HTML
string comes out, with nothing in between that depends on program state,
time, or the filesystem.

## Usage

```go
package main

import (
	"fmt"

	ansihtml "github.com/lmmartinez9/ansi-html-bridge"
)

func main() {
	// A line a build tool might print: bold red "FAIL", then plain text.
	line := "\x1b[1;31mFAIL\x1b[0m internal/parser (0.4s)"

	spans := ansihtml.Decode(line)
	fmt.Println(ansihtml.ToHTML(spans))
	// <span style="font-weight:bold;color:#800000">FAIL</span> internal/parser (0.4s)

	// Spans can also be turned back into an ANSI string, e.g. after
	// filtering or re-coloring them in code.
	fmt.Println(ansihtml.Encode(spans) == line) // true (same styling, re-encoded)
}
```

`Decode` understands:

- text attributes: bold, faint, italic, underline, blink, reverse,
  strikethrough, and their "off" codes
- the 8 standard and 8 bright colors (SGR 30-37, 40-47, 90-97, 100-107)
- 256-color palette codes (`38;5;N` / `48;5;N`)
- 24-bit truecolor codes (`38;2;R;G;B` / `48;2;R;G;B`)
- reset (SGR 0)

Non-SGR control sequences (cursor movement, screen clearing, and similar)
are recognized and dropped rather than leaking into the output text,
since the `Span` representation has nowhere to put them.

## Why a `Span` in the middle

`Decode` and `ToHTML`/`Encode` never talk to each other directly. Every
public function takes plain values and returns plain values:

```go
func Decode(input string) []Span
func Encode(spans []Span) string
func ToHTML(spans []Span) string
func HTMLToSpans(input string) []Span
```

`HTMLToSpans` is the inverse of `ToHTML`, so a colored ANSI line can go
in, come out as HTML, and be parsed back into spans - useful if you store
the HTML and later want to re-render it as ANSI with `Encode`, or filter
it by style. It only understands the shape `ToHTML` itself produces
(bare text, `<br>` line breaks, `<span style="...">` runs), not arbitrary
HTML, and it collapses every color to 24-bit RGB, since CSS has no
concept of the basic/256/truecolor distinction ANSI makes.

That makes each one trivial to unit test in isolation (see
`ansi_test.go`), and it means adding a third output format later - say,
Markdown code fences with a color legend, or a JSON export - only
requires a new `func([]Span) string`, not changes to the parser.

## Command line

`cmd/ansihtml` wraps the library for piping logs through it:

```sh
go run ./cmd/ansihtml < build.log > build.html   # ANSI in, HTML out
go run ./cmd/ansihtml -pre < build.log > build.html  # wrap output in <pre>
go run ./cmd/ansihtml -reverse < build.html > build.log  # HTML back to ANSI
```

It reads all of stdin, converts it, and writes the result to stdout; there's
no in-place file editing or directory walking, since the point is to sit in
a shell pipeline next to the tool that produced the colored output.

## Status

Early skeleton. Decoding, HTML rendering, parsing HTML back into spans,
encoding spans back to ANSI (as a minimal diff against the previous span's
style, not a full reset-and-restyle every time), and the `ansihtml` CLI
wrapper all work and are tested (the library, not the CLI itself, which is
a thin enough wrapper that the library tests cover its logic).
