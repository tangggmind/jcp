package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kbinani/screenshot"
	"github.com/run-bigpig/jcp/internal/adk"
	adkopenai "github.com/run-bigpig/jcp/internal/adk/openai"
	"github.com/run-bigpig/jcp/internal/models"
	"github.com/run-bigpig/jcp/internal/pkg/proxy"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

const reviewOCRPrompt = `你是一个严格的 OCR 转写引擎。请只识别并转写图片中实际可见的文字内容，按原图结构整理为可读 Markdown。

要求：
1. 只输出图片里存在的文字，不要解释 OCR 过程，不要输出接口、模型、stream、配置或请求说明。
2. 保留数字、股票代码、价格、百分比、时间、表格结构和标题层级。
3. 如果图片是交易软件、行情图或表格，优先用 Markdown 表格转写可见字段。
4. 不要臆造图片中不存在的内容；看不清的内容标记为 [无法识别]。
5. 如果图片中没有可识别文字，只输出：未识别到文字。`

const reviewOCRUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) CherryStudio/1.2.4 Chrome/126.0.6478.234 Electron/31.7.6 Safari/537.36"

func (a *App) CaptureReviewScreen() models.ReviewScreenCaptureResult {
	if a.ctx != nil {
		runtime.WindowMinimise(a.ctx)
		time.Sleep(350 * time.Millisecond)
		defer func() {
			runtime.WindowUnminimise(a.ctx)
			runtime.WindowShow(a.ctx)
		}()
	}

	dataURL, width, height, err := captureFullScreenPNGDataURL()
	if err != nil {
		return models.ReviewScreenCaptureResult{Error: err.Error()}
	}
	return models.ReviewScreenCaptureResult{
		DataBase64: dataURL,
		Width:      width,
		Height:     height,
	}
}

func captureFullScreenPNGDataURL() (string, int, int, error) {
	count := screenshot.NumActiveDisplays()
	if count == 0 {
		return "", 0, 0, fmt.Errorf("未检测到可截图的显示器")
	}

	var union image.Rectangle
	for i := 0; i < count; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		if i == 0 {
			union = bounds
			continue
		}
		union = union.Union(bounds)
	}
	if union.Empty() {
		return "", 0, 0, fmt.Errorf("屏幕区域为空")
	}

	canvas := image.NewRGBA(image.Rect(0, 0, union.Dx(), union.Dy()))
	for i := 0; i < count; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			return "", 0, 0, fmt.Errorf("截取屏幕失败: %w", err)
		}
		target := image.Rect(bounds.Min.X-union.Min.X, bounds.Min.Y-union.Min.Y, bounds.Max.X-union.Min.X, bounds.Max.Y-union.Min.Y)
		draw.Draw(canvas, target, img, image.Point{}, draw.Src)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return "", 0, 0, fmt.Errorf("编码截图失败: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), union.Dx(), union.Dy(), nil
}

func (a *App) OCRReviewImage(req models.ReviewOCRRequest) models.ReviewOCRResult {
	data, mimeType, dataURL, err := normalizeReviewOCRImage(req)
	if err != nil {
		return models.ReviewOCRResult{Error: err.Error()}
	}

	cfg := a.configService.GetConfig()
	aiConfig := a.getDefaultAIConfig(cfg)
	if aiConfig == nil {
		return models.ReviewOCRResult{Error: "未配置 AI 服务"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var text string
	switch aiConfig.Provider {
	case models.AIProviderOpenAI:
		text, err = callOpenAIVisionOCR(ctx, aiConfig, dataURL)
	case models.AIProviderAnthropic:
		text, err = callAnthropicVisionOCR(ctx, aiConfig, data, mimeType)
	case models.AIProviderGemini, models.AIProviderVertexAI:
		text, err = callADKVisionOCR(ctx, aiConfig, data, mimeType)
	default:
		err = fmt.Errorf("当前 AI Provider 不支持 OCR: %s", aiConfig.Provider)
	}
	if err != nil {
		return models.ReviewOCRResult{Error: err.Error()}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return models.ReviewOCRResult{Error: "AI 未返回可用识别结果"}
	}
	return models.ReviewOCRResult{Text: text}
}

func normalizeReviewOCRImage(req models.ReviewOCRRequest) ([]byte, string, string, error) {
	raw := strings.TrimSpace(req.DataBase64)
	if raw == "" {
		return nil, "", "", fmt.Errorf("图片数据不能为空")
	}

	mimeType := strings.TrimSpace(req.MimeType)
	payload := raw
	if strings.HasPrefix(raw, "data:") {
		header, body, ok := strings.Cut(raw, ",")
		if !ok {
			return nil, "", "", fmt.Errorf("图片 Data URL 格式错误")
		}
		payload = body
		if strings.Contains(header, ";base64") {
			mimeType = strings.TrimPrefix(strings.TrimSuffix(header, ";base64"), "data:")
		}
	}
	if mimeType == "" {
		mimeType = "image/png"
	}
	if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return nil, "", "", fmt.Errorf("仅支持图片 OCR")
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", "", fmt.Errorf("图片 base64 解码失败: %w", err)
	}
	if len(data) == 0 {
		return nil, "", "", fmt.Errorf("图片数据为空")
	}

	dataURL := raw
	if !strings.HasPrefix(raw, "data:") {
		dataURL = "data:" + mimeType + ";base64," + payload
	}
	return data, mimeType, dataURL, nil
}

func callADKVisionOCR(ctx context.Context, aiConfig *models.AIConfig, data []byte, mimeType string) (string, error) {
	factory := adk.NewModelFactory()
	llm, err := factory.CreateModel(ctx, aiConfig)
	if err != nil {
		return "", err
	}

	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: genai.RoleUser,
				Parts: []*genai.Part{
					{Text: reviewOCRPrompt},
					{InlineData: &genai.Blob{Data: data, MIMEType: mimeType}},
				},
			},
		},
		Config: &genai.GenerateContentConfig{MaxOutputTokens: 4096},
	}

	var result strings.Builder
	for resp, err := range llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", err
		}
		if resp == nil || resp.Content == nil {
			continue
		}
		for _, part := range resp.Content.Parts {
			if part.Thought || part.Text == "" {
				continue
			}
			result.WriteString(part.Text)
		}
	}
	return result.String(), nil
}

func callOpenAIVisionOCR(ctx context.Context, aiConfig *models.AIConfig, dataURL string) (string, error) {
	baseURL := normalizeReviewOpenAIBaseURL(aiConfig.BaseURL)
	if aiConfig.UseResponses {
		endpoint := strings.TrimSuffix(baseURL, "/") + "/responses"
		body := map[string]any{
			"model": aiConfig.ModelName,
			"input": []map[string]any{
				{
					"role": "user",
					"content": []map[string]any{
						{"type": "input_text", "text": reviewOCRPrompt},
						{"type": "input_image", "image_url": dataURL},
					},
				},
			},
			"max_output_tokens": 4096,
		}
		if aiConfig.ForceStream {
			body["stream"] = true
		}
		respBody, err := doReviewAIJSONRequest(ctx, endpoint, aiConfig.APIKey, body, aiConfig.ForceStream)
		if err != nil {
			return "", err
		}
		return extractReviewResponsesText(respBody), nil
	}

	endpoint := strings.TrimSuffix(baseURL, "/") + "/chat/completions"
	body := map[string]any{
		"model": aiConfig.ModelName,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": reviewOCRPrompt},
					{"type": "image_url", "image_url": map[string]any{"url": dataURL}},
				},
			},
		},
	}
	switch adkopenai.ResolveTokenParamMode(aiConfig.ModelName, string(aiConfig.TokenParamMode)) {
	case adkopenai.TokenParamModeMaxCompletionTokens:
		body["max_completion_tokens"] = 4096
	default:
		body["max_tokens"] = 4096
	}
	if aiConfig.ForceStream {
		body["stream"] = true
	}
	respBody, err := doReviewAIJSONRequest(ctx, endpoint, aiConfig.APIKey, body, aiConfig.ForceStream)
	if err != nil {
		return "", err
	}
	return extractReviewChatCompletionText(respBody), nil
}

func callAnthropicVisionOCR(ctx context.Context, aiConfig *models.AIConfig, data []byte, mimeType string) (string, error) {
	baseURL := normalizeReviewAnthropicBaseURL(aiConfig.BaseURL)
	endpoint, err := url.JoinPath(baseURL, "v1", "messages")
	if err != nil {
		return "", fmt.Errorf("无效 Anthropic BaseURL: %w", err)
	}

	body := map[string]any{
		"model":      aiConfig.ModelName,
		"max_tokens": 4096,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": reviewOCRPrompt},
					{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": mimeType,
							"data":       base64.StdEncoding.EncodeToString(data),
						},
					},
				},
			},
		},
	}
	if aiConfig.ForceStream {
		body["stream"] = true
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("请求构造失败: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("请求创建失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", aiConfig.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("User-Agent", reviewOCRUserAgent)
	if aiConfig.ForceStream {
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Cache-Control", "no-cache")
	}

	client := &http.Client{Transport: proxy.GetManager().GetTransport()}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("AI OCR 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("AI OCR 请求失败 HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if aiConfig.ForceStream {
		return extractReviewAnthropicStreamText(respBody), nil
	}
	return extractReviewAnthropicText(respBody), nil
}

func doReviewAIJSONRequest(ctx context.Context, endpoint, apiKey string, body map[string]any, stream bool) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("请求构造失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("请求创建失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", reviewOCRUserAgent)
	if stream {
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
	}

	client := &http.Client{Transport: proxy.GetManager().GetTransport()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AI OCR 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI OCR 请求失败 HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func extractReviewChatCompletionText(respBody []byte) string {
	if text := extractReviewChatCompletionStreamText(respBody); text != "" {
		return text
	}

	var resp struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil || len(resp.Choices) == 0 {
		return ""
	}
	return stringifyReviewAIContent(resp.Choices[0].Message.Content)
}

func extractReviewResponsesText(respBody []byte) string {
	if text := extractReviewResponsesStreamText(respBody); text != "" {
		return text
	}

	var top struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &top); err != nil {
		return ""
	}
	if strings.TrimSpace(top.OutputText) != "" {
		return top.OutputText
	}
	var out strings.Builder
	for _, item := range top.Output {
		for _, content := range item.Content {
			if content.Text != "" {
				out.WriteString(content.Text)
			}
		}
	}
	return out.String()
}

func extractReviewAnthropicText(respBody []byte) string {
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ""
	}
	var out strings.Builder
	for _, content := range resp.Content {
		if content.Text != "" {
			out.WriteString(content.Text)
		}
	}
	return out.String()
}

func extractReviewChatCompletionStreamText(respBody []byte) string {
	var out strings.Builder
	for _, data := range reviewSSEDataLines(respBody) {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content any `json:"content"`
				} `json:"delta"`
				Message struct {
					Content any `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil || len(chunk.Choices) == 0 {
			continue
		}
		out.WriteString(stringifyReviewAIContent(chunk.Choices[0].Delta.Content))
		out.WriteString(stringifyReviewAIContent(chunk.Choices[0].Message.Content))
	}
	return out.String()
}

func extractReviewResponsesStreamText(respBody []byte) string {
	var (
		out          strings.Builder
		currentEvent string
	)
	for _, line := range strings.Split(string(respBody), "\n") {
		line = strings.TrimSpace(line)
		if event, ok := cutReviewSSEField(line, "event"); ok {
			currentEvent = event
			continue
		}
		data, ok := cutReviewSSEField(line, "data")
		if !ok || data == "" || data == "[DONE]" {
			continue
		}

		switch currentEvent {
		case "response.output_text.delta":
			var delta struct {
				Delta string `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &delta); err == nil {
				out.WriteString(delta.Delta)
			}
		case "response.completed":
			if out.Len() > 0 {
				currentEvent = ""
				continue
			}
			var completed struct {
				Response struct {
					OutputText string `json:"output_text"`
					Output     []struct {
						Content []struct {
							Text string `json:"text"`
						} `json:"content"`
					} `json:"output"`
				} `json:"response"`
			}
			if err := json.Unmarshal([]byte(data), &completed); err == nil {
				if completed.Response.OutputText != "" {
					out.WriteString(completed.Response.OutputText)
				}
				for _, item := range completed.Response.Output {
					for _, content := range item.Content {
						out.WriteString(content.Text)
					}
				}
			}
		default:
			var generic struct {
				OutputText string `json:"output_text"`
				Delta      string `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &generic); err == nil {
				out.WriteString(generic.OutputText)
				out.WriteString(generic.Delta)
			}
		}
		currentEvent = ""
	}
	return out.String()
}

func extractReviewAnthropicStreamText(respBody []byte) string {
	var out strings.Builder
	for _, data := range reviewSSEDataLines(respBody) {
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			ContentBlock struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content_block"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Text != "" {
			out.WriteString(event.Delta.Text)
		}
		if event.Type == "content_block_start" && event.ContentBlock.Text != "" {
			out.WriteString(event.ContentBlock.Text)
		}
	}
	return out.String()
}

func reviewSSEDataLines(respBody []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(respBody))
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	var dataLines []string
	for scanner.Scan() {
		data, ok := cutReviewSSEField(scanner.Text(), "data")
		if !ok {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}
		dataLines = append(dataLines, data)
	}
	return dataLines
}

func cutReviewSSEField(line string, field string) (string, bool) {
	line = strings.TrimSpace(line)
	prefix := field + ":"
	value, ok := strings.CutPrefix(line, prefix)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func stringifyReviewAIContent(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		var out strings.Builder
		for _, item := range typed {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					out.WriteString(text)
				}
			}
		}
		return out.String()
	default:
		return ""
	}
}

func normalizeReviewOpenAIBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if baseURL == "" {
		return "https://api.openai.com/v1"
	}
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	return baseURL
}

func normalizeReviewAnthropicBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if baseURL == "" {
		return "https://api.anthropic.com"
	}
	return strings.TrimSuffix(baseURL, "/v1")
}
