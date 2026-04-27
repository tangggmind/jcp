package models

const (
	ReviewTypeDaily   = "daily_review"
	ReviewTypeSummary = "summary_review"

	ReviewSourcePaste    = "paste"
	ReviewSourceDownload = "download"
	ReviewSourceLocal    = "local"
)

type ReviewArticle struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	Date            string   `json:"date"`
	Title           string   `json:"title"`
	FilePath        string   `json:"filePath"`
	TemplateID      string   `json:"templateId"`
	TemplateName    string   `json:"templateName"`
	Summary         string   `json:"summary"`
	Tags            []string `json:"tags"`
	Stocks          []string `json:"stocks"`
	ProfitLoss      float64  `json:"profitLoss"`
	Emotion         string   `json:"emotion"`
	DisciplineScore *int     `json:"disciplineScore,omitempty"`
	ImageCount      int      `json:"imageCount"`
	CreatedAt       int64    `json:"createdAt"`
	UpdatedAt       int64    `json:"updatedAt"`
}

type ReviewArticleDetail struct {
	Article ReviewArticle `json:"article"`
	Content string        `json:"content"`
	Warning string        `json:"warning,omitempty"`
	Error   string        `json:"error,omitempty"`
}

type ReviewListRequest struct {
	Query     string   `json:"query"`
	Type      string   `json:"type"`
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	Tags      []string `json:"tags"`
	Stocks    []string `json:"stocks"`
	Page      int      `json:"page"`
	PageSize  int      `json:"pageSize"`
}

type ReviewListResult struct {
	Items []ReviewArticle `json:"items"`
	Total int             `json:"total"`
	Error string          `json:"error,omitempty"`
}

type CreateDailyReviewRequest struct {
	Date           string   `json:"date"`
	TemplateID     string   `json:"templateId"`
	Title          string   `json:"title"`
	Stocks         []string `json:"stocks"`
	MarketSnapshot string   `json:"marketSnapshot"`
}

type SaveReviewArticleRequest struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type ReviewTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	IsBuiltin   bool   `json:"isBuiltin"`
	IsDefault   bool   `json:"isDefault"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type SaveReviewTemplateRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	IsDefault   bool   `json:"isDefault"`
}

type SaveReviewImageRequest struct {
	ArticleID  string `json:"articleId"`
	Date       string `json:"date"`
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	DataBase64 string `json:"dataBase64"`
}

type DownloadReviewImageRequest struct {
	ArticleID string `json:"articleId"`
	Date      string `json:"date"`
	URL       string `json:"url"`
}

type SaveReviewImageResult struct {
	AssetID      string `json:"assetId"`
	FilePath     string `json:"filePath"`
	MarkdownPath string `json:"markdownPath"`
	MarkdownText string `json:"markdownText"`
	MimeType     string `json:"mimeType"`
	Size         int64  `json:"size"`
	Error        string `json:"error,omitempty"`
}

type CompareReviewRequest struct {
	ArticleIDs []string `json:"articleIds"`
	StartDate  string   `json:"startDate"`
	EndDate    string   `json:"endDate"`
}

type CompareReviewResult struct {
	Items      []CompareReviewItem `json:"items"`
	TagStats   []CompareStatItem   `json:"tagStats"`
	StockStats []CompareStatItem   `json:"stockStats"`
	Error      string              `json:"error,omitempty"`
}

type CompareReviewItem struct {
	ArticleID       string            `json:"articleId"`
	Date            string            `json:"date"`
	Title           string            `json:"title"`
	Summary         string            `json:"summary"`
	Tags            []string          `json:"tags"`
	Stocks          []string          `json:"stocks"`
	ProfitLoss      float64           `json:"profitLoss"`
	Emotion         string            `json:"emotion"`
	DisciplineScore *int              `json:"disciplineScore,omitempty"`
	Sections        map[string]string `json:"sections"`
}

type CompareStatItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
