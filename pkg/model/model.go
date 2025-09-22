package model

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"

	"wiki-search/pkg/utils"
	"wiki-search/pkg/wiki"
)

// State represents the current view of the application.
type state int

const (
	wikiSelectionView state = iota
	searchResultsView
	articleView
	searchArticleView
)

// Model holds the state of our application.
type Model struct {
	state             state
	textInput         textinput.Model
	results           []wiki.SearchResult
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

// New initializes a new model.
func New(ti textinput.Model, vp viewport.Model, urlRegex *regexp.Regexp) Model {
	return Model{
		textInput:   ti,
		results:     []wiki.SearchResult{},
		state:       wikiSelectionView,
		wikiOptions: []string{"wikipedia", "arch"},
		viewport:    vp,
		urlRegex:    urlRegex,
	}
}

// Init initializes the application state.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all user input and model updates.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var vpCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4
		wrappedContent := utils.WrapText(m.articleContent, m.viewport.Width)
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
				m.viewport.SetYOffset(utils.CalculateLineFromIndex(m.articleContent, m.matchIndexes[m.currentMatchIndex]))
			}
		case "p":
			if m.state == articleView && len(m.matchIndexes) > 0 {
				m.currentMatchIndex = (m.currentMatchIndex - 1 + len(m.matchIndexes)) % len(m.matchIndexes)
				m.viewport.SetYOffset(utils.CalculateLineFromIndex(m.articleContent, m.matchIndexes[m.currentMatchIndex]))
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
				m.matchIndexes = utils.FindMatches(m.articleContent, m.searchQuery)
				m.currentMatchIndex = 0
				m.textInput.Blur()
				m.state = articleView
				if len(m.matchIndexes) > 0 {
					m.viewport.SetYOffset(utils.CalculateLineFromIndex(m.articleContent, m.matchIndexes[0]))
				}
				return m, nil
			} else if m.textInput.Focused() {
				if m.textInput.Value() != "" {
					m.statusMsg = "Searching..."
					m.textInput.Blur()
					return m, wiki.PerformSearch(m.textInput.Value(), m.searchType)
				}
			} else if m.state == searchResultsView && len(m.results) > 0 {
				m.selectedTitle = m.results[m.cursor].Title
				m.statusMsg = "Fetching article..."
				return m, wiki.FetchArticle(m.selectedTitle, m.searchType)
			}
		}

	case wiki.SearchMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
			m.textInput.Focus()
		} else {
			m.results = msg.Results
			m.statusMsg = fmt.Sprintf("Found %d results for '%s'. Press Enter to select one.", len(m.results), m.textInput.Value())
			m.cursor = 0
		}

	case wiki.ArticleMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			m.state = articleView
			m.articleContent = msg.Content
			m.urlMatches = m.urlRegex.FindAllStringIndex(m.articleContent, -1)
			m.statusMsg = fmt.Sprintf("Displaying article: %s", m.selectedTitle)

			wrappedContent := utils.WrapText(m.articleContent, m.viewport.Width)
			m.viewport.SetContent(wrappedContent)
		}
	}

	m.viewport, vpCmd = m.viewport.Update(msg)
	m.textInput, cmd = m.textInput.Update(msg)

	return m, tea.Batch(cmd, vpCmd)
}

// View renders the UI to the terminal.
func (m Model) View() string {
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
			formattedContent := utils.FormatText(m.articleContent)
			wrappedContent := utils.WrapText(formattedContent, m.viewport.Width)
			highlightedContent := utils.HighlightText(wrappedContent, m.searchQuery, m.matchIndexes, m.currentMatchIndex, m.urlMatches)
			m.viewport.SetContent(highlightedContent)
			s.WriteString(m.viewport.View())
			s.WriteString(mainColor("\n\nPress 'esc' to go back, Up/Down to scroll, '/' to search, 'n/p' to jump between matches, 'q' to quit."))
		}
	}
	return s.String()
}
