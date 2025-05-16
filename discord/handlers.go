package discord

import (
	"context"
	"fmt"

	"github.com/K3das/orange/utils"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *DiscordBot) handleMessageCreate(s *discordgo.Session, e *discordgo.MessageCreate) {
	ctx, log := utils.LogContextWith(context.Background(), b.log, zap.String("initiating_message", fmt.Sprintf("/%s/%s/%s", e.GuildID, e.ChannelID, e.ID)))

	defer utils.PanicRecovery(log)

	if !b.isGuildInScope(e.GuildID) {
		return // not a supported guild
	}

	if e.Author == nil || e.Author.ID == "" || e.Author.Bot {
		return
	}

	err := b.handleMessageCreateASR(ctx, e)
	if err != nil {
		log.Error("error handling message asr", zap.Error(err))
	}
}

func (b *DiscordBot) handleInteractionCreate(s *discordgo.Session, e *discordgo.InteractionCreate) {
	ctx, log := utils.LogContextWith(context.Background(), b.log, zap.String("initiating_interaction", fmt.Sprintf("/%s/%s/%s", e.GuildID, e.ChannelID, e.ID)))

	defer utils.PanicRecovery(log)

	switch e.Type {
	case discordgo.InteractionApplicationCommand:
		data := e.ApplicationCommandData()
		err := b.handleCommandInteraction(ctx, e, data)
		if err != nil {
			log.Error("error handling command interaction", zap.Error(err))
		}
	case discordgo.InteractionMessageComponent:
		data := e.MessageComponentData()
		err := b.handleComponentInteraction(ctx, e, data)
		if err != nil {
			log.Error("error handling component interaction", zap.Error(err))
		}
	}
}

type DeletedMessage struct {
	GuildID   string
	ChannelID string
	ID        string
}

func (b *DiscordBot) handleMessageDeleteEvent(s *discordgo.Session, e *discordgo.MessageDelete) {
	if !b.isGuildInScope(e.GuildID) {
		return // not a supported guild
	}

	d := DeletedMessage{
		GuildID:   e.GuildID,
		ChannelID: e.ChannelID,
		ID:        e.ID,
	}

	ctx, log := utils.LogContextWith(context.Background(), b.log, zap.String("deleted_message", fmt.Sprintf("/%s/%s/%s", d.GuildID, d.ChannelID, d.ID)))

	err := b.handleMessageDeleteASR(ctx, d)
	if err != nil {
		log.Error("error handling delete asr", zap.Error(err))
	}
}
