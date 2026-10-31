package roundtrip

import (
	"os"
	"strings"
	"testing"
)

func book(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../../testdata/rich.fb2")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestValidDocumentPasses(t *testing.T) {
	fb, err := ParseVerified(book(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(fb.Bodies) == 0 {
		t.Fatal("no bodies parsed")
	}
}

func TestRejectsWhatWouldLoseData(t *testing.T) {
	src := book(t)
	cases := []struct {
		name, from, to, want string
	}{
		{"typo in an opening tag", "<book-title>", "<book-titel>", "Unknown element <book-titel>"},
		{"typo eats the rest of the document", "<first-name>", "<fist-name>", "Unknown element <fist-name>"},
		{"mismatched close tag", "</section>", "</sektion>", ""},
		{"unclosed root", "</FictionBook>", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bad := strings.Replace(src, c.from, c.to, 1)
			if bad == src {
				t.Fatalf("%q not in rich.fb2", c.from)
			}
			_, err := ParseVerified(bad)
			if err == nil {
				t.Fatal("accepted a document that loses data")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q, want it to mention %q", err, c.want)
			}
		})
	}
}

func TestDataLossMessage(t *testing.T) {
	in := "<body><section><p>a</p><p>b</p></section></body>"
	if got := dataLossMessage(in, in); got != "" {
		t.Errorf("no loss, got %q", got)
	}
	got := dataLossMessage(in, "<body><section><p>a</p></section></body>")
	if !strings.Contains(got, "<p> count dropped from 2 to 1") {
		t.Errorf("got %q", got)
	}
	// The topmost lost element is named, not the cascade under it.
	got = dataLossMessage(in, "<body></body>")
	if !strings.Contains(got, "<section>") {
		t.Errorf("got %q", got)
	}
}

func TestCountOpen(t *testing.T) {
	s := `<p>x</p><p class="y"/><publish-info/><p
>`
	if n := countOpen(s, "p"); n != 3 {
		t.Errorf("countOpen = %d, want 3 (must skip <publish-info>)", n)
	}
}
