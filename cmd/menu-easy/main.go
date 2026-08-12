package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/unit"

	"menu-easy/internal/config"
	"menu-easy/internal/desktop"
	"menu-easy/internal/ui"
)

func main() {
	entries, err := desktop.Discover(desktop.ApplicationDirs(), desktop.CurrentDesktops())
	if err != nil {
		log.Fatal(err)
	}
	configPath, err := config.Path()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("configurazione ignorata: %v", err)
		cfg = config.Config{}
	}

	go func() {
		window := new(app.Window)
		window.Option(
			app.Title("Menu Easy"),
			app.Size(unit.Dp(860), unit.Dp(620)),
			app.MinSize(unit.Dp(680), unit.Dp(480)),
			app.Decorated(false),
			app.TopMost(true),
		)
		menu := ui.New(window, entries, cfg, configPath)
		if err := ui.Run(window, menu); err != nil {
			log.Print(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
