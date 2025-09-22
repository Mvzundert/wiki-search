package utils

import (
	"github.com/fatih/color"
	"strings"
)

// FormatText applies basic formatting for readability (e.g., bold for headers).
func FormatText(text string) string {
	var formatted strings.Builder
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.ToUpper(line) == line && len(line) > 0 {
			formatted.WriteString(color.New(color.Bold).Sprint(line))
			formatted.WriteString("\n\n")
		} else {
			formatted.WriteString(line)
			formatted.WriteString("\n")
		}
	}
	return formatted.String()
}

// FindMatches returns the starting index of all matches
func FindMatches(content, query string) []int {
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

// HighlightText handles all text formatting, including search matches and URLs
func HighlightText(content, query string, searchMatches []int, currentMatch int, urlMatches [][]int) string {
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

// WrapText wraps the given string to the specified width.
func WrapText(text string, width int) string {
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

// CalculateLineFromIndex determines the line number based on a character index
func CalculateLineFromIndex(content string, index int) int {
	return strings.Count(content[:index], "\n")
}
