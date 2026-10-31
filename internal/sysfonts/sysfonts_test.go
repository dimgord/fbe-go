package sysfonts

import (
	"reflect"
	"testing"
)

func TestParseFontconfig(t *testing.T) {
	out := "Noto Serif,Noto Serif Regular\n\nArial\nNoto Serif\n , Bad\n"
	want := []string{"Arial", "Noto Serif"}
	if got := parseFontconfig(out); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFamilyFromFilename(t *testing.T) {
	cases := map[string]string{
		"/nix/store/abc/DejaVuSans-Bold.ttf": "Deja Vu Sans",
		"/fonts/PTSans-Italic.otf":           "PT Sans",
		"/fonts/open_sans_Regular.ttf":       "open sans",
		"/fonts/Roboto.ttf":                  "Roboto",
	}
	for path, want := range cases {
		if got := familyFromFilename(path); got != want {
			t.Errorf("familyFromFilename(%q) = %q, want %q", path, got, want)
		}
	}
}
