package services

import (
	"fmt"
	"sort"
	"strings"

	"github.com/run-bigpig/jcp/internal/models"
)

func (s *ReviewService) Compare(req models.CompareReviewRequest) (models.CompareReviewResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var articles []models.ReviewArticle
	if len(req.ArticleIDs) > 0 {
		for _, id := range req.ArticleIDs {
			article, err := s.getArticleByIDNoLock(id)
			if err != nil {
				continue
			}
			if article.Type == models.ReviewTypeDaily {
				articles = append(articles, article)
			}
		}
	} else {
		result, err := s.listDailyArticlesForCompareNoLock(req.StartDate, req.EndDate)
		if err != nil {
			return models.CompareReviewResult{}, err
		}
		articles = result
	}

	sort.Slice(articles, func(i, j int) bool { return articles[i].Date < articles[j].Date })
	if len(articles) < 2 {
		return models.CompareReviewResult{}, fmt.Errorf("至少选择 2 篇每日复盘")
	}

	items := make([]models.CompareReviewItem, 0, len(articles))
	tagCounts := map[string]int{}
	stockCounts := map[string]int{}
	for _, article := range articles {
		content, _ := s.readArticleContentNoLock(article)
		for _, tag := range article.Tags {
			tagCounts[tag]++
		}
		for _, stock := range article.Stocks {
			stockCounts[stock]++
		}
		items = append(items, models.CompareReviewItem{
			ArticleID:       article.ID,
			Date:            article.Date,
			Title:           article.Title,
			Summary:         article.Summary,
			Tags:            article.Tags,
			Stocks:          article.Stocks,
			ProfitLoss:      article.ProfitLoss,
			Emotion:         article.Emotion,
			DisciplineScore: article.DisciplineScore,
			Sections:        ExtractReviewSections(content),
		})
	}
	return models.CompareReviewResult{Items: items, TagStats: statItems(tagCounts), StockStats: statItems(stockCounts)}, nil
}

func (s *ReviewService) listDailyArticlesForCompareNoLock(startDate, endDate string) ([]models.ReviewArticle, error) {
	rows, err := s.db.Query(`
		SELECT id, type, date, title, file_path, template_id, template_name, summary,
			tags_json, stocks_json, profit_loss, emotion, discipline_score,
			image_count, created_at, updated_at
		FROM review_articles
		WHERE type = ?
	`, models.ReviewTypeDaily)
	if err != nil {
		return nil, fmt.Errorf("查询复盘对比文章失败: %w", err)
	}
	defer rows.Close()

	var articles []models.ReviewArticle
	for rows.Next() {
		article, err := scanReviewArticle(rows)
		if err != nil {
			return nil, err
		}
		if startDate != "" && article.Date < startDate {
			continue
		}
		if endDate != "" && article.Date > endDate {
			continue
		}
		articles = append(articles, article)
	}
	return articles, rows.Err()
}

func ExtractReviewSections(markdown string) map[string]string {
	standard := map[string]struct{}{
		"今日市场": {}, "今日操作": {}, "持仓复盘": {}, "错误与偏差": {},
		"做得好的地方": {}, "情绪记录": {}, "明日计划": {}, "经验教训": {},
	}
	body := ParseReviewMarkdown(markdown).Content
	lines := strings.Split(body, "\n")
	sections := map[string]string{}
	current := ""
	var buffer []string
	flush := func() {
		if current == "" {
			return
		}
		text := strings.TrimSpace(strings.Join(buffer, "\n"))
		if len([]rune(text)) > 300 {
			text = string([]rune(text)[:300])
		}
		sections[current] = text
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			if _, ok := standard[title]; ok {
				current = title
				buffer = []string{}
			} else {
				current = ""
				buffer = nil
			}
			continue
		}
		if current != "" {
			buffer = append(buffer, line)
		}
	}
	flush()
	return sections
}

func statItems(counts map[string]int) []models.CompareStatItem {
	items := make([]models.CompareStatItem, 0, len(counts))
	for name, count := range counts {
		if name != "" {
			items = append(items, models.CompareStatItem{Name: name, Count: count})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Name < items[j].Name
		}
		return items[i].Count > items[j].Count
	})
	return items
}
