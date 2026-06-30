package openai

import (
	"fmt"
	"net/http"
	"strings"
)

// SiliconFlowClient 硅基流动 API 客户端 (OpenAI 兼容)
type SiliconFlowClient struct {
	Config Config
	client *http.Client
}

// GenerateScript 实现文本/剧本生成 (OpenAI 兼容格式)
func (c *SiliconFlowClient) GenerateScript(req ScriptRequest) (string, error) {
	payload := openAIChatReq{
		Model:     c.Config.SiliconFlowModel,
		Messages:  req.Messages,
		MaxTokens: req.MaxTokens,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.Config.SiliconFlowKey,
	}

	baseURL := c.Config.SiliconFlowBaseURL
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}
	url := strings.TrimRight(baseURL, "/") + "/chat/completions"

	resp, err := doRequest[*openAIChatResp](c.client, "POST", url, headers, payload)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("siliconflow returned no choices")
}

// GenerateImage 实现图片生成 (OpenAI 兼容格式)
func (c *SiliconFlowClient) GenerateImage(req ImageRequest) ([]string, error) {
	model := c.Config.SiliconFlowImageModel
	if model == "" {
		return nil, fmt.Errorf("SILICONFLOW_IMAGE_MODEL is not configured in .env")
	}

	n := req.N
	if n <= 0 {
		n = 1
	}

	size := req.Size
	if size == "" {
		size = "1024x1024"
	}

	payload := openAIImageReq{
		Model:  model,
		Prompt: req.Prompt,
		N:      n,
		Size:   size,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.Config.SiliconFlowKey,
	}

	baseURL := c.Config.SiliconFlowBaseURL
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}
	url := strings.TrimRight(baseURL, "/") + "/images/generations"

	resp, err := doRequest[*openAIImageResp](c.client, "POST", url, headers, payload)
	if err != nil {
		return nil, err
	}

	var urls []string
	for _, item := range resp.Data {
		urls = append(urls, item.URL)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("siliconflow returned empty image list")
	}

	return urls, nil
}
