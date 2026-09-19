// Command ansihtml pipes ANSI-colored text through the ansihtml library.
// By default it reads SGR-colored text on stdin and writes HTML on stdout;
// with -reverse it does the opposite, turning HTML back into ANSI.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	ansihtml "github.com/lmmartinez9/ansi-html-bridge"
)

func main() {
	reverse := flag.Bool("reverse", false, "convert HTML back into ANSI instead of ANSI into HTML")
	pre := flag.Bool("pre", false, "wrap HTML output in a <pre> element")
	flag.Parse()

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ansihtml:", err)
		os.Exit(1)
	}

	var out string
	if *reverse {
		out = ansihtml.Encode(ansihtml.HTMLToSpans(string(input)))
	} else {
		out = ansihtml.ToHTML(ansihtml.Decode(string(input)))
		if *pre {
			out = "<pre>" + out + "</pre>"
		}
	}

	if _, err := io.WriteString(os.Stdout, out); err != nil {
		fmt.Fprintln(os.Stderr, "ansihtml:", err)
		os.Exit(1)
	}
}
