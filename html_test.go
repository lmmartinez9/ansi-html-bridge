package ansihtml

import "testing"

func TestHTMLToSpansParsesStyleAndText(t *testing.T) {
	got := HTMLToSpans(`<span style="font-weight:bold;color:#008000">ok&lt;/script&gt;</span>`)
	want := []Span{{
		Text:  "ok</script>",
		Style: Style{Bold: true, Foreground: Color{Mode: ColorTrue, G: 0x80}},
	}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("HTMLToSpans() = %+v, want %+v", got, want)
	}
}

func TestHTMLToSpansPlainText(t *testing.T) {
	got := HTMLToSpans("just text")
	if len(got) != 1 || got[0] != (Span{Text: "just text"}) {
		t.Errorf("HTMLToSpans() = %+v, want a single plain span", got)
	}
}

func TestHTMLToSpansLineBreak(t *testing.T) {
	got := HTMLToSpans("line one<br>\nline two")
	want := Span{Text: "line one\nline two"}
	if len(got) != 1 || got[0] != want {
		t.Errorf("HTMLToSpans() = %+v, want %+v", got, want)
	}
}

func TestHTMLToSpansUnderlineAndStrikethrough(t *testing.T) {
	got := HTMLToSpans(`<span style="text-decoration:underline line-through">both</span>`)
	want := Span{Text: "both", Style: Style{Underline: true, Strikethrough: true}}
	if len(got) != 1 || got[0] != want {
		t.Errorf("HTMLToSpans() = %+v, want %+v", got, want)
	}
}

func TestToHTMLRendersHyperlink(t *testing.T) {
	spans := []Span{{Text: "docs", Style: Style{Bold: true, Link: "https://example.com/a?b=c&d"}}}
	got := ToHTML(spans)
	want := `<a href="https://example.com/a?b=c&amp;d" style="font-weight:bold">docs</a>`
	if got != want {
		t.Errorf("ToHTML() = %q, want %q", got, want)
	}
}

func TestHTMLToSpansParsesHyperlink(t *testing.T) {
	got := HTMLToSpans(`<a href="https://example.com/a?b=c&amp;d" style="font-weight:bold">docs</a>`)
	want := Span{Text: "docs", Style: Style{Bold: true, Link: "https://example.com/a?b=c&d"}}
	if len(got) != 1 || got[0] != want {
		t.Errorf("HTMLToSpans() = %+v, want %+v", got, want)
	}
}

func TestToHTMLAndHTMLToSpansRoundTripHyperlink(t *testing.T) {
	spans := Decode("\x1b]8;;https://example.com\x1b\\click\x1b]8;;\x1b\\ plain")
	roundTripped := HTMLToSpans(ToHTML(spans))

	if len(roundTripped) != len(spans) {
		t.Fatalf("got %d spans after round trip, want %d: %+v", len(roundTripped), len(spans), roundTripped)
	}
	for i := range spans {
		if roundTripped[i].Text != spans[i].Text {
			t.Errorf("span %d text = %q, want %q", i, roundTripped[i].Text, spans[i].Text)
		}
		if roundTripped[i].Style.Link != spans[i].Style.Link {
			t.Errorf("span %d link = %q, want %q", i, roundTripped[i].Style.Link, spans[i].Style.Link)
		}
	}
}

func TestToHTMLAndHTMLToSpansRoundTrip(t *testing.T) {
	spans := Decode("\x1b[1;31mwarning\x1b[0m: \x1b[4mdisk\x1b[0m almost\nfull")
	roundTripped := HTMLToSpans(ToHTML(spans))

	if len(roundTripped) != len(spans) {
		t.Fatalf("got %d spans after round trip, want %d: %+v", len(roundTripped), len(spans), roundTripped)
	}
	for i := range spans {
		if roundTripped[i].Text != spans[i].Text {
			t.Errorf("span %d text = %q, want %q", i, roundTripped[i].Text, spans[i].Text)
		}
		wantHex, wantOK := spans[i].Style.Foreground.Hex()
		gotHex, gotOK := roundTripped[i].Style.Foreground.Hex()
		if wantOK != gotOK || wantHex != gotHex {
			t.Errorf("span %d foreground = %v (ok=%v), want %v (ok=%v)", i, gotHex, gotOK, wantHex, wantOK)
		}
		if roundTripped[i].Style.Bold != spans[i].Style.Bold {
			t.Errorf("span %d bold = %v, want %v", i, roundTripped[i].Style.Bold, spans[i].Style.Bold)
		}
		if roundTripped[i].Style.Underline != spans[i].Style.Underline {
			t.Errorf("span %d underline = %v, want %v", i, roundTripped[i].Style.Underline, spans[i].Style.Underline)
		}
	}
}
