package gemini

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Client はGemini APIとの通信を管理します
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// APIリクエストの構造体
type geminiRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

// APIレスポンスの構造体
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
	} `json:"candidates"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// NewClient は新しいGeminiクライアントを作成します
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("Gemini APIキーが提供されていません")
	}
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}, nil
}

// GenerateContent はプロンプトを基にテキストを生成します
func (c *Client) GenerateContent(prompt string) (string, error) {
	apiURL := "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=" + c.apiKey

	reqBody := geminiRequest{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Parts: []struct {
					Text string `json:"text"`
				}{
					{Text: prompt},
				},
			},
		},
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("リクエストJSONの作成に失敗: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("HTTPリクエストの作成に失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("APIへのリクエストに失敗: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("レスポンスボディの読み込みに失敗: %w", err)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("レスポンスJSONのパースに失敗: %w", err)
	}

	if geminiResp.Error.Message != "" {
		return "", fmt.Errorf("APIエラー: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", errors.New("AIから有効な応答がありませんでした")
}
