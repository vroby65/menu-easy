package desktop

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const AllCategory = "Tutte"

var Categories = []string{
	AllCategory,
	"Preferiti",
	"Accessori",
	"Sviluppo",
	"Istruzione",
	"Giochi",
	"Grafica",
	"Internet",
	"Audio e video",
	"Ufficio",
	"Preferenze",
	"Sistema",
	"Altro",
}

// Entry is an application described by a freedesktop .desktop file.
type Entry struct {
	ID          string
	Name        string
	GenericName string
	Comment     string
	Icon        string
	Exec        string
	File        string
	Path        string
	Categories  []string
	Keywords    []string
	Terminal    bool
	Visible     bool
	Hidden      bool
	TryExec     string
}

// Parse reads the [Desktop Entry] group. Visibility is evaluated for the
// supplied desktop names, such as MATE, X-Cinnamon or KDE.
func Parse(path string, data []byte, desktops []string) (Entry, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inDesktopEntry := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inDesktopEntry = line == "[Desktop Entry]"
			continue
		}
		if !inDesktopEntry {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return Entry{}, err
	}
	if values["Type"] != "Application" {
		return Entry{}, errors.New("not an application desktop entry")
	}

	locales := preferredLocales()
	entry := Entry{
		Name:        localized(values, "Name", locales),
		GenericName: localized(values, "GenericName", locales),
		Comment:     localized(values, "Comment", locales),
		Icon:        unescapeValue(values["Icon"]),
		Exec:        values["Exec"],
		File:        path,
		Path:        unescapeValue(values["Path"]),
		Categories:  splitList(values["Categories"]),
		Keywords:    splitList(localized(values, "Keywords", locales)),
		Terminal:    parseBool(values["Terminal"]),
		Hidden:      parseBool(values["Hidden"]),
		TryExec:     unescapeValue(values["TryExec"]),
	}
	if entry.Name == "" || entry.Exec == "" {
		return Entry{}, errors.New("desktop entry requires Name and Exec")
	}
	entry.Visible = !entry.Hidden && !parseBool(values["NoDisplay"])
	if entry.Visible && len(desktops) > 0 {
		only := splitList(values["OnlyShowIn"])
		not := splitList(values["NotShowIn"])
		if len(only) > 0 && !intersectsFold(only, desktops) {
			entry.Visible = false
		}
		if intersectsFold(not, desktops) {
			entry.Visible = false
		}
	}
	return entry, nil
}

// Discover reads application files in priority order. An earlier entry with
// the same desktop-file ID shadows later entries, even when it is hidden.
func Discover(dirs []string, desktops []string) ([]Entry, error) {
	seen := make(map[string]bool)
	var entries []Entry
	for _, root := range dirs {
		info, err := os.Stat(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".desktop") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			id := strings.ReplaceAll(filepath.ToSlash(rel), "/", "-")
			if seen[id] {
				return nil
			}
			seen[id] = true
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			entry, err := Parse(path, data, desktops)
			if err != nil || !entry.Visible {
				return nil
			}
			if entry.TryExec != "" {
				if _, err := exec.LookPath(entry.TryExec); err != nil {
					return nil
				}
			}
			entry.ID = id
			entries = append(entries, entry)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

// ApplicationDirs returns the XDG application directories in override order.
func ApplicationDirs() []string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	dirs := []string{filepath.Join(dataHome, "applications")}
	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}
	for _, dir := range filepath.SplitList(dataDirs) {
		if dir != "" {
			dirs = append(dirs, filepath.Join(dir, "applications"))
		}
	}
	return dirs
}

// CurrentDesktops returns XDG_CURRENT_DESKTOP as a list.
func CurrentDesktops() []string {
	var result []string
	for _, value := range filepath.SplitList(os.Getenv("XDG_CURRENT_DESKTOP")) {
		for _, desktop := range strings.Split(value, ":") {
			if desktop = strings.TrimSpace(desktop); desktop != "" {
				result = append(result, desktop)
			}
		}
	}
	return result
}

// Category maps freedesktop categories to a compact launcher category.
func Category(categories []string) string {
	set := make(map[string]bool, len(categories))
	for _, category := range categories {
		set[category] = true
	}
	switch {
	case set["AudioVideo"] || set["Audio"] || set["Video"]:
		return "Audio e video"
	case set["Development"]:
		return "Sviluppo"
	case set["Education"] || set["Science"]:
		return "Istruzione"
	case set["Game"]:
		return "Giochi"
	case set["Graphics"]:
		return "Grafica"
	case set["Network"]:
		return "Internet"
	case set["Office"]:
		return "Ufficio"
	case set["Settings"] || set["DesktopSettings"]:
		return "Preferenze"
	case set["System"]:
		return "Sistema"
	case set["Utility"] || set["Accessories"]:
		return "Accessori"
	default:
		return "Altro"
	}
}

// Filter returns entries matching a query and category. Search terms must all
// occur in the name, generic name, comment or keywords.
func Filter(entries []Entry, query, category string) []Entry {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	type match struct {
		entry Entry
		score int
	}
	matches := make([]match, 0, len(entries))
	for _, entry := range entries {
		if category != "" && category != "All" && category != AllCategory && Category(entry.Categories) != category {
			continue
		}
		name := strings.ToLower(entry.Name)
		haystack := strings.ToLower(strings.Join([]string{
			entry.Name, entry.GenericName, entry.Comment, strings.Join(entry.Keywords, " "),
		}, " "))
		score := 0
		matched := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				matched = false
				break
			}
			switch {
			case name == term:
				score += 100
			case strings.HasPrefix(name, term):
				score += 80
			case hasWordPrefix(name, term):
				score += 60
			default:
				score += 40
			}
		}
		if matched {
			matches = append(matches, match{entry: entry, score: score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return strings.ToLower(matches[i].entry.Name) < strings.ToLower(matches[j].entry.Name)
	})
	result := make([]Entry, len(matches))
	for i := range matches {
		result[i] = matches[i].entry
	}
	return result
}

// CommandArgs converts the desktop Exec value into argv without invoking a
// shell. File and URL placeholders are omitted because the launcher has no
// document argument.
func (e Entry) CommandArgs() ([]string, error) {
	words, err := splitExec(e.Exec)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, word := range words {
		if word == "%i" {
			if e.Icon != "" {
				result = append(result, "--icon", e.Icon)
			}
			continue
		}
		var expanded strings.Builder
		for i := 0; i < len(word); i++ {
			if word[i] != '%' {
				expanded.WriteByte(word[i])
				continue
			}
			if i+1 >= len(word) {
				return nil, errors.New("dangling % in Exec")
			}
			i++
			switch word[i] {
			case '%':
				expanded.WriteByte('%')
			case 'c':
				expanded.WriteString(e.Name)
			case 'k':
				expanded.WriteString(e.File)
			case 'f', 'F', 'u', 'U', 'd', 'D', 'n', 'N', 'v', 'm':
				// No file or URL was supplied; deprecated codes are ignored too.
			case 'i':
				return nil, errors.New("%i must be a separate argument")
			default:
				return nil, fmt.Errorf("unsupported field code %%%c", word[i])
			}
		}
		if expanded.Len() > 0 {
			result = append(result, expanded.String())
		}
	}
	if len(result) == 0 || result[0] == "" {
		return nil, errors.New("Exec produced an empty command")
	}
	return result, nil
}

func localized(values map[string]string, key string, locales []string) string {
	for _, locale := range locales {
		if value := values[key+"["+locale+"]"]; value != "" {
			return unescapeValue(value)
		}
	}
	return unescapeValue(values[key])
}

func preferredLocales() []string {
	var raw []string
	if language := os.Getenv("LANGUAGE"); language != "" {
		raw = append(raw, strings.Split(language, ":")...)
	}
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(name); value != "" {
			raw = append(raw, value)
			break
		}
	}
	seen := make(map[string]bool)
	var result []string
	add := func(locale string) {
		locale = strings.TrimSpace(locale)
		if locale != "" && locale != "C" && locale != "POSIX" && !seen[locale] {
			seen[locale] = true
			result = append(result, locale)
		}
	}
	for _, locale := range raw {
		locale, _, _ = strings.Cut(locale, ".")
		add(locale)
		base, modifier, hasModifier := strings.Cut(locale, "@")
		add(base)
		language, _, hasTerritory := strings.Cut(base, "_")
		if hasTerritory && hasModifier {
			add(language + "@" + modifier)
		}
		if hasTerritory {
			add(language)
		}
	}
	return result
}

func splitList(value string) []string {
	parts := strings.Split(value, ";")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = unescapeValue(strings.TrimSpace(part)); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func parseBool(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func intersectsFold(a, b []string) bool {
	for _, left := range a {
		for _, right := range b {
			if strings.EqualFold(left, right) {
				return true
			}
		}
	}
	return false
}

func unescapeValue(value string) string {
	var result strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			result.WriteByte(value[i])
			continue
		}
		i++
		switch value[i] {
		case 's':
			result.WriteByte(' ')
		case 'n':
			result.WriteByte('\n')
		case 't':
			result.WriteByte('\t')
		case 'r':
			result.WriteByte('\r')
		case '\\':
			result.WriteByte('\\')
		default:
			result.WriteByte(value[i])
		}
	}
	return result.String()
}

func hasWordPrefix(value, prefix string) bool {
	for _, field := range strings.FieldsFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if strings.HasPrefix(field, prefix) {
			return true
		}
	}
	return false
}

func splitExec(value string) ([]string, error) {
	var words []string
	var word strings.Builder
	inDouble := false
	inSingle := false
	escaped := false
	haveWord := false
	for _, r := range value {
		if escaped {
			word.WriteRune(r)
			haveWord = true
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			haveWord = true
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			haveWord = true
			continue
		}
		if r == '\'' && !inDouble {
			inSingle = !inSingle
			haveWord = true
			continue
		}
		if unicode.IsSpace(r) && !inDouble && !inSingle {
			if haveWord {
				words = append(words, word.String())
				word.Reset()
				haveWord = false
			}
			continue
		}
		word.WriteRune(r)
		haveWord = true
	}
	if escaped || inDouble || inSingle {
		return nil, errors.New("malformed quoting in Exec")
	}
	if haveWord {
		words = append(words, word.String())
	}
	return words, nil
}
