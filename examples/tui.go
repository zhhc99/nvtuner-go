package main

import (
	"log"
	"nvtuner-go/internal/driver/nvidia"
	"nvtuner-go/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer f.Close()

	drv, err := nvidia.New()
	if err != nil {
		log.Fatalf("Failed to load driver: %v", err)
	}
	if err := drv.Init(); err != nil {
		log.Fatalf("Failed to init driver: %v", err)
	}
	defer drv.Shutdown()

	model, err := ui.NewModel(drv)
	if err != nil {
		log.Fatalf("Failed to create UI model: %v", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
