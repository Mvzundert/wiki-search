# Go Wiki Search CLI

A lightweight, terminal-based tool for searching and reading articles from Wikipedia and ArchWiki. Built with Go and the Bubble Tea framework.

## Features

-   🌐 **Multi-Wiki Support:** Search for articles on both Wikipedia and ArchWiki.
-   🔎 **Full-text Search:** Find articles by keywords.
-   📖 **Article Viewer:** Read article content directly in the terminal.
-   ⌨️ **Vim-like Navigation:** Navigate articles and search results with familiar `j`, `k`, `n`, `p`, `ctrl+d`, and `ctrl+u` keybindings.
-   🔍 **In-Article Search:** Search for text within the current article.
-   🔗 **Hyperlink Highlighting:** Automatically highlights URLs in blue for easy identification.
-   🚀 **External Links:** Open a selected article in your default web browser with a single keypress.

---

## Installation

### Prerequisites

You must have Go installed on your system.

```bash
# Clone the repository
git clone [https://github.com/Mvzundert/wiki-search.git](https://github.com/Mvzundert/wiki-search.git)

# Navigate to the project directory
cd wiki-search

# Build and run the application
go build -o wiki-search

./wiki-search

### Usage

#### Wiki Selection

When you first launch the application, you'll be prompted to select a wiki to search. Use the **Up** and **Down** arrow keys (or `k` and `j`) to navigate and press **Enter** to select.

#### Searching

Once a wiki is selected, type your search query and press **Enter**. The application will display a list of matching articles.

#### Navigation

* **Up/Down (`j`/`k`):** Navigate through search results or scroll the article content line by line.
* **Enter:** Select a search result to view the article.
* **Ctrl+d/Ctrl+u:** Scroll the article content half a page at a time (like Vim).
* **Esc:** Go back to the previous screen (e.g., from an article to search results).
* **`o`:** Open the currently selected article in your web browser.
* **`q` or `Ctrl+c`:** Quit the application.

#### In-Article Search

* **`/`:** Start an in-article search. Type your query and press **Enter**.
* **`n`:** Jump to the next search result.
* **`p`:** Jump to the previous search result.

***

### Dependencies

This project relies on the following Go packages:

* `github.com/charmbracelet/bubbletea`
* `github.com/charmbracelet/bubbles/textinput`
* `github.com/charmbracelet/bubbles/viewport`
* `github.com/fatih/color`
* `github.com/go-shiori/go-readability`
* `regexp`

These dependencies will be automatically installed when you build the project using `go build`.
