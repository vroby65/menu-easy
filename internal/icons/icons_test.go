package icons

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReverseDNSIconName(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "TestTheme", "apps", "64", "org.gnome.Calculator.png")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{
		themes: []string{"TestTheme"},
		roots:  []string{root},
		cache:  make(map[string]cached),
	}
	if _, found := loader.Load("org.gnome.Calculator"); !found {
		t.Fatal("reverse-DNS icon name was not resolved with image extensions")
	}
}
