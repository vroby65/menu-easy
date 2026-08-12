package windowctrl

import "image"

type panel struct {
	orientation string
	autoHide    bool
	x           int
	y           int
	size        int
}

func menuPosition(screen, work image.Rectangle, size image.Point) image.Point {
	x := work.Min.X
	y := work.Max.Y - size.Y
	if work.Min.Y > screen.Min.Y {
		y = work.Min.Y
	}
	if y < work.Min.Y {
		y = work.Min.Y
	}
	return image.Pt(x, y)
}

func menuPositionForPanel(screen image.Rectangle, p panel, size image.Point) (image.Point, bool) {
	if p.autoHide {
		return image.Point{}, false
	}
	x := p.x
	if x < screen.Min.X || x >= screen.Max.X {
		x = screen.Min.X
	}
	switch p.orientation {
	case "top":
		y := p.y + p.size
		if y < screen.Min.Y {
			y = screen.Min.Y
		}
		return image.Pt(x, y), true
	case "bottom":
		y := p.y - size.Y
		if p.y <= screen.Min.Y || p.y > screen.Max.Y {
			y = screen.Max.Y - p.size - size.Y
		}
		if y < screen.Min.Y {
			y = screen.Min.Y
		}
		return image.Pt(x, y), true
	default:
		return image.Point{}, false
	}
}
