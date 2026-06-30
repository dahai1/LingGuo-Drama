package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BailianClient 阿里百炼 API 客户端 (文本 OpenAI 兼容, 图片 DashScope 原生)
type BailianClient struct {
	Config Config
	client *http.Client
}

// ================== 百炼图片原生结构体 ==================

type bailianImageReq struct {
	Model      string             `json:"model"`
	Input      bailianImageInput  `json:"input"`
	Parameters bailianImageParams `json:"parameters,omitempty"`
}

type bailianImageInput struct {
	Prompt       string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
}

type bailianImageParams struct {
	N    int    `json:"n,omitempty"`
	Size string `json:"size,omitempty"`
}

type bailianImageResp struct {
	Output struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
		Results    []struct {
			URL string `json:"url"`
		} `json:"results"`
	} `json:"output"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// GenerateScript 实现文本生成 (OpenAI 兼容格式)
func (c *BailianClient) GenerateScript(req ScriptRequest) (string, error) {
	payload := openAIChatReq{
		Model:     c.Config.BailianModel,
		Messages:  req.Messages,
		MaxTokens: req.MaxTokens,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.Config.BailianKey,
	}

	baseURL := c.Config.BailianBaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	url := strings.TrimRight(baseURL, "/") + "/chat/completions"

	resp, err := doRequest[*openAIChatResp](c.client, "POST", url, headers, payload)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("bailian returned no choices")
}

// GenerateImage 实现图片生成 (DashScope 原生 API)
func (c *BailianClient) GenerateImage(req ImageRequest) ([]string, error) {
	model := c.Config.BailianImageModel
	if model == "" {
		model = "wanx-v1"
	}

	n := req.N
	if n <= 0 {
		n = 1
	}

	size := req.Size
	if size == "" {
		size = "1024*1024"
	}
	// 百炼使用 "1024*1024" 格式而非 "1024x1024"
	size = strings.ReplaceAll(size, "x", "*")

	payload := bailianImageReq{
		Model: model,
		Input: bailianImageInput{
			Prompt: req.Prompt,
		},
		Parameters: bailianImageParams{
			N:    n,
			Size: size,
		},
	}

	jsonBytes, _ := json.Marshal(payload)

	// 百炼图片使用原生 API 端点 (非 OpenAI 兼容)
	// Base URL: https://dashscope.aliyuncs.com
	// 图片端点: /api/v1/services/aigc/image-generation/generation
	dashScopeBaseURL := c.Config.BailianBaseURL
	// 如果配置的是 compatible-mode URL，替换为原生 API URL
	dashScopeBaseURL = strings.Replace(dashScopeBaseURL, "/compatible-mode/v1", "", 1)
	if dashScopeBaseURL == "" || dashScopeBaseURL == "https://dashscope.aliyuncs.com/compatible-mode/v1" {
		dashScopeBaseURL = "https://dashscope.aliyuncs.com"
	}
	url := strings.TrimRight(dashScopeBaseURL, "/") + "/api/v1/services/aigc/image-generation/generation"

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Config.BailianKey)
	// 使用同步模式，直接返回结果
	httpReq.Header.Set("X-DashScope-Async", "disable")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bailian image api error (status %d): %s", resp.StatusCode, string(body))
	}

	var result bailianImageResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(body))
	}

	if result.Code != "" && result.Code != "0" {
		return nil, fmt.Errorf("bailian api error: code=%s, message=%s", result.Code, result.Message)
	}

	// 提取图片 URL
	var urls []string
	for _, r := range result.Output.Results {
		if r.URL != "" {
			urls = append(urls, r.URL)
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("bailian returned no images (task_status: %s)", result.Output.TaskStatus)
	}

	return urls, nil
}
