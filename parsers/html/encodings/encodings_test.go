package encodings

import "testing"

func TestSniffBOM(t *testing.T) {
	result := Sniff("", []byte{0xEF, 0xBB, 0xBF, 'h', 'i'})
	if result.Name != "utf-8" || !result.Certain {
		t.Fatalf("UTF-8 BOM sniff = (%q, certain=%v)", result.Name, result.Certain)
	}
	result = Sniff("", []byte{0xFF, 0xFE, 'h', 0x00})
	if result.Name != "utf-16le" || !result.Certain {
		t.Fatalf("UTF-16LE BOM sniff = (%q, certain=%v)", result.Name, result.Certain)
	}
}

func TestSniffContentType(t *testing.T) {
	result := Sniff("text/html; charset=iso-8859-2", nil)
	if result.Name != "iso-8859-2" || !result.Certain {
		t.Fatalf("content type sniff = (%q, certain=%v)", result.Name, result.Certain)
	}
}

func TestSniffPrescan(t *testing.T) {
	html := []byte(`<!doctype html><meta charset="windows-1251"><p>hi`)
	result := Sniff("text/html", html)
	if result.Name != "windows-1251" || result.Certain {
		t.Fatalf("prescan = (%q, certain=%v)", result.Name, result.Certain)
	}

	httpEquiv := []byte(`<meta http-equiv="Content-Type" content="text/html; charset=Shift_JIS">`)
	result = Sniff("", httpEquiv)
	if result.Name != "shift_jis" {
		t.Fatalf("http-equiv prescan = %q, want shift_jis", result.Name)
	}

	comment := []byte(`<!-- <meta charset="utf-8"> --><meta charset="gbk">`)
	result = Sniff("", comment)
	if result.Name != "gbk" {
		t.Fatalf("comment-skipping prescan = %q, want gbk", result.Name)
	}
}

func TestSniffDefault(t *testing.T) {
	result := Sniff("", []byte("hello"))
	if result.Name != "windows-1252" || result.Certain {
		t.Fatalf("default sniff = (%q, certain=%v)", result.Name, result.Certain)
	}
}

func TestDecode(t *testing.T) {
	result := Sniff("text/html; charset=windows-1252", nil)
	decoded, err := result.Decode([]byte{0xE9})
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "é" {
		t.Fatalf("decoded = %q, want é", string(decoded))
	}
}

func TestApplyMetaChange(t *testing.T) {
	result := Sniff("", []byte("plain"))
	updated, changed := result.ApplyMetaChange("utf-8")
	if !changed || updated.Name != "utf-8" {
		t.Fatalf("ApplyMetaChange = (%q, %v), want (utf-8, true)", updated.Name, changed)
	}
	_, changed = updated.ApplyMetaChange("utf-8")
	if changed {
		t.Fatal("expected no change for the same encoding")
	}
	_, changed = updated.ApplyMetaChange("not-a-real-encoding")
	if changed {
		t.Fatal("expected no change for an invalid encoding")
	}
}
