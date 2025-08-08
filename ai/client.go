package ai

import (
	"context"
	"fmt"
	"io"
	"luna/config"
	"net/http"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Client はAIサービスとのやり取りを管理します。
type Client struct {
	genaiClient *genai.Client
}

// NewClient は新しいAIクライアントを作成します。
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	if cfg.Google.APIKey == "" {
		return nil, fmt.Errorf("Google AI APIキーが設定されていません")
	}

	genaiClient, err := genai.NewClient(ctx, option.WithAPIKey(cfg.Google.APIKey))
	if err != nil {
		return nil, fmt.Errorf("GenAIクライアントの作成に失敗しました: %w", err)
	}

	return &Client{
		genaiClient: genaiClient,
	}, nil
}

// Close はクライアントをクローズします。
func (c *Client) Close() {
	c.genaiClient.Close()
}

// GenerateText は、与えられたプロンプトに基づいてテキストを生成します。
func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	model := c.genaiClient.GenerativeModel("gemini-pro")
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("テキスト生成に失敗しました: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("AIから有効な応答がありませんでした")
	}

	return string(resp.Candidates[0].Content.Parts[0].(genai.Text)), nil
}

// GenerateTextFromImage は、画像とプロンプトに基づいてテキストを生成します。
func (c *Client) GenerateTextFromImage(ctx context.Context, prompt string, imageURL string) (string, error) {
	imagePart, err := urlToGenaiPart(imageURL)
	if err != nil {
		return "", fmt.Errorf("画像データの取得に失敗しました: %w", err)
	}

	model := c.genaiClient.GenerativeModel("gemini-pro-vision")
	resp, err := model.GenerateContent(ctx, imagePart, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("画像からのテキスト生成に失敗しました: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("AIから有効な応答がありませんでした")
	}

	return string(resp.Candidates[0].Content.Parts[0].(genai.Text)), nil
}

// urlToGenaiPart は画像URLからgenai.Partを作成します。
func urlToGenaiPart(url string) (genai.Part, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return genai.ImageData(resp.Header.Get("Content-Type"), imageData), nil
}
