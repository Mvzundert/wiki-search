package wiki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

// Custom messages to pass data between functions.
type SearchMsg struct {
	Results []SearchResult
	Err     error
}
type ArticleMsg struct {
	Content string
	Err     error
}

// PerformSearch is a command that makes the API call.
func PerformSearch(term string, wikiType string) tea.Cmd {
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
			return SearchMsg{Err: err}
		}
		req.Header.Set("User-Agent", "Your-CLI-Tool-Name/1.0 (Contact: your-email@example.com)")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return SearchMsg{Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return SearchMsg{Err: fmt.Errorf("API request failed with status code: %d %s", resp.StatusCode, resp.Status)}
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return SearchMsg{Err: err}
		}
		var data Response
		if err := json.Unmarshal(body, &data); err != nil {
			return SearchMsg{Err: fmt.Errorf("failed to parse API response: %w", err)}
		}
		return SearchMsg{Results: data.Query.Search}
	}
}

// FetchArticle fetches the full article content.
func FetchArticle(title string, wikiType string) tea.Cmd {
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
			return ArticleMsg{Err: err}
		}
		req.Header.Set("User-Agent", "Your-CLI-Tool-Name/1.0 (Contact: your-email@example.com)")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return ArticleMsg{Err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return ArticleMsg{Err: fmt.Errorf("API request failed with status code: %d %s", resp.StatusCode, resp.Status)}
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ArticleMsg{Err: err}
		}
		var data ArticleResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return ArticleMsg{Err: fmt.Errorf("failed to parse article response: %w", err)}
		}
		parsedURL, err := url.Parse(fullURL)
		if err != nil {
			return ArticleMsg{Err: fmt.Errorf("failed to parse URL: %w", err)}
		}
		contentReader := bytes.NewReader([]byte(data.Parse.Text.Content))
		article, err := readability.FromReader(contentReader, parsedURL)
		if err != nil {
			return ArticleMsg{Err: fmt.Errorf("failed to make content readable: %w", err)}
		}
		return ArticleMsg{Content: article.TextContent}
	}
}
