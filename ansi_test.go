package ansihtml

import "testing"

func TestDecodeBasicColor(t *testing.T) {
	spans := Decode("\x1b[31mhello\x1b[0m world")
	if len(spans) != 2 {
		t.Fatalf("got %d spans, want 2: %#v", len(spans), spans)
	}
	if spans[0].Text != "hello" {
		t.Errorf("spans[0].Text = %q, want %q", spans[0].Text, "hello")
	}
	if spans[0].Style.Foreground != (Color{Mode: ColorBasic, Index: 1}) {
		t.Errorf("spans[0].Style.Foreground = %+v, want red", spans[0].Style.Foreground)
	}
	if spans[1].Text != " world" {
		t.Errorf("spans[1].Text = %q, want %q", spans[1].Text, " world")
	}
	if spans[1].Style != (Style{}) {
		t.Errorf("spans[1].Style = %+v, want zero value", spans[1].Style)
	}
}

func TestDecodeTruecolorAndBold(t *testing.T) {
	spans := Decode("\x1b[1;38;2;10;20;30mtext\x1b[0m")
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1: %#v", len(spans), spans)
	}
	want := Style{Bold: true, Foreground: Color{Mode: ColorTrue, R: 10, G: 20, B: 30}}
	if spans[0].Style != want {
		t.Errorf("style = %+v, want %+v", spans[0].Style, want)
	}
}

func TestDecodeDropsNonSGRSequences(t *testing.T) {
	spans := Decode("\x1b[2Jclear screen")
	if len(spans) != 1 || spans[0].Text != "clear screen" {
		t.Fatalf("got %#v, want a single plain span", spans)
	}
}

func TestEncodeRoundTripsStyle(t *testing.T) {
	original := "\x1b[1;31mwarning\x1b[0m: disk almost full"
	spans := Decode(original)
	reencoded := Encode(spans)

	roundTripped := Decode(reencoded)
	if len(roundTripped) != len(spans) {
		t.Fatalf("got %d spans after round trip, want %d", len(roundTripped), len(spans))
	}
	for i := range spans {
		if roundTripped[i] != spans[i] {
			t.Errorf("span %d = %+v, want %+v", i, roundTripped[i], spans[i])
		}
	}
}

func TestEncodeEmitsMinimalDiff(t *testing.T) {
	spans := []Span{
		{Text: "warning", Style: Style{Bold: true, Foreground: Color{Mode: ColorBasic, Index: 1}}},
		{Text: ": disk full", Style: Style{}},
	}
	got := Encode(spans)
	want := "\x1b[1;31mwarning\x1b[22;39m: disk full"
	if got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}
}

func TestEncodeKeepsFaintWhenBoldTurnsOff(t *testing.T) {
	spans := []Span{
		{Text: "a", Style: Style{Bold: true, Faint: true}},
		{Text: "b", Style: Style{Faint: true}},
	}
	got := Encode(spans)
	want := "\x1b[1;2ma\x1b[22;2mb\x1b[0m"
	if got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}

	roundTripped := Decode(got)
	if len(roundTripped) != len(spans) {
		t.Fatalf("got %d spans after round trip, want %d: %+v", len(roundTripped), len(spans), roundTripped)
	}
	for i := range spans {
		if roundTripped[i] != spans[i] {
			t.Errorf("span %d = %+v, want %+v", i, roundTripped[i], spans[i])
		}
	}
}

func TestEncodeOmitsCodesForUnchangedStyle(t *testing.T) {
	spans := []Span{
		{Text: "one ", Style: Style{Underline: true}},
		{Text: "two", Style: Style{Underline: true}},
	}
	got := Encode(spans)
	want := "\x1b[4mone two\x1b[0m"
	if got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}
}

func TestToHTMLEscapesAndStyles(t *testing.T) {
	spans := Decode("\x1b[1;32mok</script>\x1b[0m")
	got := ToHTML(spans)
	want := `<span style="font-weight:bold;color:#008000">ok&lt;/script&gt;</span>`
	if got != want {
		t.Errorf("ToHTML() = %q, want %q", got, want)
	}
}

func TestToHTMLPlainTextHasNoSpan(t *testing.T) {
	spans := Decode("just text")
	got := ToHTML(spans)
	if got != "just text" {
		t.Errorf("ToHTML() = %q, want %q", got, "just text")
	}
}
