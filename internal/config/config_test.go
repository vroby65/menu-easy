package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestFavoritesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "menu-easy", "config.json")
	cfg := Config{Favorites: []string{"editor.desktop", "browser.desktop"}}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, cfg) {
		t.Fatalf("Load()=%#v, want %#v", got, cfg)
	}
}

func TestToggleFavorite(t *testing.T) {
	cfg := Config{}
	if favorite := cfg.Toggle("editor.desktop"); !favorite {
		t.Fatal("entry should have been added")
	}
	if !cfg.IsFavorite("editor.desktop") {
		t.Fatal("entry is not marked as favorite")
	}
	if favorite := cfg.Toggle("editor.desktop"); favorite {
		t.Fatal("entry should have been removed")
	}
	if len(cfg.Favorites) != 0 {
		t.Fatalf("unexpected favorites: %#v", cfg.Favorites)
	}
}
