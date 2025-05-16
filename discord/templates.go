package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/K3das/orange/utils"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type MessageOutput struct {
	Content    string                       `json:"content,omitempty"`
	Components []discordgo.MessageComponent `json:"components,omitempty"`
	Embeds     []*discordgo.MessageEmbed    `json:"embeds,omitempty"`
}

type messageOutputRaw struct {
	Content    string                    `json:"content,omitempty"`
	Components []json.RawMessage         `json:"components,omitempty"`
	Embeds     []*discordgo.MessageEmbed `json:"embeds,omitempty"`
}

type MessageContextUserSettings struct {
	ASREnabled            bool   `json:"asr_enabled"`
	ASREnableComponentID  string `json:"asr_enable_component_id"`
	ASRDisableComponentID string `json:"asr_disable_component_id"`
}
type MessageContextUserSettingsToggleResponse struct {
	Enabled bool   `json:"enabled"`
	Changed bool   `json:"changed"`
	Setting string `json:"setting"`
}
type MessageContextInteractionError struct {
	Message string `json:"message"`
}
type MessageContextCommandError struct {
	Message string `json:"message"`
}
type MessageContextCommandCreateHookResponse struct {
	Hook *discordgo.Webhook `json:"hook"`
}

type MessageContextAsrError struct {
	Message string `json:"message"`
}
type MessageContextAsrProgress struct{}
type MessageContextAsrResult struct {
	Text          string             `json:"text"`
	CallerMessage *discordgo.Message `json:"caller_message"`
	Duration      float64            `json:"duration"`
}
type MessageContextAsrNudge struct {
	ASREnableComponentID string `json:"asr_enable_component_id"`

	Guild     *discordgo.Guild `json:"guild"`
	ChannelID string           `json:"channel_id"`
}

type MessageContext struct {
	UserSettings               *MessageContextUserSettings               `json:"user_settings"`
	UserSettingsToggleResponse *MessageContextUserSettingsToggleResponse `json:"user_settings_toggle_response"`
	InteractionError           *MessageContextInteractionError           `json:"interaction_error"`
	CommandError               *MessageContextCommandError               `json:"command_error"`
	CommandCreateHookResponse  *MessageContextCommandCreateHookResponse  `json:"command_create_hook_response"`

	AsrError    *MessageContextAsrError    `json:"asr_error,omitempty"`
	AsrProgress *MessageContextAsrProgress `json:"asr_progress,omitempty"`
	AsrResult   *MessageContextAsrResult   `json:"asr_result,omitempty"`
	AsrNudge    *MessageContextAsrNudge    `json:"asr_nudge,omitempty"`

	Timestamp          string                                   `json:"timestamp"`
	RegisteredCommands map[string]*discordgo.ApplicationCommand `json:"registered_commands"`
}

func (b *DiscordBot) executeMessageTemplate(ctx context.Context, messageName string, data MessageContext) (*MessageOutput, error) {
	log := utils.GetLogFromContext(ctx, b.log)

	data.Timestamp = time.Now().UTC().Format(time.RFC3339)
	b.commandsMu.RLock()
	data.RegisteredCommands = b.commands
	defer b.commandsMu.RUnlock()

	jsonOut, err := b.messages.ExecuteMessage(messageName, data)
	if err != nil {
		return nil, err
	}

	var outputRaw messageOutputRaw
	err = json.Unmarshal([]byte(jsonOut), &outputRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling output: %w", err)
	}

	output := &MessageOutput{
		Content: outputRaw.Content,
		Embeds:  outputRaw.Embeds,
	}

	if outputRaw.Components != nil {
		for _, c := range outputRaw.Components {
			bytes, err := c.MarshalJSON()
			if err != nil {
				return nil, fmt.Errorf("marshaling component: %w", err)
			}
			messageComponent, err := discordgo.MessageComponentFromJSON(bytes)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling component: %w", err)
			}
			output.Components = append(output.Components, messageComponent)
		}
	}

	log.With(zap.Any("output", output)).Debug("got message template output")

	return output, nil
}
