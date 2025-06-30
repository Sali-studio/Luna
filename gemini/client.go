package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Gemini APIに送信するリクエストの構造体
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

// Gemini APIから返ってくるレスポンスの構造体
type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
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

	// リクエストボディを作成
	reqBody := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}
	
	// Goの構造体をJSONに変換
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	
	// HTTPリクエストを作成
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// HTTPリクエストを送信
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send http request: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスボディを読み込み
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// JSONレスポンスをGoの構造体に変換
	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	
	// 応答テキストを抽出
	if len(geminiResp.Candidates) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "すみません、応答を取得できませんでした。", nil
}