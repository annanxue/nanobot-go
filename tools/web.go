package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	UserAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7_2) AppleWebKit/537.36"
	MaxRedirects = 5
)

type WebSearchTool struct {
	apiKey     string
	maxResults int
}

func NewWebSearchTool(apiKey string, maxResults int) *WebSearchTool {
	if maxResults <= 0 {
		maxResults = 5
	}
	return &WebSearchTool{
		apiKey:     apiKey,
		maxResults: maxResults,
	}
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Search the web. Returns titles, URLs, and snippets."
}

func (t *WebSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query",
			},
			"count": map[string]interface{}{
				"type":        "number",
				"description": "Results (1-10)",
				"minimum":     1,
				"maximum":     10,
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if t.apiKey == "" {
		return "", fmt.Errorf("BRAVE_API_KEY not configured")
	}

	query, ok := params["query"].(string)
	if !ok {
		return "", fmt.Errorf("query is required")
	}

	count := t.maxResults
	if c, ok := params["count"].(int); ok {
		if c < 1 {
			count = 1
		} else if c > 10 {
			count = 10
		} else {
			count = c
		}
	}

	reqURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d", url.QueryEscape(query), count)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", t.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	web, ok := result["web"].(map[string]interface{})
	if !ok {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	results, ok := web["results"].([]interface{})
	if !ok || len(results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s\n", query))

	for i, item := range results {
		if i >= count {
			break
		}
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		title, _ := itemMap["title"].(string)
		itemURL, _ := itemMap["url"].(string)
		description, _ := itemMap["description"].(string)

		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, title, itemURL))
		if description != "" {
			lines = append(lines, fmt.Sprintf("   %s", description))
		}
	}

	return strings.Join(lines, "\n"), nil
}

type WebFetchTool struct {
	maxChars int
}

func NewWebFetchTool(maxChars int) *WebFetchTool {
	if maxChars <= 0 {
		maxChars = 50000
	}
	return &WebFetchTool{maxChars: maxChars}
}

func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

func (t *WebFetchTool) Description() string {
	return "Fetch URL and extract readable content (HTML → markdown/text)."
}

func (t *WebFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch",
			},
			"extractMode": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"markdown", "text"},
				"default":     "markdown",
				"description": "Output format",
			},
			"maxChars": map[string]interface{}{
				"type":        "number",
				"minimum":     100,
				"description": "Maximum characters to return",
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	urlStr, ok := params["url"].(string)
	if !ok {
		return "", fmt.Errorf("url is required")
	}

	isValid, errorMsg := validateURL(urlStr)
	if !isValid {
		return "", fmt.Errorf("URL validation failed: %s", errorMsg)
	}

	maxChars := t.maxChars
	if mc, ok := params["maxChars"].(int); ok {
		maxChars = mc
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error fetching URL: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	var text, extractor string

	if strings.Contains(contentType, "application/json") {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			formatted, _ := json.MarshalIndent(jsonData, "", "  ")
			text = string(formatted)
		} else {
			text = string(body)
		}
		extractor = "json"
	} else if strings.Contains(contentType, "text/html") || strings.HasPrefix(strings.ToLower(string(body[:min(256, len(body))])), "<!doctype") || strings.HasPrefix(strings.ToLower(string(body[:min(256, len(body))])), "<html") {
		text = t.toMarkdown(string(body))
		extractor = "readability"
	} else {
		text = string(body)
		extractor = "raw"
	}

	truncated := len(text) > maxChars
	if truncated {
		text = text[:maxChars]
	}

	result := map[string]interface{}{
		"url":       urlStr,
		"finalUrl":  resp.Request.URL.String(),
		"status":    resp.StatusCode,
		"extractor": extractor,
		"truncated": truncated,
		"length":    len(text),
		"text":      text,
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

func (t *WebFetchTool) toMarkdown(htmlStr string) string {
	text := stripTags(htmlStr)

	linkRe := regexp.MustCompile(`(?i)<a\s+[^>]*href=["']([^"']+)["'][^>]*>([\s\S]*?)</a>`)
	text = linkRe.ReplaceAllStringFunc(text, func(m string) string {
		matches := linkRe.FindStringSubmatch(m)
		if len(matches) > 2 {
			return fmt.Sprintf("[%s](%s)", stripTags(matches[2]), matches[1])
		}
		return m
	})

	headingRe := regexp.MustCompile(`(?i)<h([1-6])[^>]*>([\s\S]*?)</h[1-6]>`)
	text = headingRe.ReplaceAllStringFunc(text, func(m string) string {
		matches := headingRe.FindStringSubmatch(m)
		if len(matches) > 2 {
			level, _ := strconv.Atoi(matches[1])
			return fmt.Sprintf("\n%s %s\n", strings.Repeat("#", level), stripTags(matches[2]))
		}
		return m
	})

	listRe := regexp.MustCompile(`(?i)<li[^>]*>([\s\S]*?)</li>`)
	text = listRe.ReplaceAllStringFunc(text, func(m string) string {
		matches := listRe.FindStringSubmatch(m)
		if len(matches) > 1 {
			return fmt.Sprintf("\n- %s", stripTags(matches[1]))
		}
		return m
	})

	text = regexp.MustCompile(`(?i)</(p|div|section|article)>`).ReplaceAllString(text, "\n\n")
	text = regexp.MustCompile(`(?i)<(br|hr)\s*/?>`).ReplaceAllString(text, "\n")

	return normalizeWhitespace(text)
}

func stripTags(htmlStr string) string {
	htmlStr = regexp.MustCompile(`(?i)<script[\s\S]*?</script>`).ReplaceAllString(htmlStr, "")
	htmlStr = regexp.MustCompile(`(?i)<style[\s\S]*?</style>`).ReplaceAllString(htmlStr, "")
	htmlStr = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(htmlStr, "")
	return html.UnescapeString(strings.TrimSpace(htmlStr))
}

func normalizeWhitespace(text string) string {
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

func validateURL(urlStr string) (bool, string) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false, err.Error()
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false, fmt.Sprintf("Only http/https allowed, got '%s'", parsed.Scheme)
	}

	if parsed.Host == "" {
		return false, "Missing domain"
	}

	return true, ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
