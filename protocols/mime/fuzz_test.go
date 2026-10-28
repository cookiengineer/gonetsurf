package mime

import "testing"

func FuzzContentDisposition(f *testing.F) {
	for _, seed := range []string{
		"",
		"attachment",
		`attachment; filename="report.pdf"`,
		`inline; filename*=UTF-8''%E2%82%AC.txt`,
		"attachment; filename=../../etc/passwd",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		_, _, _ = ParseContentDisposition(value)
		_ = SanitizeFilename(value)
	})
}
