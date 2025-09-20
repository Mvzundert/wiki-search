# Go Wiki Search CLI

A lightweight, terminal-based tool for searching and reading articles from Wikipedia and ArchWiki. Built with Go and the Bubble Tea framework.

---

## Features

* **Multi-Wiki Support:** Search for articles on both Wikipedia and ArchWiki.
* **Full-text Search:** Find articles by keywords.
* **Article Viewer:** Read article content directly in the terminal.
* **Vim-like Navigation:** Navigate articles and search results with familiar `j`, `k`, `n`, `p`, `ctrl+d`, and `ctrl+u` keybindings.
* **In-Article Search:** Search for text within the current article.
* **Hyperlink Highlighting:** Automatically highlights URLs in blue for easy identification.
* **External Links:** Open a selected article in your default web browser with a single keypress.

---

## Installation

The easiest way to get started is by downloading the pre-compiled binary for your operating system from the **Releases** page.

### Using the Binary

1.  Go to the [Releases](https://github.com/Mvzundert/wiki-search/releases) page.
2.  Download the binary for your operating system (e.g., `wiki-search-linux-amd64` for Linux, `wiki-search-windows-amd64.exe` for Windows).
3.  Place the file in a convenient directory.
4.  Open your terminal and run the executable.

```bash
# For Linux/macOS
# 1. Download the binary and place it in a bin directory (e.g., ~/bin)
mkdir -p ~/bin
mv wiki-search-linux-amd64 ~/bin/wiki-search

# 2. Add the directory to your PATH in your shell's profile file (~/.zshrc, ~/.bashrc, etc.)
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc

# 3. Reload your shell or open a new terminal
source ~/.zshrc

# You can now run the binary from anywhere
wiki-search
```

For Windows
```Bash
1. Place the downloaded .exe file in a directory you want to use.
2. Open the Start Menu, search for "Edit the system environment variables" and open it.
3. Click "Environment Variables..."
4. Under "User variables for <your-username>", select "Path" and click "Edit...".
5. Click "New" and add the path to the directory where you placed the binary.
6. Click "OK" on all windows to save the changes.
7. Open a new Command Prompt or PowerShell window to use the binary.
```

# Building from Source
If you prefer to build the application from source, you must have Go installed on your system.

Clone the repository and navigate to the project directory.

```Bash
git clone https://github.com/Mvzundert/wiki-search.git
```

```Bash
cd wiki-search
```

## Build and run the application.

```Bash
go build -o wiki-search
./wiki-search
```

# Usage

Wiki Selection
When you first launch the application, you'll be prompted to select a wiki to search. Use the Up and Down arrow keys (or k and j) to navigate and press Enter to select.

## Searching
Once a wiki is selected, type your search query and press Enter. The application will display a list of matching articles.

## Navigation
- Up/Down (j/k): Navigate through search results or scroll the article content line by line.
- Enter: Select a search result to view the article.
- Ctrl+d/Ctrl+u: Scroll the article content half a page at a time (like Vim).
- Esc: Go back to the previous screen (e.g., from an article to search results).
- o: Open the currently selected article in your web browser.
- q or Ctrl+c: Quit the application.

## In-Article Search
- /: Start an in-article search. Type your query and press Enter.
- n: Jump to the next search result.
- p: Jump to the previous search result.

## Dependencies
This project relies on the following Go packages:

github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles/textinput
github.com/charmbracelet/bubbles/viewport
github.com/fatih/color
github.com/go-shiori/go-readability
regexp

These dependencies will be automatically installed when you build the project using go build.
