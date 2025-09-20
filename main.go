package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

// Define the structs to match the JSON response from the MediaWiki API.
type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

type Query struct {
	Search []SearchResult `json:"search"`
}

type Response struct {
	Query Query `json:"query"`
}

// Model holds the state of our application.
type model struct {
	textInput  textinput.Model
	results    []SearchResult
	cursor     int
	statusMsg  string
	searchType string
}

// Init initializes the application state.
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all user input and model updates.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}

		case "enter":
			if m.textInput.Value() != "" && len(m.results) == 0 {
				m.statusMsg = "Searching..."
				m.searchType = "wikipedia"
				return m, performSearch(m.textInput.Value(), m.searchType)
			} else if len(m.results) > 0 {
				selectedTitle := m.results[m.cursor].Title
				var pageURL string
				if m.searchType == "arch" {
					pageURL = "https://wiki.archlinux.org/index.php/" + strings.ReplaceAll(selectedTitle, " ", "_")
				} else {
					pageURL = "https://en.wikipedia.org/wiki/" + strings.ReplaceAll(selectedTitle, " ", "_")
				}

				var openCmd *exec.Cmd
				switch runtime.GOOS {
				case "linux":
					openCmd = exec.Command("xdg-open", pageURL)
				case "darwin": // macOS
					openCmd = exec.Command("open", pageURL)
				case "windows":
					openCmd = exec.Command("cmd", "/c", "start", pageURL)
				}
				if openCmd != nil {
					openCmd.Start()
				}
			}
		}

	case searchMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.results = msg.results
			m.statusMsg = fmt.Sprintf("Found %d results for '%s'.", len(m.results), m.textInput.Value())
			m.cursor = 0
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	s := strings.Builder{}

	s.WriteString(m.textInput.View())
	s.WriteString("\n\n")

	s.WriteString(color.New(color.FgCyan).Sprint(m.statusMsg))
	s.WriteString("\n\n")

	if len(m.results) > 0 {
		s.WriteString(color.New(color.FgCyan).Sprint("Search Results:\n"))
		for i, result := range m.results {
			var cursor string
			if i == m.cursor {
				cursor = color.New(color.Bold, color.FgGreen).Sprint("> ")
			} else {
				cursor = "  "
			}
			s.WriteString(fmt.Sprintf("%s%s\n", cursor, result.Title))
		}
	}

	s.WriteString(color.New(color.FgHiBlack).Sprint("\n\nPress enter to search or open selected, 'q' to quit."))

	return s.String()
}

type searchMsg struct {
	results []SearchResult
	err     error
}

func performSearch(term string, wikiType string) tea.Cmd {
	return func() tea.Msg {
		urlStr := "https://en.wikipedia.org/w/api.php"
		if wikiType == "arch" {
			urlStr = "https://wiki.archlinux.org/api.php"
		}

		params := url.Values{}
		params.Add("action", "query")
		params.Add("format", "json")
		params.Add("list", "search")
		params.Add("srsearch", term)

		fullURL := urlStr + "?" + params.Encode()

		// Create a new request and set the User-Agent header
		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return searchMsg{err: err}
		}
		req.Header.Set("User-Agent", "Your-CLI-Tool-Name/1.0 (Contact: your-email@example.com)")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return searchMsg{err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return searchMsg{err: fmt.Errorf("API request failed with status code: %d %s", resp.StatusCode, resp.Status)}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return searchMsg{err: err}
		}

		var data Response
		if err := json.Unmarshal(body, &data); err != nil {
			return searchMsg{err: fmt.Errorf("failed to parse API response: %w", err)}
		}

		return searchMsg{results: data.Query.Search}
	}
}

func main() {
	ti := textinput.New()
	ti.Placeholder = "Enter your search query..."
	ti.Focus()
	ti.CharLimit = 150
	ti.Width = 50

	p := tea.NewProgram(model{textInput: ti, results: []SearchResult{}})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

