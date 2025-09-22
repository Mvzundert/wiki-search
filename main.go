package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"wiki-search/pkg/model"
)

func main() {
	urlRegex := regexp.MustCompile(`https?://[^\s/$.?#].[^\s]*`)

	// Initial model setup
	ti := textinput.New()
	ti.Placeholder = "Enter your search query..."
	ti.CharLimit = 150
	ti.Width = 50
	vp := viewport.New(0, 0)
	vp.YPosition = 2

	p := tea.NewProgram(model.New(ti, vp, urlRegex))

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
