package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"luna/config"
	"net/http"

	aiplatform "cloud.google.com/go/aiplatform/apiv1beta1"
	"cloud.google.com/go/aiplatform/apiv1beta1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

// Client はAIサービスとのやり取りを管理します。
type Client struct {
	vertexClient *aiplatform.PredictionClient
	projectID    string
	location     string
}

// NewClient は新しいAIクライアントを作成します。
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	if cfg.Google.ProjectID == "" || cfg.Google.CredentialsPath == "" {
		return nil, fmt.Errorf("Google ProjectIDまたはCredentialsPathが設定されていません")
	}

	location := "us-central1" // Imagen 4 is available in us-central1
	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)

	// Vertex AI Client
	vertexClient, err := aiplatform.NewPredictionClient(ctx, option.WithCredentialsFile(cfg.Google.CredentialsPath), option.WithEndpoint(endpoint))
	if err != nil {
		return nil, fmt.Errorf("Vertex AIクライアントの作成に失敗しました: %w", err)
	}

	return &Client{
		vertexClient: vertexClient,
		projectID:    cfg.Google.ProjectID,
		location:     location,
	}, nil
}

// Close はクライアントをクローズします。
func (c *Client) Close() {
	c.vertexClient.Close()
}

// --- Private Helper ---

func (c *Client) predictText(ctx context.Context, modelID string, prompt string, mimeType string, imageData []byte) (string, error) {
	var instances []*structpb.Value
	var parts []*structpb.Value

	// Prompt Part
	textPart, _ := structpb.NewStruct(map[string]interface{}{"text": prompt})
	parts = append(parts, structpb.NewStructValue(textPart))

	// Image Part (if exists)
	if imageData != nil {
		imgPart, _ := structpb.NewStruct(map[string]interface{}{
			"inline_data": base64.StdEncoding.EncodeToString(imageData),
			"mime_type":   mimeType,
		})
		parts = append(parts, structpb.NewStructValue(imgPart))
	}

	// Instance
	instance, _ := structpb.NewStruct(map[string]interface{}{
		"messages": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
			structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
				"role":  structpb.NewStringValue("user"),
				"parts": structpb.NewListValue(&structpb.ListValue{Values: parts}),
			}}),
		}}),
	})
	instances = append(instances, structpb.NewStructValue(instance))

	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", c.projectID, c.location, modelID)

	req := &aiplatformpb.PredictRequest{
		Endpoint:  endpoint,
		Instances: instances,
	}

	resp, err := c.vertexClient.Predict(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Vertex AI予測リクエストに失敗: %w", err)
	}

	if len(resp.Predictions) == 0 {
		return "", fmt.Errorf("AIから有効な応答がありませんでした")
	}

	content, ok := resp.Predictions[0].GetStructValue().Fields["content"]
	if !ok {
		return "", fmt.Errorf("応答にcontentフィールドが含まれていません")
	}

	return content.GetStringValue(), nil
}

// --- Public Methods ---

// GenerateText は、与えられたプロンプトに基づいてテキストを生成します。
func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	return c.predictText(ctx, "gemini-2.5-pro", prompt, "", nil)
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
	
	return c.predictText(ctx, "gemini-2.5-pro", prompt, resp.Header.Get("Content-Type"), imageData)
}

// GenerateImage は、与えられたプロンプトに基づいて画像を生成します。
func (c *Client) GenerateImage(ctx context.Context, prompt string) (string, error) {
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

	resp, err := c.vertexClient.Predict(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Vertex AI (Imagen)予測リクエストに失敗: %w", err)
	}

	if len(resp.Predictions) == 0 {
		return "", fmt.Errorf("AIから有効な応答がありませんでした")
	}

	// Assuming the response contains a base64 encoded image
	bytes, ok := resp.Predictions[0].GetStructValue().Fields["bytesBase64Encoded"]
	if !ok {
		return "", fmt.Errorf("応答にbytesBase64Encodedフィールドが含まれていません")
	}

	return bytes.GetStringValue(), nil
}
