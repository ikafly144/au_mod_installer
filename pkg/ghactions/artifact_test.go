package ghactions

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
)

func TestIsArtifactURL(t *testing.T) {
	if !IsArtifactURL("gha://owner/repo/artifact/123") {
		t.Fatal("expected gha url to be detected")
	}
	if IsArtifactURL("https://example.com/file.zip") {
		t.Fatal("did not expect https url to be detected as gha")
	}
}

func makeTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return b.Bytes()
}

func TestExtractFromArtifactZip_FilePatternSelectsInnerFile(t *testing.T) {
	zipData := makeTestZip(t, map[string]string{
		"a/readme.txt":                         "hello",
		"out/BepInEx-Unity.IL2CPP-win-x64.zip": "x64",
		"out/BepInEx-Unity.IL2CPP-win-x86.zip": "x86",
	})

	name, data, err := extractFromArtifactZip(zipData, `BepInEx-Unity\.IL2CPP-win-x64.*\.zip`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "BepInEx-Unity.IL2CPP-win-x64.zip" {
		t.Fatalf("got filename %q", name)
	}
	if string(data) != "x64" {
		t.Fatalf("got content %q, want %q", string(data), "x64")
	}
}

func TestExtractFromArtifactZip_NoMatch(t *testing.T) {
	zipData := makeTestZip(t, map[string]string{
		"a/readme.txt": "hello",
	})

	_, _, err := extractFromArtifactZip(zipData, `\.dll$`)
	if err == nil {
		t.Fatal("expected error for no match")
	}
}
