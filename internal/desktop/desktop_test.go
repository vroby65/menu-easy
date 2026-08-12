package desktop

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseLocalizedDesktopEntry(t *testing.T) {
	t.Setenv("LANGUAGE", "it_IT:it:en")
	data := `[Desktop Entry]
Type=Application
Name=Text Editor
Name[it]=Editor di testo
Comment=Edit files
Comment[it_IT]=Modifica i file
GenericName=Editor
Exec=editor --new-window %F
Icon=editor
Categories=Utility;TextEditor;
Keywords=write;text;
Terminal=false
`

	entry, err := Parse("/tmp/editor.desktop", []byte(data), []string{"MATE"})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Name != "Editor di testo" || entry.Comment != "Modifica i file" {
		t.Fatalf("localization not applied: %#v", entry)
	}
	if !reflect.DeepEqual(entry.Categories, []string{"Utility", "TextEditor"}) {
		t.Fatalf("unexpected categories: %#v", entry.Categories)
	}
	if !reflect.DeepEqual(entry.Keywords, []string{"write", "text"}) {
		t.Fatalf("unexpected keywords: %#v", entry.Keywords)
	}
}

func TestParseVisibility(t *testing.T) {
	tests := []struct {
		name    string
		extra   string
		desktop []string
		visible bool
	}{
		{"no display", "NoDisplay=true", []string{"MATE"}, false},
		{"only matching", "OnlyShowIn=MATE;GNOME;", []string{"MATE"}, true},
		{"only different", "OnlyShowIn=KDE;", []string{"MATE"}, false},
		{"excluded", "NotShowIn=MATE;", []string{"MATE"}, false},
		{"normal", "", []string{"MATE"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte("[Desktop Entry]\nType=Application\nName=Demo\nExec=demo\n" + tt.extra + "\n")
			entry, err := Parse("/tmp/demo.desktop", data, tt.desktop)
			if err != nil {
				t.Fatal(err)
			}
			if entry.Visible != tt.visible {
				t.Fatalf("Visible=%v, want %v", entry.Visible, tt.visible)
			}
		})
	}
}

func TestCommandArgsExpandsDesktopFieldCodes(t *testing.T) {
	entry := Entry{
		Name: "Editor di testo",
		Icon: "org.example.Editor",
		File: "/apps/editor.desktop",
		Exec: `editor --title "two words" %% %c %i %k %F`,
	}
	got, err := entry.CommandArgs()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"editor", "--title", "two words", "%", "Editor di testo",
		"--icon", "org.example.Editor", "/apps/editor.desktop",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CommandArgs()=%#v, want %#v", got, want)
	}
}

func TestCommandArgsRejectsMalformedQuotes(t *testing.T) {
	entry := Entry{Exec: `editor "unfinished`}
	if _, err := entry.CommandArgs(); err == nil {
		t.Fatal("expected an error")
	}
}

func TestDiscoverHonorsUserOverride(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user", "applications")
	system := filepath.Join(root, "system", "applications")
	for _, dir := range []string{user, system} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(system, "demo.desktop"), []byte("[Desktop Entry]\nType=Application\nName=System Demo\nExec=demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(user, "demo.desktop"), []byte("[Desktop Entry]\nType=Application\nName=Hidden override\nExec=demo\nHidden=true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := Discover([]string{user, system}, []string{"MATE"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("hidden user override must hide system entry: %#v", entries)
	}
}

func TestDiscoverSkipsBrokenDesktopFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "valid.desktop"), []byte("[Desktop Entry]\nType=Application\nName=Valid\nExec=valid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing.desktop"), filepath.Join(root, "broken.desktop")); err != nil {
		t.Fatal(err)
	}

	entries, err := Discover([]string{root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "Valid" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestFilterRanksPrefixAndKeywords(t *testing.T) {
	entries := []Entry{
		{ID: "calc", Name: "Calculator", Comment: "Do arithmetic"},
		{ID: "writer", Name: "Writer", Keywords: []string{"document", "text"}},
		{ID: "calendar", Name: "Calendar"},
	}
	got := Filter(entries, "cal", "All")
	if len(got) != 2 || got[0].ID != "calc" || got[1].ID != "calendar" {
		t.Fatalf("unexpected rank: %#v", got)
	}
	got = Filter(entries, "document", "All")
	if len(got) != 1 || got[0].ID != "writer" {
		t.Fatalf("keyword search failed: %#v", got)
	}
}

func TestCategory(t *testing.T) {
	tests := []struct {
		categories []string
		want       string
	}{
		{[]string{"AudioVideo", "Audio"}, "Audio e video"},
		{[]string{"Development", "IDE"}, "Sviluppo"},
		{[]string{"Game"}, "Giochi"},
		{[]string{"Network", "WebBrowser"}, "Internet"},
		{[]string{"Office"}, "Ufficio"},
		{[]string{"Settings"}, "Preferenze"},
		{[]string{"Utility"}, "Accessori"},
		{nil, "Altro"},
	}
	for _, tt := range tests {
		if got := Category(tt.categories); got != tt.want {
			t.Errorf("Category(%v)=%q, want %q", tt.categories, got, tt.want)
		}
	}
}
