package icons

import (
	"image"
	"image/color"
	"image/jpeg"
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

func TestLoadXPMIcon(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "TestTheme", "apps", "64", "demo.xpm")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`/* XPM */
static char *demo_xpm[] = {
"4 4 2 1",
" 	c None",
"R	c #ff0000",
"RRRR",
"R  R",
"R  R",
"RRRR"
};`)
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
		t.Fatal("XPM icon was not resolved")
	}
	if got := img.Bounds().Dx(); got != 4 {
		t.Fatalf("width=%d, want 4", got)
	}
	if _, _, _, alpha := img.At(0, 0).RGBA(); alpha == 0 {
		t.Fatal("border pixel is transparent")
	}
	if _, _, _, alpha := img.At(2, 2).RGBA(); alpha != 0 {
		t.Fatal("center pixel should be transparent (None)")
	}
}

func TestLoadActionsIcon(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "TestTheme", "actions", "16", "system-log-out.png")
	if err := writePNG(path); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{
		themes: []string{"TestTheme"},
		roots:  []string{root},
		cache:  make(map[string]cached),
	}
	if _, found := loader.Load("system-log-out"); !found {
		t.Fatal("action icon in <theme>/actions/<size> was not resolved")
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

func TestLoadPrefersActiveThemeAcrossRoots(t *testing.T) {
	localRoot := t.TempDir()
	systemRoot := t.TempDir()
	if err := writeColorPNG(filepath.Join(localRoot, "hicolor", "apps", "64", "demo.png"), color.NRGBA{B: 255, A: 255}); err != nil {
		t.Fatal(err)
	}
	if err := writeColorPNG(filepath.Join(systemRoot, "TestTheme", "apps", "64", "demo.png"), color.NRGBA{R: 255, A: 255}); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{
		themes: []string{"TestTheme", "hicolor"},
		roots:  []string{localRoot, systemRoot},
		cache:  make(map[string]cached),
	}
	img, found := loader.Load("demo")
	if !found {
		t.Fatal("theme icon was not resolved")
	}
	assertPixel(t, img, color.NRGBA{R: 255, A: 255})
}

func TestLoadInheritedThemeIcon(t *testing.T) {
	root := t.TempDir()
	if err := writeThemeIndex(filepath.Join(root, "Child", "index.theme"), "Inherits=Parent\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeThemeIndex(filepath.Join(root, "Parent", "index.theme"), "Directories=apps/64\n\n[apps/64]\nSize=64\nContext=Applications\nType=Fixed\n"); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(filepath.Join(root, "Parent", "apps", "64", "demo.png")); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{
		themes: []string{"Child"},
		roots:  []string{root},
		cache:  make(map[string]cached),
	}
	if _, found := loader.Load("demo"); !found {
		t.Fatal("inherited theme icon was not resolved")
	}
}

func TestLoadDeclaredJPGDirectory(t *testing.T) {
	root := t.TempDir()
	if err := writeThemeIndex(filepath.Join(root, "TestTheme", "index.theme"), "Directories=custom/apps\n\n[custom/apps]\nSize=64\nContext=Applications\nType=Fixed\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeJPEG(filepath.Join(root, "TestTheme", "custom", "apps", "demo.jpg")); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{
		themes: []string{"TestTheme"},
		roots:  []string{root},
		cache:  make(map[string]cached),
	}
	if _, found := loader.Load("demo"); !found {
		t.Fatal("JPG icon in a declared theme directory was not resolved")
	}
}

func TestLoadUndeclaredThemeDirectory(t *testing.T) {
	root := t.TempDir()
	if err := writeThemeIndex(filepath.Join(root, "TestTheme", "index.theme"), "Directories=apps/48\n\n[apps/48]\nSize=48\nContext=Applications\nType=Fixed\n"); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(filepath.Join(root, "TestTheme", "apps", "48-light", "demo.png")); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{
		themes: []string{"TestTheme"},
		roots:  []string{root},
		cache:  make(map[string]cached),
	}
	if _, found := loader.Load("demo"); !found {
		t.Fatal("icon in an undeclared theme directory was not resolved")
	}
}

func TestLoadStandaloneSubdirectoryIcon(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "mate-system-monitor", "upload.svg")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect width="16" height="16" fill="#00ff00"/></svg>`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{
		standaloneDirs: []string{root},
		cache:          make(map[string]cached),
	}
	if _, found := loader.Load("mate-system-monitor/upload"); !found {
		t.Fatal("standalone pixmap subdirectory icon was not resolved")
	}
}

func writePNG(path string) error {
	return writeColorPNG(path, color.White)
}

func writeColorPNG(path string, c color.Color) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, c)
	if err := png.Encode(file, img); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func writeJPEG(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	if err := jpeg.Encode(file, img, nil); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func writeThemeIndex(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("[Icon Theme]\n"+body), 0o644)
}

func assertPixel(t *testing.T, img image.Image, want color.NRGBA) {
	t.Helper()
	r, g, b, a := img.At(0, 0).RGBA()
	got := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	if got != want {
		t.Fatalf("pixel=%#v, want %#v", got, want)
	}
}
