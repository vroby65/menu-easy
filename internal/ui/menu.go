package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/semantic"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"menu-easy/internal/config"
	"menu-easy/internal/desktop"
	appicons "menu-easy/internal/icons"
	"menu-easy/internal/launch"
)

var palette = struct {
	background color.NRGBA
	header     color.NRGBA
	sidebar    color.NRGBA
	panel      color.NRGBA
	row        color.NRGBA
	hover      color.NRGBA
	selected   color.NRGBA
	accent     color.NRGBA
	text       color.NRGBA
	muted      color.NRGBA
	error      color.NRGBA
}{
	background: color.NRGBA{R: 27, G: 29, B: 35, A: 255},
	header:     color.NRGBA{R: 30, G: 33, B: 40, A: 255},
	sidebar:    color.NRGBA{R: 35, G: 38, B: 46, A: 255},
	panel:      color.NRGBA{R: 27, G: 29, B: 35, A: 255},
	row:        color.NRGBA{R: 35, G: 38, B: 46, A: 255},
	hover:      color.NRGBA{R: 46, G: 51, B: 61, A: 255},
	selected:   color.NRGBA{R: 50, G: 66, B: 59, A: 255},
	accent:     color.NRGBA{R: 112, G: 184, B: 110, A: 255},
	text:       color.NRGBA{R: 239, G: 241, B: 245, A: 255},
	muted:      color.NRGBA{R: 162, G: 168, B: 181, A: 255},
	error:      color.NRGBA{R: 244, G: 128, B: 128, A: 255},
}

type rowControls struct {
	launch   widget.Clickable
	favorite widget.Clickable
}

type cachedImage struct {
	image widget.Image
	found bool
}

type Menu struct {
	window     *app.Window
	theme      *material.Theme
	entries    []desktop.Entry
	config     config.Config
	configPath string

	search       widget.Editor
	applications widget.List
	categories   widget.List
	category     string
	selected     int
	lastQuery    string
	lastCategory string
	status       string
	firstFrame   bool

	categoryButtons map[string]*widget.Clickable
	rows            map[string]*rowControls
	iconLoader      *appicons.Loader
	images          map[string]cachedImage
}

func New(window *app.Window, entries []desktop.Entry, cfg config.Config, configPath string) *Menu {
	theme := material.NewTheme()
	theme.Palette = material.Palette{
		Bg:         palette.background,
		Fg:         palette.text,
		ContrastBg: palette.accent,
		ContrastFg: color.NRGBA{R: 20, G: 30, B: 23, A: 255},
	}
	menu := &Menu{
		window:          window,
		theme:           theme,
		entries:         entries,
		config:          cfg,
		configPath:      configPath,
		category:        desktop.AllCategory,
		firstFrame:      true,
		categoryButtons: make(map[string]*widget.Clickable),
		rows:            make(map[string]*rowControls),
		iconLoader:      appicons.NewLoader(),
		images:          make(map[string]cachedImage),
	}
	menu.search.SingleLine = true
	menu.search.Submit = true
	menu.search.MaxLen = 120
	menu.applications.List.Axis = layout.Vertical
	menu.categories.List.Axis = layout.Vertical
	for _, category := range desktop.Categories {
		menu.categoryButtons[category] = new(widget.Clickable)
	}
	for _, entry := range entries {
		menu.rows[entry.ID] = new(rowControls)
	}
	return menu
}

func Run(window *app.Window, menu *Menu) error {
	var ops op.Ops
	hadFocus := false
	for {
		switch event := window.Event().(type) {
		case app.DestroyEvent:
			return event.Err
		case app.ConfigEvent:
			if event.Config.Focused {
				hadFocus = true
			} else if hadFocus {
				window.Perform(system.ActionClose)
			}
		case app.FrameEvent:
			gtx := app.NewContext(&ops, event)
			menu.Layout(gtx)
			event.Frame(gtx.Ops)
		}
	}
}

func (m *Menu) Layout(gtx layout.Context) layout.Dimensions {
	fillBackground(gtx, palette.background)

	for _, category := range desktop.Categories {
		if m.categoryButtons[category].Clicked(gtx) {
			m.category = category
			m.selected = 0
			m.applications.ScrollTo(0)
		}
	}
	m.handleKeys(gtx, len(m.visibleEntries(m.search.Text())))

	submit := false
	for {
		event, ok := m.search.Update(gtx)
		if !ok {
			break
		}
		if _, ok := event.(widget.SubmitEvent); ok {
			submit = true
		}
	}

	query := m.search.Text()
	if query != m.lastQuery || m.category != m.lastCategory {
		m.selected = 0
		m.applications.ScrollTo(0)
		m.lastQuery = query
		m.lastCategory = m.category
	}
	visible := m.visibleEntries(query)

	favoritesChanged := false
	for _, entry := range visible {
		controls := m.rows[entry.ID]
		if controls.launch.Clicked(gtx) {
			m.start(entry)
		}
		if controls.favorite.Clicked(gtx) {
			favorite := m.config.Toggle(entry.ID)
			if err := config.Save(m.configPath, m.config); err != nil {
				m.status = "Impossibile salvare i preferiti: " + err.Error()
			} else if favorite {
				m.status = entry.Name + " aggiunta ai preferiti"
			} else {
				m.status = entry.Name + " rimossa dai preferiti"
			}
			favoritesChanged = true
		}
	}
	if favoritesChanged {
		visible = m.visibleEntries(query)
	}
	if len(visible) == 0 {
		m.selected = 0
	} else if m.selected >= len(visible) {
		m.selected = len(visible) - 1
	}
	if submit && len(visible) > 0 {
		m.start(visible[m.selected])
	}

	if m.firstFrame {
		gtx.Execute(key.FocusCmd{Tag: &m.search})
		m.firstFrame = false
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return exactHeight(gtx, 92, m.layoutHeader)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return m.layoutBody(gtx, visible)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return exactHeight(gtx, 34, m.layoutFooter)
		}),
	)
}

func (m *Menu) handleKeys(gtx layout.Context, count int) {
	filters := []key.Filter{
		{Name: key.NameEscape},
		{Name: key.NameDownArrow},
		{Name: key.NameUpArrow},
	}
	for {
		event, ok := gtx.Event(filters[0], filters[1], filters[2])
		if !ok {
			break
		}
		keyEvent, ok := event.(key.Event)
		if !ok || keyEvent.State != key.Press {
			continue
		}
		switch keyEvent.Name {
		case key.NameEscape:
			m.window.Perform(system.ActionClose)
		case key.NameDownArrow:
			if m.selected+1 < count {
				m.selected++
				m.applications.ScrollTo(m.selected)
			}
		case key.NameUpArrow:
			if m.selected > 0 {
				m.selected--
				m.applications.ScrollTo(m.selected)
			}
		}
	}
}

func (m *Menu) visibleEntries(query string) []desktop.Entry {
	entries := desktop.Filter(m.entries, query, mapCategory(m.category))
	if m.category != "Preferiti" {
		return entries
	}
	result := entries[:0]
	for _, entry := range entries {
		if m.config.IsFavorite(entry.ID) {
			result = append(result, entry)
		}
	}
	return result
}

func mapCategory(category string) string {
	if category == "Preferiti" {
		return desktop.AllCategory
	}
	return category
}

func (m *Menu) start(entry desktop.Entry) {
	if err := launch.Start(entry); err != nil {
		m.status = "Avvio non riuscito: " + err.Error()
		return
	}
	m.window.Perform(system.ActionClose)
}

func (m *Menu) layoutHeader(gtx layout.Context) layout.Dimensions {
	fillBackground(gtx, palette.header)
	return layout.Inset{Top: 18, Bottom: 18, Left: 20, Right: 20}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return exactSize(gtx, 54, 54, m.layoutLogo)
			}),
			layout.Rigid(layout.Spacer{Width: 14}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H5(m.theme, "Menu Easy")
						title.Color = palette.text
						title.TextSize = 22
						return title.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						caption := material.Caption(m.theme, fmt.Sprintf("%d applicazioni", len(m.entries)))
						caption.Color = palette.muted
						return caption.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				width := unit.Dp(350)
				if gtx.Constraints.Max.X < gtx.Dp(width) {
					width = unit.Dp(float32(gtx.Constraints.Max.X) / gtx.Metric.PxPerDp)
				}
				return exactWidth(gtx, width, m.layoutSearch)
			}),
		)
	})
}

func (m *Menu) layoutLogo(gtx layout.Context) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return roundedBackground(gtx, palette.accent, 13)
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Label(m.theme, 19, "ME")
				label.Color = color.NRGBA{R: 22, G: 39, B: 27, A: 255}
				return label.Layout(gtx)
			})
		},
	)
}

func (m *Menu) layoutSearch(gtx layout.Context) layout.Dimensions {
	return exactHeight(gtx, 46, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return roundedBackground(gtx, palette.row, 11)
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: 15, Right: 15, Top: 11, Bottom: 9}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					editor := material.Editor(m.theme, &m.search, "Cerca applicazioni…")
					editor.Color = palette.text
					editor.HintColor = palette.muted
					editor.TextSize = 16
					return editor.Layout(gtx)
				})
			},
		)
	})
}

func (m *Menu) layoutBody(gtx layout.Context, visible []desktop.Entry) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return exactWidth(gtx, 208, m.layoutSidebar)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return m.layoutApplications(gtx, visible)
		}),
	)
}

func (m *Menu) layoutSidebar(gtx layout.Context) layout.Dimensions {
	fillBackground(gtx, palette.sidebar)
	return layout.Inset{Top: 15, Bottom: 10, Left: 11, Right: 11}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: 10, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					label := material.Caption(m.theme, "CATEGORIE")
					label.Color = palette.muted
					label.TextSize = 12
					return label.Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return m.categories.Layout(gtx, len(desktop.Categories), func(gtx layout.Context, index int) layout.Dimensions {
					category := desktop.Categories[index]
					return m.layoutCategory(gtx, category)
				})
			}),
		)
	})
}

func (m *Menu) layoutCategory(gtx layout.Context, category string) layout.Dimensions {
	return layout.Inset{Bottom: 3}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return exactHeight(gtx, 37, func(gtx layout.Context) layout.Dimensions {
			button := m.categoryButtons[category]
			return button.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				semantic.Button.Add(gtx.Ops)
				background := color.NRGBA{}
				if category == m.category {
					background = palette.selected
				} else if button.Hovered() {
					background = palette.hover
				}
				return layout.Background{}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return roundedBackground(gtx, background, 8)
					},
					func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: 11, Right: 8, Top: 8, Bottom: 7}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							label := material.Body2(m.theme, category)
							label.Color = palette.text
							label.MaxLines = 1
							return label.Layout(gtx)
						})
					},
				)
			})
		})
	})
}

func (m *Menu) layoutApplications(gtx layout.Context, visible []desktop.Entry) layout.Dimensions {
	fillBackground(gtx, palette.panel)
	return layout.Inset{Top: 14, Bottom: 8, Left: 18, Right: 18}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return exactHeight(gtx, 42, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							label := material.H6(m.theme, m.category)
							label.Color = palette.text
							label.TextSize = 18
							return label.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Caption(m.theme, fmt.Sprintf("%d risultati", len(visible)))
							label.Color = palette.muted
							return label.Layout(gtx)
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(visible) == 0 {
					return m.layoutEmpty(gtx)
				}
				return m.applications.Layout(gtx, len(visible), func(gtx layout.Context, index int) layout.Dimensions {
					return m.layoutApplication(gtx, visible[index], index == m.selected)
				})
			}),
		)
	})
}

func (m *Menu) layoutEmpty(gtx layout.Context) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.H6(m.theme, "Nessuna applicazione trovata")
				label.Color = palette.muted
				label.TextSize = 17
				return label.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: 4}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.Caption(m.theme, "Prova una ricerca o una categoria diversa")
				label.Color = palette.muted
				return label.Layout(gtx)
			}),
		)
	})
}

func (m *Menu) layoutApplication(gtx layout.Context, entry desktop.Entry, selected bool) layout.Dimensions {
	return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return exactHeight(gtx, 64, func(gtx layout.Context) layout.Dimensions {
			controls := m.rows[entry.ID]
			background := palette.row
			if selected {
				background = palette.selected
			} else if controls.launch.Hovered() || controls.favorite.Hovered() {
				background = palette.hover
			}
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return roundedBackground(gtx, background, 10)
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return controls.launch.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								semantic.Button.Add(gtx.Ops)
								return layout.Inset{Left: 10, Top: 9, Bottom: 9}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return exactSize(gtx, 46, 46, func(gtx layout.Context) layout.Dimensions {
												return m.layoutAppIcon(gtx, entry)
											})
										}),
										layout.Rigid(layout.Spacer{Width: 12}.Layout),
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											return m.layoutAppText(gtx, entry)
										}),
									)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return exactWidth(gtx, 50, func(gtx layout.Context) layout.Dimensions {
								return controls.favorite.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									semantic.Button.Add(gtx.Ops)
									return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										star := "☆"
										if m.config.IsFavorite(entry.ID) {
											star = "★"
										}
										label := material.Label(m.theme, 22, star)
										label.Color = palette.accent
										return label.Layout(gtx)
									})
								})
							})
						}),
					)
				},
			)
		})
	})
}

func (m *Menu) layoutAppIcon(gtx layout.Context, entry desktop.Entry) layout.Dimensions {
	if icon := m.appImage(entry); icon != nil {
		return layout.Inset{Top: 3, Bottom: 3, Left: 3, Right: 3}.Layout(gtx, icon.Layout)
	}
	letter := "?"
	if runes := []rune(strings.TrimSpace(entry.Name)); len(runes) > 0 {
		letter = strings.ToUpper(string(runes[0]))
	}
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return roundedBackground(gtx, palette.accent, 10)
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Label(m.theme, 18, letter)
				label.Color = color.NRGBA{R: 20, G: 35, B: 24, A: 255}
				return label.Layout(gtx)
			})
		},
	)
}

func (m *Menu) appImage(entry desktop.Entry) *widget.Image {
	if cached, ok := m.images[entry.ID]; ok {
		if cached.found {
			return &cached.image
		}
		return nil
	}
	img, found := m.iconLoader.Load(entry.Icon)
	if !found {
		m.images[entry.ID] = cachedImage{}
		return nil
	}
	value := cachedImage{
		found: true,
		image: widget.Image{
			Src:      paint.NewImageOp(img),
			Fit:      widget.Contain,
			Position: layout.Center,
		},
	}
	m.images[entry.ID] = value
	return &value.image
}

func (m *Menu) layoutAppText(gtx layout.Context, entry desktop.Entry) layout.Dimensions {
	subtitle := entry.Comment
	if subtitle == "" {
		subtitle = entry.GenericName
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Body1(m.theme, entry.Name)
			label.Color = palette.text
			label.TextSize = 15
			label.MaxLines = 1
			return label.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 2}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Caption(m.theme, subtitle)
			label.Color = palette.muted
			label.TextSize = 12
			label.MaxLines = 1
			return label.Layout(gtx)
		}),
	)
}

func (m *Menu) layoutFooter(gtx layout.Context) layout.Dimensions {
	fillBackground(gtx, palette.header)
	return layout.Inset{Left: 16, Right: 16, Top: 8, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				text := "↑ ↓ seleziona  ·  Invio avvia  ·  Esc chiude"
				color := palette.muted
				if m.status != "" {
					text = m.status
					color = palette.error
				}
				label := material.Caption(m.theme, text)
				label.Color = color
				label.MaxLines = 1
				return label.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.Caption(m.theme, "Xorg · Wayland")
				label.Color = palette.muted
				return label.Layout(gtx)
			}),
		)
	})
}

func roundedBackground(gtx layout.Context, background color.NRGBA, radius unit.Dp) layout.Dimensions {
	size := gtx.Constraints.Min
	defer clip.UniformRRect(image.Rectangle{Max: size}, gtx.Dp(radius)).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, background)
	return layout.Dimensions{Size: size}
}

func fillBackground(gtx layout.Context, background color.NRGBA) {
	defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, background)
}

func exactHeight(gtx layout.Context, height unit.Dp, child layout.Widget) layout.Dimensions {
	pixels := gtx.Dp(height)
	if pixels > gtx.Constraints.Max.Y {
		pixels = gtx.Constraints.Max.Y
	}
	gtx.Constraints.Min.Y = pixels
	gtx.Constraints.Max.Y = pixels
	return child(gtx)
}

func exactWidth(gtx layout.Context, width unit.Dp, child layout.Widget) layout.Dimensions {
	pixels := gtx.Dp(width)
	if pixels > gtx.Constraints.Max.X {
		pixels = gtx.Constraints.Max.X
	}
	gtx.Constraints.Min.X = pixels
	gtx.Constraints.Max.X = pixels
	return child(gtx)
}

func exactSize(gtx layout.Context, width, height unit.Dp, child layout.Widget) layout.Dimensions {
	widthPixels := gtx.Dp(width)
	heightPixels := gtx.Dp(height)
	if widthPixels > gtx.Constraints.Max.X {
		widthPixels = gtx.Constraints.Max.X
	}
	if heightPixels > gtx.Constraints.Max.Y {
		heightPixels = gtx.Constraints.Max.Y
	}
	gtx.Constraints.Min = image.Pt(widthPixels, heightPixels)
	gtx.Constraints.Max = gtx.Constraints.Min
	return child(gtx)
}
