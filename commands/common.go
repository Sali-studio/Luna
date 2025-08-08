// commands/common.go
package commands

import "github.com/bwmarrin/discordgo"

// Pythonサーバーに送るリクエストの共通構造体
type TextRequest struct {
	Prompt string `json:"prompt"`
}

// Pythonサーバーから返ってくるレスポンスの共通構造体
type TextResponse struct {
	Text  string `json:"text"`
	Error string `json:"error"`
}

// --- Helper Functions for Responses ---

// sendEmbedResponse sends a public embed response.
func sendEmbedResponse(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// sendErrorResponse sends a public error message.
func sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func sendSuccessResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ " + message,
		},
	})
}
