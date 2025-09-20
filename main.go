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
	"regexp"
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
	searchArticleView
)

// Model holds the state of our application.
type model struct {
	state             state
	textInput         textinput.Model
	results           []SearchResult
	cursor            int
	statusMsg         string
	selectedTitle     string
	articleContent    string
	searchType        string
	wikiOptions       []string
	wikiCursor        int
	viewport          viewport.Model
	searchQuery       string
	matchIndexes      []int
	currentMatchIndex int
	urlRegex          *regexp.Regexp
	urlMatches        [][]int
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
		m.viewport.Height = msg.Height - 4
		wrappedContent := wrapText(m.articleContent, m.viewport.Width)
		m.viewport.SetContent(wrappedContent)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			switch m.state {
			case articleView, searchArticleView:
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

		case "/":
			if m.state == articleView {
				m.state = searchArticleView
				m.textInput.Focus()
				m.textInput.Prompt = "/"
				m.textInput.CharLimit = 100
				return m, nil
			}

		case "n":
			if m.state == articleView && len(m.matchIndexes) > 0 {
				m.currentMatchIndex = (m.currentMatchIndex + 1) % len(m.matchIndexes)
				m.viewport.SetYOffset(calculateLineFromIndex(m.articleContent, m.matchIndexes[m.currentMatchIndex]))
			}
		case "p":
			if m.state == articleView && len(m.matchIndexes) > 0 {
				m.currentMatchIndex = (m.currentMatchIndex - 1 + len(m.matchIndexes)) % len(m.matchIndexes)
				m.viewport.SetYOffset(calculateLineFromIndex(m.articleContent, m.matchIndexes[m.currentMatchIndex]))
			}
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
			}

		case "ctrl+u", "ctrl+d":
			if m.state == articleView {
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
				case "darwin":
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
			} else if m.state == searchArticleView {
				m.searchQuery = m.textInput.Value()
				m.matchIndexes = findMatches(m.articleContent, m.searchQuery)
				m.currentMatchIndex = 0
				m.textInput.Blur()
				m.state = articleView
				if len(m.matchIndexes) > 0 {
					m.viewport.SetYOffset(calculateLineFromIndex(m.articleContent, m.matchIndexes[0]))
				}
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
			m.urlMatches = m.urlRegex.FindAllStringIndex(m.articleContent, -1)
			m.statusMsg = fmt.Sprintf("Displaying article: %s", m.selectedTitle)

			// Apply text wrapping before setting viewport content
			wrappedContent := wrapText(m.articleContent, m.viewport.Width)
			m.viewport.SetContent(wrappedContent)
		}
	}

	m.viewport, vpCmd = m.viewport.Update(msg)
	m.textInput, cmd = m.textInput.Update(msg)

	return m, tea.Batch(cmd, vpCmd)
}

// View renders the UI to the terminal.
func (m model) View() string {
	s := strings.Builder{}
	mainColor := color.New(color.FgWhite).SprintFunc()

	switch m.state {
	case wikiSelectionView:
		s.WriteString(mainColor("Select a Wiki to Search:\n\n"))
		for i, wiki := range m.wikiOptions {
			cursor := " "
			if i == m.wikiCursor {
				cursor = color.New(color.Bold, color.FgGreen).Sprint(">")
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, mainColor(wiki)))
		}
		s.WriteString(mainColor("\n\nPress Enter to select, 'q' to quit."))

	case searchResultsView:
		s.WriteString(m.textInput.View())
		s.WriteString("\n\n")
		s.WriteString(mainColor(m.statusMsg))
		s.WriteString("\n\n")
		if len(m.results) > 0 {
			s.WriteString(mainColor("Search Results:\n"))
			for i, result := range m.results {
				var cursor string
				if i == m.cursor {
					cursor = color.New(color.Bold, color.FgGreen).Sprint("> ")
				} else {
					cursor = "  "
				}
				s.WriteString(fmt.Sprintf("%s%s\n", cursor, mainColor(result.Title)))
			}
		}
		s.WriteString(mainColor("\n\nEnter to search/select, Up/Down to navigate, 'o' to open in browser, 'q' to quit."))

	case articleView, searchArticleView:
		s.WriteString(color.New(color.Bold, color.FgCyan).Sprint(m.selectedTitle))
		s.WriteString("\n\n")
		if m.state == searchArticleView {
			s.WriteString(m.textInput.View())
			s.WriteString("\n\n")
			s.WriteString(mainColor("Press Enter to search, Esc to cancel."))
		} else {
			// Wrap the content before highlighting
			wrappedContent := wrapText(m.articleContent, m.viewport.Width)
			highlightedContent := highlightText(wrappedContent, m.searchQuery, m.matchIndexes, m.currentMatchIndex, m.urlMatches)
			m.viewport.SetContent(highlightedContent)
			s.WriteString(m.viewport.View())
			s.WriteString(mainColor("\n\nPress 'esc' to go back, Up/Down to scroll, '/' to search, 'n/p' to jump between matches, 'q' to quit."))
		}
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

// findMatches returns the starting index of all matches
func findMatches(content, query string) []int {
	if query == "" {
		return nil
	}
	var matches []int
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)
	start := 0
	for {
		i := strings.Index(lowerContent[start:], lowerQuery)
		if i == -1 {
			break
		}
		matches = append(matches, start+i)
		start += i + 1
	}
	return matches
}

// highlightText handles all text formatting, including search matches and URLs
func highlightText(content, query string, searchMatches []int, currentMatch int, urlMatches [][]int) string {
	var sb strings.Builder
	lastIndex := 0
	searchMatchColor := color.New(color.BgYellow, color.FgBlack).SprintFunc()
	currentMatchColor := color.New(color.BgHiYellow, color.FgBlack).SprintFunc()
	urlColor := color.New(color.FgHiBlue).SprintFunc()
	defaultColor := color.New(color.FgWhite).SprintFunc()

	type match struct {
		start           int
		end             int
		isURL           bool
		isCurrentSearch bool
	}
	var allMatches []match
	for i, start := range searchMatches {
		end := start + len(query)
		allMatches = append(allMatches, match{start, end, false, i == currentMatch})
	}
	for _, urlMatch := range urlMatches {
		allMatches = append(allMatches, match{urlMatch[0], urlMatch[1], true, false})
	}

	for i := range allMatches {
		for j := i + 1; j < len(allMatches); j++ {
			if allMatches[i].start > allMatches[j].start {
				allMatches[i], allMatches[j] = allMatches[j], allMatches[i]
			}
		}
	}

	for _, m := range allMatches {
		if m.start > lastIndex {
			sb.WriteString(defaultColor(content[lastIndex:m.start]))
		}
		matchStr := content[m.start:m.end]
		if m.isURL {
			sb.WriteString(urlColor(matchStr))
		} else if m.isCurrentSearch {
			sb.WriteString(currentMatchColor(matchStr))
		} else {
			sb.WriteString(searchMatchColor(matchStr))
		}
		lastIndex = m.end
	}

	if lastIndex < len(content) {
		sb.WriteString(defaultColor(content[lastIndex:]))
	}
	return sb.String()
}

// wrapText wraps the given string to the specified width.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		words := strings.Fields(line)
		if len(words) == 0 {
			result.WriteString("\n")
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) > width {
				result.WriteString(currentLine + "\n")
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
		result.WriteString(currentLine + "\n")
	}
	return result.String()
}

// calculateLineFromIndex determines the line number based on a character index
func calculateLineFromIndex(content string, index int) int {
	return strings.Count(content[:index], "\n")
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
	urlRegex := regexp.MustCompile(`https?://[^\s/$.?#].[^\s]*`)
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
		urlRegex:    urlRegex,
	})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

