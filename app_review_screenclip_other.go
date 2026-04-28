//go:build !windows

package main

import "github.com/run-bigpig/jcp/internal/models"

func (a *App) CaptureReviewScreenClip() models.ReviewScreenCaptureResult {
	return models.ReviewScreenCaptureResult{Error: "当前系统暂不支持直接屏幕框选截图"}
}
