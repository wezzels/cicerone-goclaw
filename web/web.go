// Package web provides web search and fetch capabilities.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearchProvider defines the interface for web search.
type SearchProvider interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
	Fetch(ctx context.Context, url string) (string, error)
}

// SearchResult represents a search result.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// DuckDuckGoProvider implements search using DuckDuckGo HTML.
type DuckDuckGoProvider struct {
	client *http.Client
}

// NewDuckDuckGoProvider creates a new DuckDuckGo search provider.
func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return &DuckDuckGoProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search performs a web search using DuckDuckGo.
func (p *DuckDuckGoProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	// Use DuckDuckGo instant answer API (no API key required)
	searchURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Cicerone/2.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Parse DuckDuckGo response
	var ddgResp struct {
		AbstractText string `json:"AbstractText"`
		AbstractURL  string `json:"AbstractURL"`
		Heading      string `json:"Heading"`
		RelatedTopics []struct {
			Text string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var results []SearchResult

	// Add abstract if available
	if ddgResp.AbstractText != "" {
		results = append(results, SearchResult{
			Title:   ddgResp.Heading,
			URL:     ddgResp.AbstractURL,
			Snippet: ddgResp.AbstractText,
		})
	}

	// Add related topics
	for _, topic := range ddgResp.RelatedTopics {
		if topic.Text != "" && topic.FirstURL != "" {
			// Extract title from text (first part before " - ")
			title := topic.Text
			if idx := strings.Index(topic.Text, " - "); idx > 0 {
				title = topic.Text[:idx]
			}
			results = append(results, SearchResult{
				Title:   title,
				URL:     topic.FirstURL,
				Snippet: topic.Text,
			})
		}
	}

	// If no results from DuckDuckGo, create a synthetic result with the query
	if len(results) == 0 {
		// Return a helpful message suggesting the user try /web or /fetch
		return nil, fmt.Errorf("no instant answers found for \"%s\". Try /web for web search with LLM context, or try a more specific query", query)
	}

	// Limit to top 5 results
	if len(results) > 5 {
		results = results[:5]
	}

	return results, nil
}

// Fetch retrieves content from a URL.
func (p *DuckDuckGoProvider) Fetch(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Cicerone/2.0)")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	// Extract text content from HTML (simple extraction)
	content := string(body)
	
	// Remove script and style tags
	content = stripTags(content, "script")
	content = stripTags(content, "style")
	content = stripTags(content, "nav")
	content = stripTags(content, "header")
	content = stripTags(content, "footer")
	
	// Remove HTML tags
	content = stripHTML(content)
	
	// Clean up whitespace
	content = strings.TrimSpace(content)
	
	// Limit to first 4000 characters
	if len(content) > 4000 {
		content = content[:4000] + "..."
	}

	return content, nil
}

// stripTags removes content between <tag> and </tag>
func stripTags(content, tag string) string {
	startTag := "<" + tag
	endTag := "</" + tag + ">"
	
	for {
		startIdx := strings.Index(content, startTag)
		if startIdx == -1 {
			break
		}
		
		// Find end of opening tag
		tagEnd := strings.Index(content[startIdx:], ">")
		if tagEnd == -1 {
			break
		}
		tagEnd += startIdx
		
		// Find closing tag
		closeIdx := strings.Index(content[tagEnd:], endTag)
		if closeIdx == -1 {
			break
		}
		closeIdx += tagEnd + len(endTag)
		
		content = content[:startIdx] + content[closeIdx:]
	}
	
	return content
}

// stripHTML removes HTML tags
func stripHTML(content string) string {
	var result strings.Builder
	inTag := false
	
	for _, ch := range content {
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(ch)
		}
	}
	
	return result.String()
}

// FormatSearchResults formats search results for display.
func FormatSearchResults(results []SearchResult) string {
	var sb strings.Builder
	
	sb.WriteString("Search Results:\n")
	sb.WriteString(strings.Repeat("-", 50) + "\n\n")
	
	for i, result := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		if result.URL != "" {
			sb.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		}
		if result.Snippet != "" {
			// Limit snippet length
			snippet := result.Snippet
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("   %s\n", snippet))
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}