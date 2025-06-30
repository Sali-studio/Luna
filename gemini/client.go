package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GeminiRequest struct {
	Contents         []Content        `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
	SafetySettings   []SafetySetting  `json:"safetySettings"`
}
type GenerationConfig struct {
	Temperature     float32 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}
type GeminiResponse struct {
	Candidates     []Candidate    `json:"candidates"`
	PromptFeedback PromptFeedback `json:"promptFeedback,omitempty"`
}
type PromptFeedback struct {
	BlockReason string `json:"blockReason"`
}
type Content struct {
	Parts []Part `json:"parts"`
}
type Part struct {
	Text string `json:"text"`
}
type Candidate struct {
	Content Content `json:"content"`
}

// sendRequestToGemini はAPIへのリクエスト送信をまとめた共通関数
func sendRequestToGemini(apiKey, prompt string) (GeminiResponse, error) {
	var geminiResp GeminiResponse
	apiURL := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent?key=" + apiKey

	reqBody := GeminiRequest{
		Contents: []Content{{Parts: []Part{{Text: prompt}}}},
		GenerationConfig: GenerationConfig{
			Temperature:     0.7,
			MaxOutputTokens: 1000,
		},
		SafetySettings: []SafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return geminiResp, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return geminiResp, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return geminiResp, fmt.Errorf("failed to send http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return geminiResp, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return geminiResp, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return geminiResp, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return geminiResp, nil
}

// GenerateContent は一般的な質問に対する応答を生成します (/askコマンド用)
func GenerateContent(apiKey, prompt string) (string, error) {
	geminiResp, err := sendRequestToGemini(apiKey, prompt)
	if err != nil {
		return "", err
	}

	if geminiResp.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("この質問は安全上の理由からブロックされました (理由: %s)", geminiResp.PromptFeedback.BlockReason)
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "すみません、AIから有効な応答を取得できませんでした。", nil
}

// GenerateTicketResponse はチケット対応に特化した応答を生成します
func GenerateTicketResponse(apiKey, subject, details string) (string, error) {
	// チケット対応用の、より詳細なプロンプトを作成
	prompt := fmt.Sprintf(`あなたは、優秀なDiscordサーバーのモデレーターです。
以下のユーザーからの問い合わせに対して、Discordサーバーのモデレーターとして、丁寧かつ適切な一次回答を生成してください。

### 前提条件
- この問い合わせはDiscordサーバー内で発生したものです。「サイト」ではなく「サーバー」という言葉を使用してください。
- あなたが提案できる具体的な対処法は「タイムアウト」「Kick」「BAN」の3種類です。「アカウント停止」のような、サーバー管理者には実行不可能な対応は提案しないでください。

### ユーザーからの問い合わせ内容
件名: %s
詳細: %s

### 回答`, subject, details)

	geminiResp, err := sendRequestToGemini(apiKey, prompt)
	if err != nil {
		return "", err
	}

	if geminiResp.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("この質問は安全上の理由からブロックされました (理由: %s)", geminiResp.PromptFeedback.BlockReason)
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "AIによる一次回答の生成中にエラーが発生しました。", nil
}
