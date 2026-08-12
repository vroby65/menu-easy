package ui

import (
	"image"
	"strings"

	"menu-easy/internal/icons"
)

// sessionIcons are simple, self-contained SVG glyphs for the session buttons.
// They are the fallback when the active theme provides no raster (PNG/XPM)
// system-log-out/reboot/shutdown icon: the SVG decoder renders many theme
// SVGs incorrectly, so SVG-only themes get these clean glyphs instead of a
// smeared icon. They look the same in every environment.
var sessionIcons = map[string]string{
	"logout": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <g fill="none" stroke="#EFF1F5" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M 3.5 4 H 10 A 2 2 0 0 1 12 6 V 18 A 2 2 0 0 1 10 20 H 3.5 A 1.5 1.5 0 0 1 2 18.5 V 5.5 A 1.5 1.5 0 0 1 3.5 4 Z"/>
    <path d="M 8.5 12 H 21"/>
    <path d="M 16 6.5 L 10.5 12 L 16 17.5"/>
  </g>
</svg>`,
	"reboot": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <g fill="none" stroke="#EFF1F5" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M 16.7 5.9 A 7.5 7.5 0 1 1 7.3 5.9"/>
    <path d="M 19 4 L 16.7 5.9 L 19 8"/>
  </g>
</svg>`,
	"shutdown": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <g fill="none" stroke="#EFF1F5" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M 16.2 5.7 A 8 8 0 1 1 7.8 5.7"/>
    <path d="M 12 3 V 10"/>
  </g>
</svg>`,
}

func sessionIconImage(key string) (image.Image, bool) {
	svg, ok := sessionIcons[key]
	if !ok {
		return nil, false
	}
	img, err := icons.DecodeSVG(strings.NewReader(svg))
	if err != nil {
		return nil, false
	}
	return img, true
}
