package discord

import (
	"context"
	"errors"
	"fmt"

	"github.com/K3das/orange/utils"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *DiscordBot) handleComponentInteraction(ctx context.Context, e *discordgo.InteractionCreate, data discordgo.MessageComponentInteractionData) error {
	log := utils.GetLogFromContext(ctx, b.log)

	if data.ComponentType != discordgo.ButtonComponent {
		return nil
	}

	componentID, err := ParseComponentID(data.CustomID)
	if err != nil {
		return nil
	}

	var interactionErr error

	switch {
	case componentID.Action == ComponentActionASREnable || componentID.Action == ComponentActionASRDisable:
		interactionErr = b.handleASRToggleInteraction(ctx, componentID, e, data)
	}

	if interactionErr != nil {
		var discordErr DiscordExecutionError
		errorMessage := "Unknown error occurred."
		if errors.As(interactionErr, &discordErr) && discordErr.Message != "" {
			errorMessage = discordErr.Message
		}

		if !discordErr.UserError {
			log.Error("failed to respond to interaction", zap.Error(interactionErr))
		}

		output, err := b.executeMessageTemplate(ctx, "interaction_error", MessageContext{
			InteractionError: &MessageContextInteractionError{
				Message: errorMessage,
			},
		})
		if err != nil {
			log.Error("failed to render error message", zap.Error(err))
			return nil
		}

		err = b.discord.InteractionRespond(e.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:           discordgo.MessageFlagsEphemeral,
				Content:         output.Content,
				Components:      output.Components,
				Embeds:          output.Embeds,
				AllowedMentions: DefaultAllowedMentions,
			},
		})
		if err != nil {
			log.Error("failed to send response", zap.Error(err))
		}
	}

	return nil
}

func getInteractionUser(e *discordgo.InteractionCreate) (*discordgo.User, error) {
	var discordUser *discordgo.User
	if e.Member != nil && e.Member.User != nil {
		discordUser = e.Member.User
	} else if e.User != nil {
		discordUser = e.User
	} else {
		return nil, fmt.Errorf("no user found in interaction")
	}

	return discordUser, nil
}
