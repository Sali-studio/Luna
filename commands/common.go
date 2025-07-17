// commands/common.go
package commands

// Pythonサーバーに送るリクエストの共通構造体
type TextRequest struct {
	Prompt string `json:"prompt"`
}

// Pythonサーバーから返ってくるレスポンスの共通構造体
type TextResponse struct {
	Text  string `json:"text"`
	Error string `json:"error"`
}
