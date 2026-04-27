package main

import (
	"fmt"
	"strings"

	"github.com/run-bigpig/jcp/internal/models"
)

func (a *App) GetReviewArticles(req models.ReviewListRequest) models.ReviewListResult {
	if a.reviewService == nil {
		return models.ReviewListResult{Items: []models.ReviewArticle{}, Total: 0, Error: "复盘服务未初始化"}
	}
	result, err := a.reviewService.ListArticles(req)
	if err != nil {
		return models.ReviewListResult{Items: []models.ReviewArticle{}, Total: 0, Error: err.Error()}
	}
	return result
}

func (a *App) GetReviewArticle(id string) models.ReviewArticleDetail {
	if a.reviewService == nil {
		return models.ReviewArticleDetail{Error: "复盘服务未初始化"}
	}
	detail, err := a.reviewService.GetArticle(id)
	if err != nil {
		return models.ReviewArticleDetail{Error: err.Error()}
	}
	return detail
}

func (a *App) CreateDailyReview(req models.CreateDailyReviewRequest) models.ReviewArticleDetail {
	if a.reviewService == nil {
		return models.ReviewArticleDetail{Error: "复盘服务未初始化"}
	}
	if strings.TrimSpace(req.MarketSnapshot) == "" && a.marketService != nil {
		if indices, err := a.marketService.GetMarketIndices(); err == nil {
			req.MarketSnapshot = formatReviewMarketSnapshot(indices)
		}
	}
	detail, err := a.reviewService.CreateDailyReview(req)
	if err != nil {
		return models.ReviewArticleDetail{Error: err.Error()}
	}
	return detail
}

func formatReviewMarketSnapshot(indices []models.MarketIndex) string {
	byCode := make(map[string]models.MarketIndex, len(indices))
	for _, index := range indices {
		byCode[strings.ToLower(strings.TrimSpace(index.Code))] = index
	}

	order := []struct {
		code string
		name string
	}{
		{code: "sh000001", name: "上证指数"},
		{code: "sz399001", name: "深证成指"},
		{code: "sz399006", name: "创业板指"},
	}

	parts := make([]string, 0, len(order))
	for _, item := range order {
		index, ok := byCode[item.code]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %.2f（%+.2f%%）", item.name, index.Price, index.ChangePercent))
	}
	return strings.Join(parts, "，")
}

func (a *App) SaveReviewArticle(req models.SaveReviewArticleRequest) models.ReviewArticleDetail {
	if a.reviewService == nil {
		return models.ReviewArticleDetail{Error: "复盘服务未初始化"}
	}
	detail, err := a.reviewService.SaveArticle(req)
	if err != nil {
		return models.ReviewArticleDetail{Error: err.Error()}
	}
	return detail
}

func (a *App) DeleteReviewArticle(id string) string {
	if a.reviewService == nil {
		return "复盘服务未初始化"
	}
	if err := a.reviewService.DeleteArticle(id); err != nil {
		return err.Error()
	}
	return "success"
}

func (a *App) GetReviewSummaryArticle() models.ReviewArticleDetail {
	if a.reviewService == nil {
		return models.ReviewArticleDetail{Error: "复盘服务未初始化"}
	}
	detail, err := a.reviewService.GetSummaryArticle()
	if err != nil {
		return models.ReviewArticleDetail{Error: err.Error()}
	}
	return detail
}

func (a *App) RebuildReviewIndex() string {
	if a.reviewService == nil {
		return "复盘服务未初始化"
	}
	if err := a.reviewService.RebuildIndex(); err != nil {
		return err.Error()
	}
	return "success"
}

func (a *App) GetReviewTemplates() []models.ReviewTemplate {
	if a.reviewService == nil {
		return []models.ReviewTemplate{}
	}
	templates, err := a.reviewService.ListTemplates()
	if err != nil {
		log.Warn("GetReviewTemplates failed: %v", err)
		return []models.ReviewTemplate{}
	}
	return templates
}

func (a *App) SaveReviewTemplate(req models.SaveReviewTemplateRequest) models.ReviewTemplate {
	if a.reviewService == nil {
		return models.ReviewTemplate{ID: "", Name: "复盘服务未初始化"}
	}
	template, err := a.reviewService.SaveTemplate(req)
	if err != nil {
		log.Warn("SaveReviewTemplate failed: %v", err)
		return models.ReviewTemplate{ID: "", Name: err.Error()}
	}
	return template
}

func (a *App) DeleteReviewTemplate(id string) string {
	if a.reviewService == nil {
		return "复盘服务未初始化"
	}
	if err := a.reviewService.DeleteTemplate(id); err != nil {
		return err.Error()
	}
	return "success"
}

func (a *App) SaveReviewPastedImage(req models.SaveReviewImageRequest) models.SaveReviewImageResult {
	if a.reviewService == nil {
		return models.SaveReviewImageResult{Error: "复盘服务未初始化"}
	}
	result, err := a.reviewService.SavePastedImage(req)
	if err != nil {
		return models.SaveReviewImageResult{Error: err.Error()}
	}
	return result
}

func (a *App) DownloadReviewImage(req models.DownloadReviewImageRequest) models.SaveReviewImageResult {
	if a.reviewService == nil {
		return models.SaveReviewImageResult{Error: "复盘服务未初始化"}
	}
	result, err := a.reviewService.DownloadImage(req)
	if err != nil {
		return models.SaveReviewImageResult{Error: err.Error()}
	}
	return result
}

func (a *App) GetReviewAssetBase64(filePath string) string {
	if a.reviewService == nil {
		return "复盘服务未初始化"
	}
	dataURL, err := a.reviewService.GetReviewAssetBase64(filePath)
	if err != nil {
		return err.Error()
	}
	return dataURL
}

func (a *App) CompareReviewArticles(req models.CompareReviewRequest) models.CompareReviewResult {
	if a.reviewService == nil {
		return models.CompareReviewResult{Error: "复盘服务未初始化"}
	}
	result, err := a.reviewService.Compare(req)
	if err != nil {
		return models.CompareReviewResult{Error: err.Error()}
	}
	return result
}
