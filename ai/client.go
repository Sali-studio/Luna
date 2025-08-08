package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"luna/config"
	"net/http"
	"time"

	aiplatform "cloud.google.com/go/aiplatform/apiv1beta1"
	"cloud.google.com/go/aiplatform/apiv1beta1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

// Client はAIサービスとのやり取りを管理します。
type Client struct {
	genClient    *aiplatform.GenerativeClient
	predictClient *aiplatform.PredictionClient
	projectID     string
	location      string
}

// NewClient は新しいAIクライアントを作成します。
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	if cfg.Google.ProjectID == "" || cfg.Google.CredentialsPath == "" {
		return nil, fmt.Errorf("Google ProjectIDまたはCredentialsPathが設定されていません")
	}

	location := "us-central1"
	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)

	// Gemini Client (GenerativeClient)
	genClient, err := aiplatform.NewGenerativeClient(ctx, option.WithCredentialsFile(cfg.Google.CredentialsPath), option.WithEndpoint(endpoint))
	if err != nil {
		return nil, fmt.Errorf("Vertex AI GenerativeClientの作成に失敗しました: %w", err)
	}

	// Imagen Client (PredictionClient)
	predictClient, err := aiplatform.NewPredictionClient(ctx, option.WithCredentialsFile(cfg.Google.CredentialsPath), option.WithEndpoint(endpoint))
	if err != nil {
		return nil, fmt.Errorf("Vertex AI PredictionClientの作成に失敗しました: %w", err)
	}

	return &Client{
		genClient:     genClient,
		predictClient: predictClient,
		projectID:     cfg.Google.ProjectID,
		location:      location,
	}, nil
}

// Close はクライアントをクローズします。
func (c *Client) Close() {
	c.genClient.Close()
	c.predictClient.Close()
}

// --- Public Methods ---

// GenerateText は、与えられたプロンプトに基づいてテキストを生成します。
func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	model := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/gemini-2.5-pro", c.projectID, c.location)
	req := &aiplatformpb.GenerateContentRequest{
		Model: model,
		Contents: []*aiplatformpb.Content{
			{
				Role: "user",
				Parts: []*aiplatformpb.Part{
					{Data: &aiplatformpb.Part_Text{Text: prompt}},
				},
			},
		},
	}

	resp, err := c.genClient.GenerateContent(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Vertex AI (Gemini)からの応答生成に失敗: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("AIから有効な応答がありませんでした")
	}

	return resp.Candidates[0].Content.Parts[0].GetText(), nil
}

// GenerateTextFromImage は、画像とプロンプトに基づいてテキストを生成します。
func (c *Client) GenerateTextFromImage(ctx context.Context, prompt string, imageURL string) (string, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	model := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/gemini-2.5-pro", c.projectID, c.location)
	req := &aiplatformpb.GenerateContentRequest{
		Model: model,
		Contents: []*aiplatformpb.Content{
			{
				Role: "user",
				Parts: []*aiplatformpb.Part{
					{Data: &aiplatformpb.Part_Text{Text: prompt}},
					{Data: &aiplatformpb.Part_InlineData{InlineData: &aiplatformpb.Blob{MimeType: resp.Header.Get("Content-Type"), Data: imageData}}},
				},
			},
		},
	}

	resp, err = c.genClient.GenerateContent(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Vertex AI (Gemini)からの応答生成に失敗: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("AIから有効な応答がありませんでした")
	}

	return resp.Candidates[0].Content.Parts[0].GetText(), nil
}

// GenerateImage は、与えられたプロンプトに基づいて画像を生成します。
func (c *Client) GenerateImage(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	parameters, _ := structpb.NewStruct(map[string]interface{}{
		"sampleCount": 1,
	})

	instance, _ := structpb.NewStruct(map[string]interface{}{
		"prompt": prompt,
	})

	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/imagen-4.0-fast-generate-preview-06-06", c.projectID, c.location)

	req := &aiplatformpb.PredictRequest{
		Endpoint:   endpoint,
		Instances:  []*structpb.Value{structpb.NewStructValue(instance)},
		Parameters: structpb.NewStructValue(parameters),
	}

	resp, err := c.predictClient.Predict(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Vertex AI (Imagen)予測リクエストに失敗: %w", err)
	}

	if len(resp.Predictions) == 0 {
		return "", fmt.Errorf("AIから有効な応答がありませんでした")
	}

	bytesVal, ok := resp.Predictions[0].GetStructValue().Fields["bytesBase64Encoded"]
	if !ok {
		return "", fmt.Errorf("応答にbytesBase64Encodedフィールドが含まれていません")
	}

	return bytesVal.GetStringValue(), nil
}
