package discord

import (
	"context"
	"errors"
	"fmt"

	"github.com/K3das/orange/utils"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const (
	CommandNameCreateHook   = "create-owned-hook"
	CommandNameUserSettings = "settings"
)

const (
	ComponentSourceSettings = ComponentIDSource("settings")
)

func (b *DiscordBot) registerCommands(ctx context.Context) error {
	adminPerms := int64(discordgo.PermissionAdministrator)
	defaultPerms := int64(discordgo.PermissionViewChannel)
	createdCommands, err := b.discord.ApplicationCommandBulkOverwrite(b.self.ID, "", []*discordgo.ApplicationCommand{
		{
			Type:                     discordgo.ChatApplicationCommand,
			Name:                     CommandNameCreateHook,
			DefaultMemberPermissions: &adminPerms,
			Description:              "Create an application-owned webhook for sending interaction-supporting messages.",
			Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		},
		{
			Type:                     discordgo.ChatApplicationCommand,
			Name:                     CommandNameUserSettings,
			DefaultMemberPermissions: &defaultPerms,
			Description:              "Configure Orange's features.",
		},
	}, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}

	b.commandsMu.Lock()
	b.commands = make(map[string]*discordgo.ApplicationCommand)
	for _, command := range createdCommands {
		b.commands[command.Name] = command
	}
	b.commandsMu.Unlock()

	return nil
}

func (b *DiscordBot) getCommand(name string) (*discordgo.ApplicationCommand, bool) {
	b.commandsMu.RLock()
	defer b.commandsMu.RUnlock()
	command, ok := b.commands[name]
	return command, ok
}

func (b *DiscordBot) handleCommandInteraction(ctx context.Context, e *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) error {
	log := utils.GetLogFromContext(ctx, b.log)

	var commandErr error
	switch data.Name {
	case CommandNameCreateHook:
		commandErr = b.handleCommandCreateHook(ctx, e, data)
	case CommandNameUserSettings:
		commandErr = b.handleCommandUserSettings(ctx, e, data)
	}

	if commandErr != nil {
		var discordErr DiscordExecutionError
		errorMessage := "Unknown error occurred."
		if errors.As(commandErr, &discordErr) && discordErr.Message != "" {
			errorMessage = discordErr.Message
		}

		if !discordErr.UserError {
			log.Error("failed to respond to command", zap.Error(commandErr))
		}

		output, err := b.executeMessageTemplate(ctx, "command_error", MessageContext{
			CommandError: &MessageContextCommandError{
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

func (b *DiscordBot) handleCommandCreateHook(ctx context.Context, e *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) error {
	if e.Member == nil || e.GuildID == "" {
		return DiscordExecutionError{
			Message:   "Run this command in a server.",
			UserError: true,
		}
	}
	if e.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		return DiscordExecutionError{
			Message:   "You need to be administrator to run this.",
			UserError: true,
		}
	}

	hook, err := b.discord.WebhookCreate(e.ChannelID, "Orange Hook", "")
	if err != nil {
		return DiscordExecutionError{
			Message: "Couldn't create webhook.",
			Err:     err,
		}
	}

	output, err := b.executeMessageTemplate(ctx, "command_create_hook_response", MessageContext{
		CommandCreateHookResponse: &MessageContextCommandCreateHookResponse{
			Hook: hook,
		},
	})
	if err != nil {
		return fmt.Errorf("rendering message: %w", err)
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
		return fmt.Errorf("responding: %s", err)
	}

	return nil
}

func (b *DiscordBot) handleCommandUserSettings(ctx context.Context, e *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) error {
	discordUser, err := getInteractionUser(e)
	if err != nil {
		return err
	}

	user, err := b.store.GetOrCreateUser(ctx, discordUser.ID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	output, err := b.executeMessageTemplate(ctx, "user_settings", MessageContext{
		UserSettings: &MessageContextUserSettings{
			ASREnabled:            user.AsrEnabled,
			ASREnableComponentID:  ComponentIDString(ComponentSourceSettings, ComponentActionASREnable),
			ASRDisableComponentID: ComponentIDString(ComponentSourceSettings, ComponentActionASRDisable),
		},
	})
	if err != nil {
		return fmt.Errorf("rendering message: %w", err)
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
		return fmt.Errorf("responding: %s", err)
	}

	return nil
}
