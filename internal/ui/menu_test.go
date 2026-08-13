package ui

import (
	"testing"

	"menu-easy/internal/config"
)

func TestNewDoesNotWarmIconLoader(t *testing.T) {
	menu := New(nil, nil, config.Config{}, "")

	if menu.iconLoader != nil {
		t.Fatal("New initialized the icon loader before the first frame")
	}
	if menu.iconWarmupStart {
		t.Fatal("New started icon warmup before the first frame")
	}
	if menu.iconsReady {
		t.Fatal("New marked icons as ready before warmup")
	}
}
