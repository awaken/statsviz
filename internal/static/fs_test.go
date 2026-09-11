package static

import (
	"bytes"
	"io/fs"
	"testing"
)

func TestAssetsFaviconType(t *testing.T) {
	assets := Assets()
	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(index, []byte(`rel="icon"`)) || !bytes.Contains(index, []byte(`href="./favicon.ico"`)) {
		t.Fatal("embedded index does not link the favicon")
	}
	if bytes.Contains(index, []byte(`type="image/svg+xml"`)) {
		t.Fatal("embedded index declares the ICO favicon as SVG")
	}

	icon, err := fs.ReadFile(assets, "favicon.ico")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(icon, []byte{0, 0, 1, 0}) {
		t.Fatal("embedded favicon is not an ICO image")
	}
}
