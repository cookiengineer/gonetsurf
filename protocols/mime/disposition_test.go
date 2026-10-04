package mime

import "testing"

func TestParseContentDisposition(t *testing.T) {
	disposition, filename, err := ParseContentDisposition(`attachment; filename="report.pdf"`)
	if err != nil || disposition != "attachment" || filename != "report.pdf" {
		t.Fatalf("got (%q, %q, %v)", disposition, filename, err)
	}

	_, filename, err = ParseContentDisposition(`attachment; filename*=UTF-8''%E2%82%AC%20rates.txt`)
	if err != nil || filename != "€ rates.txt" {
		t.Fatalf("filename* = (%q, %v)", filename, err)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"../../etc/passwd", "passwd"},
		{`C:\Windows\evil.exe`, "evil.exe"},
		{"..", ""},
		{"CON", ""},
		{"CON.txt", ""},
		{"  spaced.txt  ", "spaced.txt"},
		{"with\u0000null", "withnull"},
	}
	for _, test := range tests {
		if got := SanitizeFilename(test.input); got != test.want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
