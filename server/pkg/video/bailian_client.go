package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BailianVideoClient 阿里百炼视频生成客户端 (DashScope 原生 API)
type BailianVideoClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type bailianVideoReq struct {
	Model      string              `json:"model"`
	Input      bailianVideoInput   `json:"input"`
	Parameters bailianVideoParams  `json:"parameters,omitempty"`
}

type bailianVideoInput struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
}

type bailianVideoParams struct {
	Duration int    `json:"duration,omitempty"`
	Size     string `json:"size,omitempty"`
}

type bailianVideoResp struct {
	Output struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
		VideoURL   string `json:"video_url"`
	} `json:"output"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

type bailianTaskResp struct {
	Output struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
		VideoURL   string `json:"video_url"`
	} `json:"output"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

func NewBailianVideoClient(baseURL, apiKey, model string) *BailianVideoClient {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com"
	}
	if model == "" {
		model = "wanx2.0-t2v-advanced"
	}
	return &BailianVideoClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Model:      model,
		HTTPClient: defaultHTTPClient(),
	}
}

// GenerateVideo 发起视频生成请求
func (c *BailianVideoClient) GenerateVideo(prompt string, opts ...VideoOption) (*VideoResult, error) {
	options := &VideoOptions{
		Duration:    5,
		AspectRatio: "16:9",
	}
	for _, opt := range opts {
		opt(options)
	}

	model := c.Model
	if options.Model != "" {
		model = options.Model
	}

	// 百炼图片尺寸使用 "*" 而非 "x"
	size := options.AspectRatio
	if size == "16:9" {
		size = "1280*720"
	} else if size == "9:16" {
		size = "720*1280"
	} else if size == "1:1" {
		size = "1024*1024"
	}

	payload := bailianVideoReq{
		Model: model,
		Input: bailianVideoInput{
			Prompt: prompt,
		},
		Parameters: bailianVideoParams{
			Duration: options.Duration,
			Size:     size,
		},
	}

	jsonBytes, _ := json.Marshal(payload)

	baseURL := strings.TrimRight(c.BaseURL, "/")
	url := baseURL + "/api/v1/services/aigc/video-generation/generation"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("X-DashScope-Async", "enable")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bailian video api error (status %d): %s", resp.StatusCode, string(body))
	}

	var result bailianVideoResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Code != "" && result.Code != "0" {
		return nil, fmt.Errorf("bailian api error: code=%s, message=%s", result.Code, result.Message)
	}

	status := result.Output.TaskStatus
	if status == "" {
		status = "pending"
	}

	return &VideoResult{
		TaskID:    result.Output.TaskID,
		Status:    status,
		VideoURL:  result.Output.VideoURL,
		Completed: status == "SUCCEEDED" || status == "completed",
		Duration:  options.Duration,
	}, nil
}

// GetTaskStatus 查询视频任务状态
func (c *BailianVideoClient) GetTaskStatus(taskID string) (*VideoResult, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	url := baseURL + "/api/v1/tasks/" + taskID

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bailian task api error (status %d): %s", resp.StatusCode, string(body))
	}

	var result bailianTaskResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	status := result.Output.TaskStatus

	return &VideoResult{
		TaskID:    taskID,
		Status:    status,
		VideoURL:  result.Output.VideoURL,
		Completed: status == "SUCCEEDED" || status == "completed",
	}, nil
}
