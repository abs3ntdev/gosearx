package htmlx

import "testing"

func TestSanitizeHTML(t *testing.T) {
	cases := []struct{ in, want string }{
		// empty/broken img dropped, text kept
		{`<img src="" height="16px" width="16px" alt="favicon" class="css-1gwoof1"/>Go Packages`, "Go Packages"},
		// safe formatting kept; disallowed class attr dropped
		{`<b class="x">bold</b> and <i>italic</i>`, `<b>bold</b> and <i>italic</i>`},
		// script dropped entirely
		{`hi<script>alert(1)</script> there`, "hi there"},
		// valid http image kept
		{`<img src="https://e.com/a.png" alt="a"/>`, `<img src="https://e.com/a.png" alt="a"/>`},
		// disallowed tag unwrapped
		{`<div><span>x</span></div>`, `<span>x</span>`},
		// plain text passthrough
		{"just text", "just text"},
	}
	for _, c := range cases {
		got := SanitizeHTML(c.in)
		if got != c.want {
			t.Errorf("SanitizeHTML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestStripHTML(t *testing.T) {
	if got := StripHTML(`<b>Title</b> - <i>site</i>`); got != "Title - site" {
		t.Errorf("StripHTML = %q", got)
	}
}
