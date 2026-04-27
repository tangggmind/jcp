package services

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

type ReviewMarkdownMeta struct {
	Type            string   `yaml:"type"`
	Date            string   `yaml:"date"`
	Title           string   `yaml:"title"`
	TemplateID      string   `yaml:"templateId"`
	TemplateName    string   `yaml:"templateName"`
	Summary         string   `yaml:"summary"`
	Tags            []string `yaml:"tags"`
	Stocks          []string `yaml:"stocks"`
	ProfitLoss      float64  `yaml:"profitLoss"`
	Emotion         string   `yaml:"emotion"`
	DisciplineScore *int     `yaml:"disciplineScore"`
	CreatedAt       string   `yaml:"createdAt"`
	UpdatedAt       string   `yaml:"updatedAt"`
}

type ParsedReviewMarkdown struct {
	Meta           ReviewMarkdownMeta
	Content        string
	Summary        string
	ImageRefs      []string
	HasFrontMatter bool
	NeedsRewrite   bool
	Warning        string
}

func ParseReviewMarkdown(raw string) ParsedReviewMarkdown {
	metaRaw, content, hasFrontMatter, needsRewrite, warning := splitFrontMatter(raw)
	parsed := ParsedReviewMarkdown{
		Content:        content,
		HasFrontMatter: hasFrontMatter,
		NeedsRewrite:   needsRewrite,
		Warning:        warning,
	}

	if hasFrontMatter && metaRaw != "" {
		if err := yaml.Unmarshal([]byte(metaRaw), &parsed.Meta); err != nil {
			parsed.NeedsRewrite = true
			parsed.Warning = "Front Matter 解析失败，将在保存时重写"
		}
	}

	parsed.Meta.Tags = normalizeStringList(parsed.Meta.Tags)
	parsed.Meta.Stocks = normalizeStringList(parsed.Meta.Stocks)
	parsed.Summary = strings.TrimSpace(parsed.Meta.Summary)
	if parsed.Summary == "" {
		parsed.Summary = GenerateReviewSummary(parsed.Content, 160)
	}
	parsed.ImageRefs = ScanLocalReviewImageRefs(parsed.Content)
	return parsed
}

func splitFrontMatter(raw string) (meta string, content string, hasFrontMatter bool, needsRewrite bool, warning string) {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	if normalized == "---" {
		return "", "", true, true, "Front Matter 缺少结束分隔符"
	}
	if !strings.HasPrefix(normalized, "---\n") {
		return "", raw, false, false, ""
	}

	rest := strings.TrimPrefix(normalized, "---\n")
	lines := strings.Split(rest, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i+1:], "\n"), true, false, ""
		}
	}

	return rest, rest, true, true, "Front Matter 缺少结束分隔符"
}

func GenerateReviewSummary(content string, limit int) string {
	if limit <= 0 {
		limit = 160
	}

	cleaned := stripMarkdown(content)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if utf8.RuneCountInString(cleaned) <= limit {
		return cleaned
	}

	runes := []rune(cleaned)
	return strings.TrimSpace(string(runes[:limit]))
}

func ScanLocalReviewImageRefs(content string) []string {
	matches := markdownImagePattern.FindAllStringSubmatch(content, -1)
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ref := strings.TrimSpace(match[1])
		ref = strings.Trim(ref, `"'`)
		lower := strings.ToLower(ref)
		if strings.HasPrefix(lower, "http://") ||
			strings.HasPrefix(lower, "https://") ||
			strings.HasPrefix(lower, "data:") ||
			strings.HasPrefix(ref, "/") ||
			strings.Contains(ref, `\`) {
			continue
		}
		if strings.Contains(ref, "pictures/") || strings.HasPrefix(ref, "../pictures/") {
			refs = append(refs, ref)
		}
	}
	return refs
}

func normalizeStringList(values []string) []string {
	if values == nil {
		return []string{}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func stripMarkdown(content string) string {
	result := markdownImagePattern.ReplaceAllString(content, "")
	result = markdownLinkPattern.ReplaceAllString(result, "$1")
	result = markdownFencePattern.ReplaceAllString(result, " ")
	for _, pattern := range markdownCleanupPatterns {
		result = pattern.ReplaceAllString(result, " ")
	}
	return result
}

var (
	markdownImagePattern    = regexp.MustCompile(`!\[[^\]]*]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	markdownLinkPattern     = regexp.MustCompile(`\[([^\]]+)]\([^)]+\)`)
	markdownFencePattern    = regexp.MustCompile("(?s)```.*?```")
	markdownCleanupPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^#{1,6}\s*`),
		regexp.MustCompile(`[*_~` + "`" + `>|-]+`),
		regexp.MustCompile(`(?m)^\s*\d+\.\s+`),
		regexp.MustCompile(`\s+`),
	}
)
