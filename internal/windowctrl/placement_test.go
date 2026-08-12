package windowctrl

import (
	"image"
	"testing"
)

func TestMenuPositionBottomPanel(t *testing.T) {
	screen := image.Rect(0, 0, 1920, 1080)
	work := image.Rect(0, 0, 1920, 1040)
	size := image.Pt(860, 620)

	if got, want := menuPosition(screen, work, size), image.Pt(0, 420); got != want {
		t.Fatalf("menuPosition()=%v, want %v", got, want)
	}
}

func TestMenuPositionTopPanel(t *testing.T) {
	screen := image.Rect(0, 0, 1920, 1080)
	work := image.Rect(0, 32, 1920, 1080)
	size := image.Pt(860, 620)

	if got, want := menuPosition(screen, work, size), image.Pt(0, 32); got != want {
		t.Fatalf("menuPosition()=%v, want %v", got, want)
	}
}

func TestMenuPositionClampsToWorkarea(t *testing.T) {
	screen := image.Rect(0, 0, 800, 480)
	work := image.Rect(0, 40, 800, 480)
	size := image.Pt(860, 620)

	if got, want := menuPosition(screen, work, size), image.Pt(0, 40); got != want {
		t.Fatalf("menuPosition()=%v, want %v", got, want)
	}
}

func TestMenuPositionForMateBottomPanel(t *testing.T) {
	screen := image.Rect(0, 0, 3000, 1920)
	size := image.Pt(860, 620)
	panel := panel{orientation: "bottom", x: 0, y: 1041, size: 39}

	got, ok := menuPositionForPanel(screen, panel, size)
	if !ok {
		t.Fatal("panel position was not accepted")
	}
	if want := image.Pt(0, 421); got != want {
		t.Fatalf("menuPositionForPanel()=%v, want %v", got, want)
	}
}

func TestMenuPositionForMateTopPanel(t *testing.T) {
	screen := image.Rect(0, 0, 1920, 1080)
	size := image.Pt(860, 620)
	panel := panel{orientation: "top", x: 0, y: 0, size: 32}

	got, ok := menuPositionForPanel(screen, panel, size)
	if !ok {
		t.Fatal("panel position was not accepted")
	}
	if want := image.Pt(0, 32); got != want {
		t.Fatalf("menuPositionForPanel()=%v, want %v", got, want)
	}
}
