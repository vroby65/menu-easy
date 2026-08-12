package icons

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/xml"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/fyne-io/image/xpm" // registers the XPM decoder for image.Decode
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const desiredIconSize = 64

var supportedExtensions = []string{".png", ".jpg", ".jpeg", ".svg", ".svgz", ".xpm", ".gif"}

type cached struct {
	image image.Image
	found bool
}

type themeDirectory struct {
	rel     string
	size    int
	minSize int
	maxSize int
	scale   int
	typ     string
	context string
}

type themeInfo struct {
	inherits    []string
	directories []themeDirectory
}

// Loader resolves PNG/JPEG/GIF/SVG/XPM application icons from the active
// desktop/window-manager icon theme, its inherited themes, and the freedesktop
// hicolor fallback. Unsupported formats simply use the UI's generated fallback
// tile.
type Loader struct {
	themes         []string
	roots          []string
	standaloneDirs []string
	cache          map[string]cached
	index          map[string][]string
	indexBuilt     bool
}

func NewLoader() *Loader {
	roots := iconThemeRoots()
	themes := resolveIconThemes(activeIconThemes(), roots)
	return &Loader{
		themes:         themes,
		roots:          roots,
		standaloneDirs: standaloneIconDirs(),
		cache:          make(map[string]cached),
	}
}

func iconThemeRoots() []string {
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
	return unique(roots)
}

func standaloneIconDirs() []string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	dirs := append(iceWMIconDirs(), filepath.Join(dataHome, "pixmaps"))
	for _, dir := range filepath.SplitList(defaultDataDirs()) {
		if dir != "" {
			dirs = append(dirs, filepath.Join(dir, "pixmaps"))
		}
	}
	return unique(dirs)
}

func (l *Loader) Load(name string) (image.Image, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
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
		return directCandidates(name)
	}
	var result []string
	if strings.Contains(filepath.ToSlash(name), "/") && !filepath.IsAbs(name) {
		for _, dir := range l.standaloneDirs {
			result = append(result, directCandidates(filepath.Join(dir, filepath.FromSlash(name)))...)
		}
	}
	l.ensureIndex()
	for _, key := range lookupKeys(name) {
		result = append(result, l.index[key]...)
	}
	for _, root := range l.roots {
		result = append(result, directCandidates(filepath.Join(root, name))...)
	}
	return unique(result)
}

func (l *Loader) ensureIndex() {
	if l.indexBuilt {
		return
	}
	l.index = make(map[string][]string)
	for _, theme := range resolveIconThemes(l.themes, l.roots) {
		for _, root := range l.roots {
			l.indexTheme(filepath.Join(root, theme))
		}
	}
	for _, dir := range l.standaloneDirs {
		l.indexDirectory(dir, dir)
	}
	l.indexBuilt = true
}

func (l *Loader) indexTheme(root string) {
	info, ok := readThemeInfo(root)
	if !ok {
		if isDir(root) {
			l.indexDirectory(root, root)
		}
		return
	}

	directories := append([]themeDirectory(nil), info.directories...)
	if len(directories) == 0 {
		l.indexDirectory(root, root)
		return
	}
	sort.SliceStable(directories, func(i, j int) bool {
		return directoryPriority(directories[i]) < directoryPriority(directories[j])
	})
	for _, dir := range directories {
		l.indexDirectory(filepath.Join(root, filepath.FromSlash(dir.rel)), root)
	}
	l.indexDirectory(root, root)
}

func (l *Loader) indexDirectory(dir, root string) {
	if !isDir(dir) {
		return
	}
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && skipIconDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		l.addIcon(path, root)
		return nil
	})
}

func (l *Loader) addIcon(path, root string) {
	ext := iconExtension(path)
	if ext == "" {
		return
	}
	name := filepath.Base(path)
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	keys := []string{name, stem}
	if rel, err := filepath.Rel(root, path); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		rel = filepath.ToSlash(rel)
		keys = append(keys, rel, strings.TrimSuffix(rel, filepath.Ext(rel)))
	}
	for _, key := range unique(keys) {
		l.index[key] = appendIfMissing(l.index[key], path)
	}
}

func directCandidates(path string) []string {
	if iconExtension(path) != "" {
		return []string{path}
	}
	result := make([]string, 0, len(supportedExtensions))
	for _, ext := range supportedExtensions {
		result = append(result, path+ext)
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

func activeIconThemes() []string {
	var themes []string
	if value := os.Getenv("MENU_EASY_ICON_THEME"); value != "" {
		themes = append(themes, splitList(value)...)
	}

	desktops := currentDesktops()
	addForDesktop := func(desktop string) {
		switch strings.ToLower(desktop) {
		case "mate":
			themes = append(themes, gsettingsIconTheme("org.mate.interface"))
		case "cinnamon", "x-cinnamon":
			themes = append(themes, gsettingsIconTheme("org.cinnamon.desktop.interface"))
		case "gnome", "ubuntu", "budgie":
			themes = append(themes, gsettingsIconTheme("org.gnome.desktop.interface"))
		case "xfce", "xfce4":
			themes = append(themes, xfconfIconTheme(), xfceConfigIconTheme())
		case "kde", "plasma":
			themes = append(themes, kdeIconTheme())
		case "lxqt":
			themes = append(themes, lxqtIconTheme())
		}
	}
	for _, desktop := range desktops {
		addForDesktop(desktop)
	}

	themes = append(themes,
		gtkIconTheme(),
		xsettingsdIconTheme(),
		kdeIconTheme(),
		lxqtIconTheme(),
		qtctIconTheme(),
		xfceConfigIconTheme(),
		xfconfIconTheme(),
	)
	if len(unique(themes)) == 0 {
		themes = append(themes,
			gsettingsIconTheme("org.mate.interface"),
			gsettingsIconTheme("org.cinnamon.desktop.interface"),
			gsettingsIconTheme("org.gnome.desktop.interface"),
		)
	}
	return unique(themes)
}

func currentDesktops() []string {
	var result []string
	for _, variable := range []string{"XDG_CURRENT_DESKTOP", "DESKTOP_SESSION", "GDMSESSION", "WINDOWMANAGER"} {
		value := os.Getenv(variable)
		if value == "" {
			continue
		}
		if variable == "WINDOWMANAGER" {
			value = filepath.Base(value)
		}
		for _, item := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ':' || r == ';' || r == ','
		}) {
			item = strings.TrimSpace(item)
			if item != "" {
				result = append(result, item)
			}
		}
	}
	return unique(result)
}

func gtkIconTheme() string {
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".config", "gtk-4.0", "settings.ini"),
		filepath.Join(os.Getenv("HOME"), ".config", "gtk-3.0", "settings.ini"),
	}
	for _, path := range paths {
		if value := iniValue(path, "Settings", "gtk-icon-theme-name"); value != "" {
			return value
		}
		if value := iniValue(path, "", "gtk-icon-theme-name"); value != "" {
			return value
		}
	}
	return ""
}

func gsettingsIconTheme(schema string) string {
	if schema == "" {
		return ""
	}
	return commandString(500*time.Millisecond, "gsettings", "get", schema, "icon-theme")
}

func xfconfIconTheme() string {
	return commandString(500*time.Millisecond, "xfconf-query", "-c", "xsettings", "-p", "/Net/IconThemeName")
}

func kdeIconTheme() string {
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".config", "kdeglobals"),
		filepath.Join(os.Getenv("HOME"), ".kde", "share", "config", "kdeglobals"),
	}
	for _, path := range paths {
		if value := iniValue(path, "Icons", "Theme"); value != "" {
			return value
		}
	}
	return ""
}

func lxqtIconTheme() string {
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".config", "lxqt", "lxqt.conf"),
		filepath.Join(os.Getenv("HOME"), ".config", "lxqt", "session.conf"),
	}
	for _, path := range paths {
		for _, section := range []string{"General", "Environment"} {
			if value := iniValue(path, section, "icon_theme"); value != "" {
				return value
			}
		}
	}
	return ""
}

func qtctIconTheme() string {
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".config", "qt5ct", "qt5ct.conf"),
		filepath.Join(os.Getenv("HOME"), ".config", "qt6ct", "qt6ct.conf"),
	}
	for _, path := range paths {
		if value := iniValue(path, "Appearance", "icon_theme"); value != "" {
			return value
		}
	}
	return ""
}

func xfceConfigIconTheme() string {
	path := filepath.Join(os.Getenv("HOME"), ".config", "xfce4", "xfconf", "xfce-perchannel-xml", "xsettings.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	type property struct {
		Name       string     `xml:"name,attr"`
		Value      string     `xml:"value,attr"`
		Properties []property `xml:"property"`
	}
	var root property
	if err := xml.Unmarshal(data, &root); err != nil {
		return ""
	}
	var find func(property) string
	find = func(prop property) string {
		if prop.Name == "IconThemeName" {
			return strings.TrimSpace(prop.Value)
		}
		for _, child := range prop.Properties {
			if value := find(child); value != "" {
				return value
			}
		}
		return ""
	}
	return find(root)
}

func xsettingsdIconTheme() string {
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".config", "xsettingsd", "xsettingsd.conf"),
		filepath.Join(os.Getenv("HOME"), ".xsettingsd"),
	}
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := stripComment(strings.TrimSpace(scanner.Text()))
			key, value, ok := strings.Cut(line, " ")
			if ok && strings.TrimSpace(key) == "Net/IconThemeName" {
				file.Close()
				return cleanSettingString(value)
			}
		}
		file.Close()
	}
	return ""
}

func iceWMIconDirs() []string {
	var result []string
	for _, path := range iceWMPreferenceFiles() {
		if value := iniValue(path, "", "IconPath"); value != "" {
			result = append(result, splitPathList(expandHome(value))...)
		}
	}
	for _, theme := range iceWMThemes() {
		for _, base := range iceWMThemeBases() {
			result = append(result, filepath.Join(base, theme, "icons"))
		}
	}
	return result
}

func iceWMPreferenceFiles() []string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return []string{
		filepath.Join(configHome, "icewm", "preferences"),
		filepath.Join(os.Getenv("HOME"), ".icewm", "preferences"),
		"/etc/icewm/preferences",
	}
}

func iceWMThemes() []string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	var result []string
	for _, path := range []string{
		filepath.Join(configHome, "icewm", "theme"),
		filepath.Join(os.Getenv("HOME"), ".icewm", "theme"),
		"/etc/icewm/theme",
	} {
		value := iniValue(path, "", "Theme")
		if value == "" {
			continue
		}
		value = strings.TrimSuffix(filepath.ToSlash(value), "/default.theme")
		value = strings.TrimSuffix(value, "/theme")
		value = strings.Trim(value, "/")
		if value != "" {
			result = append(result, value)
		}
	}
	return unique(result)
}

func iceWMThemeBases() []string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return []string{
		filepath.Join(configHome, "icewm", "themes"),
		filepath.Join(os.Getenv("HOME"), ".icewm", "themes"),
		"/usr/share/icewm/themes",
		"/usr/local/share/icewm/themes",
	}
}

func resolveIconThemes(names, roots []string) []string {
	names = unique(names)
	if len(names) == 0 {
		names = []string{"hicolor", "Adwaita"}
	}
	var result []string
	seen := make(map[string]bool)
	var visit func(string)
	visit = func(name string) {
		name = canonicalThemeName(cleanSettingString(name), roots)
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			return
		}
		seen[key] = true
		result = append(result, name)
		for _, parent := range themeInherits(name, roots) {
			visit(parent)
		}
	}
	for _, name := range names {
		visit(name)
	}
	if !seen["hicolor"] {
		visit("hicolor")
	}
	return result
}

func canonicalThemeName(name string, roots []string) string {
	if name == "" {
		return ""
	}
	for _, root := range roots {
		if isDir(filepath.Join(root, name)) {
			return name
		}
	}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() && strings.EqualFold(entry.Name(), name) {
				return entry.Name()
			}
		}
	}
	return name
}

func themeInherits(name string, roots []string) []string {
	for _, root := range roots {
		info, ok := readThemeInfo(filepath.Join(root, name))
		if ok {
			return info.inherits
		}
	}
	return nil
}

func readThemeInfo(root string) (themeInfo, bool) {
	path := filepath.Join(root, "index.theme")
	values := parseINI(path)
	if len(values) == 0 {
		return themeInfo{}, false
	}
	iconTheme := values["Icon Theme"]
	info := themeInfo{
		inherits: splitList(iconTheme["Inherits"]),
	}
	for _, rel := range splitList(iconTheme["Directories"]) {
		section := values[rel]
		dir := themeDirectory{
			rel:     rel,
			size:    parseInt(section["Size"], desiredIconSize),
			minSize: parseInt(section["MinSize"], parseInt(section["Size"], desiredIconSize)),
			maxSize: parseInt(section["MaxSize"], parseInt(section["Size"], desiredIconSize)),
			scale:   parseInt(section["Scale"], 1),
			typ:     section["Type"],
			context: section["Context"],
		}
		info.directories = append(info.directories, dir)
	}
	return info, true
}

func directoryPriority(dir themeDirectory) int {
	size := dir.size * dir.scale
	minSize := dir.minSize * dir.scale
	maxSize := dir.maxSize * dir.scale
	distance := abs(size - desiredIconSize)
	if strings.EqualFold(dir.typ, "Scalable") && minSize <= desiredIconSize && desiredIconSize <= maxSize {
		distance = 0
	}
	contextPenalty := 30
	switch strings.ToLower(dir.context) {
	case "applications", "actions":
		contextPenalty = 0
	case "status", "panel":
		contextPenalty = 4
	case "places", "categories", "devices", "mimetypes":
		contextPenalty = 8
	}
	return distance*10 + contextPenalty
}

func parseINI(path string) map[string]map[string]string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	values := make(map[string]map[string]string)
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := stripComment(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			section = strings.TrimSpace(line[1:strings.Index(line, "]")])
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = cleanSettingString(value)
		if values[section] == nil {
			values[section] = make(map[string]string)
		}
		values[section][key] = value
	}
	return values
}

func iniValue(path, section, key string) string {
	values := parseINI(path)
	if len(values) == 0 {
		return ""
	}
	if section != "" {
		return strings.TrimSpace(values[section][key])
	}
	for _, sectionValues := range values {
		if value := strings.TrimSpace(sectionValues[key]); value != "" {
			return value
		}
	}
	return ""
}

func commandString(timeout time.Duration, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	return cleanSettingString(string(output))
}

func defaultDataDirs() string {
	if value := os.Getenv("XDG_DATA_DIRS"); value != "" {
		return value
	}
	return "/usr/local/share:/usr/share"
}

func lookupKeys(name string) []string {
	name = filepath.ToSlash(strings.TrimSpace(name))
	ext := iconExtension(name)
	keys := []string{name}
	if ext != "" {
		keys = append(keys, strings.TrimSuffix(name, filepath.Ext(name)))
	}
	if strings.Contains(name, "/") {
		base := filepath.Base(name)
		keys = append(keys, base)
		if iconExtension(base) != "" {
			keys = append(keys, strings.TrimSuffix(base, filepath.Ext(base)))
		}
	}
	return unique(keys)
}

func iconExtension(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	for _, supported := range supportedExtensions {
		if ext == supported {
			return ext
		}
	}
	return ""
}

func splitList(value string) []string {
	var result []string
	for _, item := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ':'
	}) {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func splitPathList(value string) []string {
	var result []string
	for _, item := range filepath.SplitList(value) {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func cleanSettingString(value string) string {
	value = stripComment(strings.TrimSpace(value))
	value = strings.TrimSpace(strings.Trim(value, "'\""))
	return value
}

func stripComment(value string) string {
	for i, r := range value {
		if r == '#' || r == ';' {
			if i == 0 || value[i-1] != '\\' {
				return strings.TrimSpace(value[:i])
			}
		}
	}
	return value
}

func expandHome(path string) string {
	if path == "~" {
		return os.Getenv("HOME")
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(os.Getenv("HOME"), path[2:])
	}
	return path
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func skipIconDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".svn", "cursors", "preview", "previews":
		return true
	default:
		return false
	}
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

func appendIfMissing(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
