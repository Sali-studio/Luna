package events

import (
	"luna/interfaces"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// OnInteractionCreate は、すべてのインタラクションを処理する中央ハブです。
func OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate, commandHandlers, componentHandlers map[string]interfaces.CommandHandler, log interfaces.Logger) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h.Handle(s, i)
		} else {
			log.Warn("Unknown command received", "command", i.ApplicationCommandData().Name)
		}
	case discordgo.InteractionMessageComponent:
		if h, ok := componentHandlers[i.MessageComponentData().CustomID]; ok {
			h.HandleComponent(s, i)
		} else {
			// Handle dynamic component IDs (e.g., with prefixes)
			for prefix, handler := range componentHandlers {
				if strings.HasPrefix(i.MessageComponentData().CustomID, prefix) {
					handler.HandleComponent(s, i)
					return
				}
			}
			log.Warn("Unknown component interaction received", "customID", i.MessageComponentData().CustomID)
		}
	case discordgo.InteractionModalSubmit:
		if h, ok := componentHandlers[i.ModalSubmitData().CustomID]; ok {
			h.HandleModal(s, i)
		} else {
			// Handle dynamic modal IDs (e.g., with prefixes)
			for prefix, handler := range componentHandlers {
				if strings.HasPrefix(i.ModalSubmitData().CustomID, prefix) {
					handler.HandleModal(s, i)
					return
				}
			}
			log.Warn("Unknown modal submission received", "customID", i.ModalSubmitData().CustomID)
		}
	}
}
