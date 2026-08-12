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
	if err := writePNG(path); err != nil {
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

func TestLoadScalableSVGIcon(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "TestTheme", "apps", "scalable", "demo.svg")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect width="16" height="16" fill="#ff0000"/></svg>`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{
		themes: []string{"TestTheme"},
		roots:  []string{root},
		cache:  make(map[string]cached),
	}
	img, found := loader.Load("demo")
	if !found {
		t.Fatal("scalable SVG icon was not resolved")
	}
	if got := img.Bounds().Dx(); got != 64 {
		t.Fatalf("width=%d, want 64", got)
	}
	if _, _, _, alpha := img.At(32, 32).RGBA(); alpha == 0 {
		t.Fatal("center pixel is transparent")
	}
}

func TestLoadDirectRootIcon(t *testing.T) {
	root := t.TempDir()
	if err := writePNG(filepath.Join(root, "freedoom.png")); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{
		themes: []string{"TestTheme"},
		roots:  []string{root},
		cache:  make(map[string]cached),
	}
	if _, found := loader.Load("freedoom"); !found {
		t.Fatal("direct icon root file was not resolved")
	}
}

func writePNG(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	if err := png.Encode(file, img); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
