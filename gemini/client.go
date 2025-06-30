package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time" // ★★★ time パッケージをインポート ★★★
)

// Gemini APIに送信するリクエストの構造体
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

// Gemini APIから返ってくるレスポンスの構造体
type GeminiResponse struct {
	Candidates     []Candidate    `json:"candidates"`
	PromptFeedback PromptFeedback `json:"promptFeedback,omitempty"`
}

// プロンプトがブロックされた際の情報
type PromptFeedback struct {
	BlockReason string `json:"blockReason"`
}

// 各構造体の詳細定義
type Content struct {
	Parts []Part `json:"parts"`
}
type Part struct {
	Text string `json:"text"`
}
type Candidate struct {
	Content Content `json:"content"`
}

// GenerateContent はGemini APIにリクエストを送信し、応答を返します
func GenerateContent(apiKey, prompt string) (string, error) {
	apiURL := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent?key=" + apiKey

	reqBody := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 30秒のタイムアウトを設定したHTTPクライアントを作成
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if geminiResp.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("この質問は安全上の理由からブロックされました (理由: %s)", geminiResp.PromptFeedback.BlockReason)
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "すみません、AIから有効な応答を取得できませんでした。", nil
}
