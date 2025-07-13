package handlers

import (
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// This file will be removed after all event handlers are migrated.

type EventHandler struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func NewEventHandler(store interfaces.DataStore, log interfaces.Logger) *EventHandler {
	return &EventHandler{Store: store, Log: log}
}

func (h *EventHandler) RegisterAllHandlers(s *discordgo.Session) {
	// All event handlers are now being migrated to their own files
	// in the handlers/events directory.
}
