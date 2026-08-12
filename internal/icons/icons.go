package icons

import (
	"bufio"
	"compress/gzip"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

type cached struct {
	image image.Image
	found bool
}

// Loader resolves PNG/JPEG/GIF/SVG application icons from the active GTK theme and
// the freedesktop hicolor fallback. Unsupported formats simply use the UI's
// generated fallback tile.
type Loader struct {
	themes []string
	roots  []string
	cache  map[string]cached
}

func NewLoader() *Loader {
	theme := gtkIconTheme()
	themes := []string{theme, "hicolor", "Adwaita"}
	if theme == "" {
		themes = themes[1:]
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	roots := []string{filepath.Join(dataHome, "icons"), filepath.Join(os.Getenv("HOME"), ".icons")}
	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}
	for _, dir := range filepath.SplitList(dataDirs) {
		if dir != "" {
			roots = append(roots, filepath.Join(dir, "icons"))
		}
	}
	return &Loader{themes: unique(themes), roots: unique(roots), cache: make(map[string]cached)}
}

func (l *Loader) Load(name string) (image.Image, bool) {
	if strings.TrimSpace(name) == "" {
		return nil, false
	}
	if value, ok := l.cache[name]; ok {
		return value.image, value.found
	}
	paths := l.candidates(name)
	for _, path := range paths {
		if img, err := decode(path); err == nil {
			l.cache[name] = cached{image: img, found: true}
			return img, true
		}
	}
	l.cache[name] = cached{}
	return nil, false
}

func (l *Loader) candidates(name string) []string {
	if filepath.IsAbs(name) {
		return []string{name}
	}
	extensions := []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".svgz"}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".svgz", ".xpm":
		extensions = []string{""}
	}
	sizes := []string{"64", "48", "96", "32", "128", "256"}
	xdgSizes := []string{"64x64", "48x48", "96x96", "32x32", "128x128", "256x256"}
	var result []string
	for _, root := range l.roots {
		for _, theme := range l.themes {
			for _, ext := range extensions {
				for _, size := range sizes {
					result = append(result, filepath.Join(root, theme, "apps", size, name+ext))
				}
				result = append(result, filepath.Join(root, theme, "apps", "scalable", name+ext))
				for _, size := range xdgSizes {
					result = append(result, filepath.Join(root, theme, size, "apps", name+ext))
				}
				result = append(result, filepath.Join(root, theme, "scalable", "apps", name+ext))
				result = append(result, filepath.Join(root, theme, name+ext))
			}
		}
		for _, ext := range extensions {
			result = append(result, filepath.Join(root, name+ext))
		}
	}
	for _, dataDir := range append([]string{filepath.Dir(l.roots[0])}, filepath.SplitList(defaultDataDirs())...) {
		for _, ext := range extensions {
			result = append(result, filepath.Join(dataDir, "pixmaps", name+ext))
		}
	}
	return result
}

func decode(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".svg":
		return decodeSVG(file)
	case ".svgz":
		reader, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return decodeSVG(reader)
	default:
		img, _, err := image.Decode(file)
		return img, err
	}
}

func decodeSVG(reader io.Reader) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(reader, oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, err
	}
	if icon.ViewBox.W <= 0 || icon.ViewBox.H <= 0 {
		return nil, errors.New("svg icon has invalid size")
	}

	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	w, h := float64(size), float64(size)
	if icon.ViewBox.W > icon.ViewBox.H {
		h = w * icon.ViewBox.H / icon.ViewBox.W
	} else {
		w = h * icon.ViewBox.W / icon.ViewBox.H
	}
	icon.SetTarget((size-w)/2, (size-h)/2, w, h)
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	icon.Draw(rasterx.NewDasher(size, size, scanner), 1)
	return img, nil
}

func gtkIconTheme() string {
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".config", "gtk-4.0", "settings.ini"),
		filepath.Join(os.Getenv("HOME"), ".config", "gtk-3.0", "settings.ini"),
	}
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			key, value, ok := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
			if ok && strings.TrimSpace(key) == "gtk-icon-theme-name" {
				file.Close()
				return strings.Trim(strings.TrimSpace(value), "'\"")
			}
		}
		file.Close()
	}
	return ""
}

func defaultDataDirs() string {
	if value := os.Getenv("XDG_DATA_DIRS"); value != "" {
		return value
	}
	return "/usr/local/share:/usr/share"
}

func unique(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
