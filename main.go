package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/go-shiori/go-readability"
)

// SearchResult matches the JSON response from the MediaWiki search API.
type SearchResult struct {
	Title string `json:"title"`
}

// ArticleResponse matches the JSON response from the MediaWiki parse API.
type ArticleResponse struct {
	Parse struct {
		Text struct {
			Content string `json:"*"`
		} `json:"text"`
	} `json:"parse"`
}

// Query is for the search API.
type Query struct {
	Search []SearchResult `json:"search"`
}

// Response is for the search API.
type Response struct {
	Query Query `json:"query"`
}

// State represents the current view of the application.
type state int

const (
	wikiSelectionView state = iota
	searchResultsView
	articleView
)

// Model holds the state of our application.
type model struct {
	state          state
	textInput      textinput.Model
	results        []SearchResult
	cursor         int
	statusMsg      string
	selectedTitle  string
	articleContent string
	searchType     string
	wikiOptions    []string
	wikiCursor     int
	viewport       viewport.Model // Add viewport for scrolling
}

// Init initializes the application state.
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all user input and model updates.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var vpCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4 // Account for header and footer
		m.viewport.SetContent(m.articleContent)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			switch m.state {
			case articleView:
				m.state = searchResultsView
				m.articleContent = ""
				m.textInput.Focus()
				return m, nil
			case searchResultsView:
				m.state = wikiSelectionView
				m.textInput.Blur()
				return m, nil
			}
			return m, tea.Quit

		case "up", "k":
			switch m.state {
			case searchResultsView:
				if m.cursor > 0 {
					m.cursor--
				}
			case wikiSelectionView:
				if m.wikiCursor > 0 {
					m.wikiCursor--
				}
			case articleView:
				// Pass to viewport for scrolling
				m.viewport, vpCmd = m.viewport.Update(msg)
				return m, vpCmd
			}

		case "down", "j":
			switch m.state {
			case searchResultsView:
				if m.cursor < len(m.results)-1 {
					m.cursor++
				}
			case wikiSelectionView:
				if m.wikiCursor < len(m.wikiOptions)-1 {
					m.wikiCursor++
				}
			case articleView:
				// Pass to viewport for scrolling
				m.viewport, vpCmd = m.viewport.Update(msg)
				return m, vpCmd
			}

		case "o":
			if m.state == searchResultsView && len(m.results) > 0 {
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
				return m, tea.Quit
			}

		case "enter":
			if m.state == wikiSelectionView {
				m.searchType = m.wikiOptions[m.wikiCursor]
				m.state = searchResultsView
				m.textInput.Focus()
				return m, nil
			} else if m.textInput.Focused() {
				if m.textInput.Value() != "" {
					m.statusMsg = "Searching..."
					m.textInput.Blur()
					return m, performSearch(m.textInput.Value(), m.searchType)
				}
			} else if m.state == searchResultsView && len(m.results) > 0 {
				m.selectedTitle = m.results[m.cursor].Title
				m.statusMsg = "Fetching article..."
				return m, fetchArticle(m.selectedTitle, m.searchType)
			}
		}

	case searchMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
			m.textInput.Focus()
		} else {
			m.results = msg.results
			m.statusMsg = fmt.Sprintf("Found %d results for '%s'. Press Enter to select one.", len(m.results), m.textInput.Value())
			m.cursor = 0
		}

	case articleMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.state = articleView
			m.articleContent = msg.content
			m.statusMsg = fmt.Sprintf("Displaying article: %s", m.selectedTitle)
			m.viewport.SetContent(m.articleContent)
		}
	}

	// Only update text input if it is focused
	if m.textInput.Focused() {
		m.textInput, cmd = m.textInput.Update(msg)
	}

	return m, cmd
}

// View renders the UI to the terminal.
func (m model) View() string {
	s := strings.Builder{}

	switch m.state {
	case wikiSelectionView:
		s.WriteString("Select a Wiki to Search:\n\n")
		for i, wiki := range m.wikiOptions {
			cursor := " "
			if i == m.wikiCursor {
				cursor = color.New(color.Bold, color.FgGreen).Sprint(">")
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, wiki))
		}
		s.WriteString(color.New(color.FgHiBlack).Sprint("\n\nPress Enter to select, 'q' to quit."))
	case searchResultsView:
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

		s.WriteString(color.New(color.FgHiBlack).Sprint("\n\nEnter to search/select, Up/Down to navigate, 'o' to open in browser, 'q' to quit."))

	case articleView:
		s.WriteString(color.New(color.Bold, color.FgCyan).Sprint(m.selectedTitle))
		s.WriteString("\n\n")
		s.WriteString(m.viewport.View())
		s.WriteString(color.New(color.FgHiBlack).Sprint("\n\nPress 'esc' to go back to results, Up/Down to scroll, 'q' to quit."))
	}

	return s.String()
}

// Custom messages to pass data between functions.
type searchMsg struct {
	results []SearchResult
	err     error
}
type articleMsg struct {
	content string
	err     error
}

// performSearch is a command that makes the API call.
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

		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return searchMsg{err: err}
		}
		req.Header.Set("User-Agent", "Your-CLI-Tool-Name/1.0 (Contact: your-email@example.com)")

		// Set a timeout for the request
		client := &http.Client{Timeout: 5 * time.Second}
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

// fetchArticle fetches the full article content.
func fetchArticle(title string, wikiType string) tea.Cmd {
	return func() tea.Msg {
		urlStr := "https://en.wikipedia.org/w/api.php"
		if wikiType == "arch" {
			urlStr = "https://wiki.archlinux.org/api.php"
		}

		params := url.Values{}
		params.Add("action", "parse")
		params.Add("format", "json")
		params.Add("page", title)

		fullURL := urlStr + "?" + params.Encode()

		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return articleMsg{err: err}
		}
		req.Header.Set("User-Agent", "Your-CLI-Tool-Name/1.0 (Contact: your-email@example.com)")

		// Set a timeout for the request
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return articleMsg{err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return articleMsg{err: fmt.Errorf("API request failed with status code: %d %s", resp.StatusCode, resp.Status)}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return articleMsg{err: err}
		}

		var data ArticleResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return articleMsg{err: fmt.Errorf("failed to parse article response: %w", err)}
		}

		// Parse the fullURL string into a url.URL object
		parsedURL, err := url.Parse(fullURL)
		if err != nil {
			return articleMsg{err: fmt.Errorf("failed to parse URL: %w", err)}
		}

		// Use the go-readability library to convert the HTML content to plain text.
		contentReader := bytes.NewReader([]byte(data.Parse.Text.Content))
		article, err := readability.FromReader(contentReader, parsedURL)
		if err != nil {
			return articleMsg{err: fmt.Errorf("failed to make content readable: %w", err)}
		}

		return articleMsg{content: article.TextContent}
	}
}

func main() {
	ti := textinput.New()
	ti.Placeholder = "Enter your search query..."
	ti.CharLimit = 150
	ti.Width = 50

	vp := viewport.New(0, 0)
	vp.YPosition = 2

	p := tea.NewProgram(model{
		textInput:   ti,
		results:     []SearchResult{},
		state:       wikiSelectionView,
		wikiOptions: []string{"wikipedia", "arch"},
		viewport:    vp,
	})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

