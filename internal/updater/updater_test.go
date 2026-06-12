package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current string
		latest  string
		want    bool
	}{
		{current: "v0.2.0", latest: "v0.2.1", want: true},
		{current: "v0.2.1", latest: "v0.2.1", want: false},
		{current: "v0.2.1", latest: "v0.2.0", want: false},
		{current: "v0.2.0-1-gabc-dirty", latest: "v0.2.1", want: true},
		{current: "dev", latest: "v0.2.1", want: true},
	}

	for _, tt := range cases {
		if got := IsNewer(tt.current, tt.latest); got != tt.want {
			t.Fatalf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestExtractBinary(t *testing.T) {
	archive := testArchive(t, "diffmate", "binary")

	content, err := extractBinary(archive)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "binary" {
		t.Fatalf("unexpected binary content: %q", content)
	}
}

func testArchive(t *testing.T, name, content string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(content)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
