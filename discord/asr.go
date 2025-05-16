package discord

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/K3das/orange/store/db"
	"github.com/K3das/orange/utils"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// max attachment file size in bytes
const MaxInputFileSize = 1024 * 1024 * 4
const MaxOutputFileSize = 1024 * 1024 * 10

// the hard limit for the number of seconds audio can be before it's not transcribed
const MaxDuration = 600

const (
	ComponentActionASREnable  = ComponentIDAction("asr_enable")
	ComponentActionASRDisable = ComponentIDAction("asr_disable")

	ComponentSourceNudge = ComponentIDSource("nudge")
)

func (b *DiscordBot) handleMessageCreateASR(ctx context.Context, e *discordgo.MessageCreate) error {
	log := utils.GetLogFromContext(ctx, b.log)

	if (e.Flags&discordgo.MessageFlagsIsVoiceMessage) == 0 || len(e.Attachments) != 1 {
		return nil // not a voice message
	}

	attachment := e.Attachments[0]

	if !strings.HasPrefix(attachment.ContentType, "audio/") {
		return nil // wrong mime type
	}

	if attachment.Size > MaxInputFileSize {
		log.With(zap.Int("attachment_size", attachment.Size)).Info("voice message too big")
		return nil
	}

	if attachment.DurationSecs > MaxDuration {
		log.With(zap.Float64("duration", attachment.DurationSecs)).Info("voice message too long")
		return nil
	}

	member, err := b.store.GetOrCreateUser(context.Background(), e.Author.ID)
	if err != nil {
		return fmt.Errorf("getting member: %w", err)
	}

	if !member.AsrEnabled && !member.AsrNudged && !member.AsrNudgedTouchedAt.Valid {
		err = b.sendASRNudge(ctx, e)
		if err != nil {
			return fmt.Errorf("sending nudge: %w", err)
		}
		return nil
	} else if !member.AsrEnabled {
		return nil
	}

	renderedProgress, err := b.executeMessageTemplate(ctx, "asr_progress", MessageContext{
		AsrProgress: &MessageContextAsrProgress{},
	})
	if err != nil {
		return fmt.Errorf("rendering reply: %w", err)
	}

	replyMessage, err := b.discord.ChannelMessageSendComplex(e.ChannelID, &discordgo.MessageSend{
		Content:         renderedProgress.Content,
		Embeds:          renderedProgress.Embeds,
		Components:      renderedProgress.Components,
		Reference:       e.Reference(),
		AllowedMentions: DefaultAllowedMentions,
	}, discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("sending initial reply message: %w", err)
	}

	go func() {
		defer utils.PanicRecovery(log)

		ctx := utils.LogContext(context.Background(), utils.GetLogContextFields(ctx)...)

		transcriptionCtx, cancel := context.WithTimeout(
			ctx,
			time.Minute*2,
		)
		defer cancel()

		transcriptionErr := b.startTranscription(transcriptionCtx, attachment, e.Message, replyMessage)
		if transcriptionErr == nil {
			return
		}
		log.Error("failed to transcribe message", zap.Error(transcriptionErr))

		if _, err := b.store.UpdateTranscriptionFailed(ctx, db.UpdateTranscriptionFailedParams{
			GuildID:           e.Message.GuildID,
			ChannelID:         e.Message.ChannelID,
			OriginalMessageID: e.Message.ID,
		}); err != nil {
			log.Error("failed to mark transcription as failed in db", zap.Error(err))
		}

		var discordErr DiscordExecutionError
		errorMessage := "Unknown error occurred"
		if errors.As(transcriptionErr, &discordErr) && discordErr.Message != "" {
			errorMessage = discordErr.Message
		} else if errors.Is(transcriptionErr, context.DeadlineExceeded) {
			errorMessage = "Timeout exceeded while transcribing message."
		}

		renderedError, err := b.executeMessageTemplate(ctx, "asr_error", MessageContext{
			AsrError: &MessageContextAsrError{
				Message: errorMessage,
			},
		})
		if err != nil {
			log.Error("failed to render error message", zap.Error(err))
			return
		}

		if _, err := b.discord.ChannelMessageEditComplex(
			&discordgo.MessageEdit{
				Channel: replyMessage.ChannelID,
				ID:      replyMessage.ID,

				Content:    &renderedError.Content,
				Embeds:     &renderedError.Embeds,
				Components: &renderedError.Components,
			},
		); err != nil {
			log.Error("failed to update reply message with transcription error", zap.Error(err))
		}
	}()

	return nil
}

// startTranscription performs the transcription after the reply was sent,
// creating the transcription in the database
//
// It is the caller's responsibility to mark the transcription as errored if
// it fails.
func (b *DiscordBot) startTranscription(ctx context.Context, attachment *discordgo.MessageAttachment, callerMessage, replyMessage *discordgo.Message) error {
	err := b.store.CreateStartedTranscription(ctx, db.CreateStartedTranscriptionParams{
		GuildID:                callerMessage.GuildID,
		ChannelID:              callerMessage.ChannelID,
		OriginalMessageID:      callerMessage.ID,
		OriginalMessageDeleted: false,
		OriginalMessageTimestamp: pgtype.Timestamptz{
			Time:  callerMessage.Timestamp,
			Valid: true,
		},
		ResponseMessageID: replyMessage.ID,
	})
	if err != nil {
		return fmt.Errorf("creating in db: %w", err)
	}

	tempfile, err := b.downloadAttachmentToTemp(ctx, attachment.URL, MaxInputFileSize)
	if errors.Is(err, utils.ErrIOLimitReached) {
		return fmt.Errorf("attachment too big: %w", err)
	} else if err != nil {
		return DiscordExecutionError{
			Message: "Error downloading file.",
			Err:     fmt.Errorf("downloading attachment: %w", err),
		}
	}
	defer os.Remove(tempfile)

	start := time.Now()

	duration, err := b.ffmpeg.FFprobeDurationFromFile(ctx, tempfile)
	if err != nil {
		return fmt.Errorf("duration: %w", err)
	}
	if duration > MaxDuration {
		return fmt.Errorf("file too long: %fs", duration)
	}

	outputData, err := b.ffmpeg.FFmpegResampleAudioFromFile(ctx, tempfile, MaxOutputFileSize)
	if err != nil {
		return fmt.Errorf("resampling: %w", err)
	}

	transcriptionOutput, err := b.asrAPI.Run(ctx, outputData)
	if err != nil {
		return DiscordExecutionError{
			Message: "Error generating transcript.",
			Err:     fmt.Errorf("generating transcript: %w", err),
		}
	}

	processingTime := time.Since(start).Seconds()

	dbTranscription, err := b.store.UpdateTranscriptionDone(ctx, db.UpdateTranscriptionDoneParams{
		GuildID:           callerMessage.GuildID,
		ChannelID:         callerMessage.ChannelID,
		OriginalMessageID: callerMessage.ID,
		TranscriptionModel: pgtype.Text{
			String: transcriptionOutput.ModelName,
			Valid:  true,
		},
		VoiceMessageAudioDuration: pgtype.Float8{
			Float64: duration,
			Valid:   true,
		},
		TranscriptionProcessingTime: pgtype.Float8{
			Float64: processingTime,
			Valid:   true,
		},
	})
	if err != nil {
		return fmt.Errorf("getting transcription status from db: %w", err)
	}

	if !dbTranscription.ResponseDeleted {
		renderedResponse, err := b.executeMessageTemplate(ctx, "asr_result",
			MessageContext{
				AsrResult: &MessageContextAsrResult{
					Text:          transcriptionOutput.Text,
					CallerMessage: callerMessage,
					Duration:      processingTime,
				},
			},
		)
		if err != nil {
			return fmt.Errorf("rendering message: %w", err)
		}

		_, err = b.discord.ChannelMessageEditComplex(
			&discordgo.MessageEdit{
				Channel: replyMessage.ChannelID,
				ID:      replyMessage.ID,

				Content:    &renderedResponse.Content,
				Embeds:     &renderedResponse.Embeds,
				Components: &renderedResponse.Components,
			},
			discordgo.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("editing message: %w", err)
		}
	}
	return nil
}

// downloadAttachmentToTemp downloads url into a temp file with a size limit, returning the path.
//
// It is the caller's responsibility to clean up the temp file.
func (b *DiscordBot) downloadAttachmentToTemp(ctx context.Context, url string, maxSize int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := b.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad http status: %s", resp.Status)
	}

	tempFile, err := os.CreateTemp("", "orange-*")
	if err != nil {
		return "", fmt.Errorf("making temp file: %w", err)
	}

	defer tempFile.Close()
	_, err = utils.CopyLimit(tempFile, resp.Body, int64(maxSize))
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("writing to the temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// sendASRNudge DMs the user a nudge about their voice message and updates
// their nudge status.
func (b *DiscordBot) sendASRNudge(ctx context.Context, e *discordgo.MessageCreate) error {
	log := utils.GetLogFromContext(ctx, b.log)

	err := b.store.UpdateUserASRNudge(
		context.Background(),
		db.UpdateUserASRNudgeParams{
			ID:        e.Author.ID,
			AsrNudged: true,
		},
	)
	if err != nil {
		return fmt.Errorf("updating nudge status in db: %w", err)
	}

	guild, err := b.discord.State.Guild(e.GuildID)
	if err != nil {
		return fmt.Errorf("getting guild: %w", err)
	}

	dm, err := b.discord.UserChannelCreate(e.Author.ID, discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("creating dm channel: %w", err)
	}

	renderedNudge, err := b.executeMessageTemplate(ctx, "asr_nudge", MessageContext{
		AsrNudge: &MessageContextAsrNudge{
			ASREnableComponentID: ComponentIDString(ComponentSourceNudge, ComponentActionASREnable),
			Guild:                guild,
			ChannelID:            e.ChannelID,
		},
	})
	if err != nil {
		return fmt.Errorf("rendering nudge: %w", err)
	}

	_, err = b.discord.ChannelMessageSendComplex(dm.ID, &discordgo.MessageSend{
		Content:         renderedNudge.Content,
		Embeds:          renderedNudge.Embeds,
		Components:      renderedNudge.Components,
		AllowedMentions: DefaultAllowedMentions,
	}, discordgo.WithContext(ctx))
	if err != nil {
		log.Info("couldn't send nudge to user (likely because of DM permissions)", zap.Error(err))
	}

	return nil
}

func (b *DiscordBot) handleMessageDeleteASR(ctx context.Context, d DeletedMessage) error {
	transcription, err := b.store.UpdateTranscriptionMessageDeleted(ctx, db.UpdateTranscriptionMessageDeletedParams{
		GuildID:   d.GuildID,
		ChannelID: d.ChannelID,
		MessageID: d.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return fmt.Errorf("updating original deleted: %w", err)
	}

	if transcription.ResponseDeleted {
		return nil
	}

	err = b.discord.ChannelMessageDelete(d.ChannelID, transcription.ResponseMessageID)
	if err != nil {
		return fmt.Errorf("deleting message: %w", err)
	}

	return nil
}

func (b *DiscordBot) handleASRToggleInteraction(ctx context.Context, id *ComponentID, e *discordgo.InteractionCreate, data discordgo.MessageComponentInteractionData) error {
	discordUser, err := getInteractionUser(e)
	if err != nil {
		return err
	}

	user, err := b.store.GetOrCreateUser(ctx, discordUser.ID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	enableASR := id.Action == ComponentActionASREnable
	changed := user.AsrEnabled != enableASR

	if changed {
		err = b.store.UpdateUserASREnabled(ctx, db.UpdateUserASREnabledParams{
			ID:         discordUser.ID,
			AsrEnabled: enableASR,
		})

		if err != nil {
			return DiscordExecutionError{
				Message: "Couldn't update user",
				Err:     fmt.Errorf("updating user: %w", err),
			}
		}
	}

	response := &discordgo.InteractionResponse{
		Data: &discordgo.InteractionResponseData{
			AllowedMentions: DefaultAllowedMentions,
		},
	}

	if id.Source == ComponentSourceSettings {
		output, err := b.executeMessageTemplate(ctx, "user_settings", MessageContext{
			UserSettings: &MessageContextUserSettings{
				ASREnabled:            enableASR,
				ASREnableComponentID:  ComponentIDString(ComponentSourceSettings, ComponentActionASREnable),
				ASRDisableComponentID: ComponentIDString(ComponentSourceSettings, ComponentActionASRDisable),
			},
		})
		if err != nil {
			return fmt.Errorf("rendering settings: %w", err)
		}

		response.Type = discordgo.InteractionResponseUpdateMessage
		response.Data.Flags = discordgo.MessageFlagsEphemeral
		response.Data.Content = output.Content
		response.Data.Components = output.Components
		response.Data.Embeds = output.Embeds
	} else {
		output, err := b.executeMessageTemplate(ctx, "user_settings_toggle_response", MessageContext{
			UserSettingsToggleResponse: &MessageContextUserSettingsToggleResponse{
				Enabled: enableASR,
				Changed: changed,
				Setting: "asr",
			},
		})
		if err != nil {
			return fmt.Errorf("rendering settings toggle response: %w", err)
		}

		response.Type = discordgo.InteractionResponseChannelMessageWithSource
		response.Data.Content = output.Content
		response.Data.Components = output.Components
		response.Data.Embeds = output.Embeds

		if !(id.Source == ComponentSourceNudge && changed) {
			response.Data.Flags = discordgo.MessageFlagsEphemeral
		}
	}

	err = b.discord.InteractionRespond(e.Interaction, response)
	if err != nil {
		return fmt.Errorf("sending response: %w", err)
	}

	return nil
}
