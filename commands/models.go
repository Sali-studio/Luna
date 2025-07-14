package commands

// This file contains shared request/response structs for multiple commands
// to avoid redeclaration errors.

// TextRequest is a generic request for text generation.
type TextRequest struct {
	Prompt string `json:"prompt"`
}

// TextResponse is a generic response for text generation.
type TextResponse struct {
	Text  string `json:"text"`
	Error string `json:"error"`
}
