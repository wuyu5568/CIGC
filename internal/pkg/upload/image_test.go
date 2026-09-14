package upload

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectImageExt(t *testing.T) {
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10}
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	webp := []byte("RIFF....WEBP....")
	if ext, ok := DetectImageExt("a.jpg", jpeg); !ok || ext != ".jpg" {
		t.Fatalf("jpg: %s %v", ext, ok)
	}
	if ext, ok := DetectImageExt("a.jpeg", jpeg); !ok || ext != ".jpeg" {
		t.Fatalf("jpeg: %s %v", ext, ok)
	}
	if ext, ok := DetectImageExt("a.png", png); !ok || ext != ".png" {
		t.Fatalf("png: %s %v", ext, ok)
	}
	if ext, ok := DetectImageExt("a.webp", webp); !ok || ext != ".webp" {
		t.Fatalf("webp: %s %v", ext, ok)
	}
	if _, ok := DetectImageExt("a.gif", []byte("GIF89a")); ok {
		t.Fatal("gif should fail")
	}
	if _, ok := DetectImageExt("a.png", []byte("not an image")); ok {
		t.Fatal("bad data should fail")
	}
}

func TestSaveAndURLs(t *testing.T) {
	root := t.TempDir()
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
	rel, err := Save(root, "web3_goods", ".png", png)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel, "/uploads/web3_goods/") || !strings.HasSuffix(rel, ".png") {
		t.Fatalf("rel=%s", rel)
	}
	full := filepath.Join(root, strings.TrimPrefix(rel, "/uploads/"))
	if _, err := os.Stat(full); err != nil {
		t.Fatal(err)
	}
	r := &http.Request{Host: "ispaygijdysxt.com", Header: http.Header{"X-Forwarded-Proto": []string{"https"}}}
	base := PublicBaseURL(r)
	if base != "https://ispaygijdysxt.com" {
		t.Fatalf("base=%s", base)
	}
	got := AbsoluteURL(base, rel)
	if got != "https://ispaygijdysxt.com"+rel {
		t.Fatalf("abs=%s", got)
	}
	if NormalizeStored(got) != rel {
		t.Fatalf("norm=%s", NormalizeStored(got))
	}
}
