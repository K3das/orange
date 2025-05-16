package discord

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/K3das/orange/asr"
	"github.com/K3das/orange/media"
	"github.com/K3das/orange/messages"
	"github.com/K3das/orange/store"
	"github.com/K3das/orange/utils"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const ComponentPrefix = "o:"

var DefaultAllowedMentions = &discordgo.MessageAllowedMentions{
	Parse:       []discordgo.AllowedMentionType{},
	RepliedUser: true,
}

type DiscordExecutionError struct {
	Message string
	Err     error
	// If true, do not log this error
	UserError bool
}

func (err DiscordExecutionError) Error() string {
	if err.Err == nil {
		return err.Message
	}
	return err.Err.Error()
}

func (e *DiscordExecutionError) Unwrap() error {
	return e.Err
}

type DiscordBot struct {
	log *zap.Logger

	discord  *discordgo.Session
	store    *store.Store
	messages *messages.MessageProvider
	asrAPI   asr.SpeechRecognitionAPI

	ffmpeg *media.FFmpeg
	http   *http.Client

	self *discordgo.User

	commands   map[string]*discordgo.ApplicationCommand
	commandsMu sync.RWMutex

	knownServers map[string]struct{}
}

type DiscordBotOptions struct {
	ParentLogger *zap.Logger
	Store        *store.Store
	Messages     *messages.MessageProvider
	ASR          asr.SpeechRecognitionAPI

	Token   string
	Servers []string
}

type DiscordBotOptionsExtraOptions func(*DiscordBot)

func WithFFmpeg(ffmpeg *media.FFmpeg) DiscordBotOptionsExtraOptions {
	return func(b *DiscordBot) {
		b.ffmpeg = ffmpeg
	}
}

func WithHTTPClient(client *http.Client) DiscordBotOptionsExtraOptions {
	return func(b *DiscordBot) {
		b.http = client
	}
}

func NewDiscordBot(ctx context.Context, options DiscordBotOptions, extraOptions ...DiscordBotOptionsExtraOptions) (*DiscordBot, error) {
	b := &DiscordBot{
		log:   options.ParentLogger.Named("discord_bot"),
		store: options.Store,

		messages: options.Messages,
		asrAPI:   options.ASR,
		ffmpeg:   media.NewFFmpeg(),

		http:         http.DefaultClient,
		knownServers: make(map[string]struct{}),
	}
	for _, option := range extraOptions {
		option(b)
	}

	for _, v := range options.Servers {
		b.knownServers[v] = struct{}{}
	}

	discord, err := discordgo.New("Bot " + options.Token)
	if err != nil {
		return nil, fmt.Errorf("creating discordgo instance: %w", err)
	}
	b.discord = discord
	b.discord.Client = b.http

	state := discordgo.NewState()
	state.TrackChannels = false
	state.TrackThreads = false
	state.TrackEmojis = false
	state.TrackStickers = false
	state.TrackMembers = false
	state.TrackThreadMembers = false
	state.TrackRoles = false
	state.TrackVoice = false
	state.TrackPresences = false
	b.discord.State = state
	b.discord.StateEnabled = true

	b.discord.AddHandler(b.handleReady)
	b.discord.AddHandler(b.handleMessageCreate)
	b.discord.AddHandler(b.handleInteractionCreate)
	b.discord.AddHandler(b.handleMessageDeleteEvent)

	b.discord.Identify.Presence = discordgo.GatewayStatusUpdate{
		Game: discordgo.Activity{
			Name:  "🍊",
			Type:  discordgo.ActivityTypeCustom,
			State: "🍊",
		},
	}

	b.self, err = b.discord.User("@me", discordgo.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("checking discord session: %w", err)
	}

	b.log = b.log.With(zap.String("bot_id", b.self.ID))
	b.log.Info("discord api works")

	err = b.registerCommands(ctx)
	if err != nil {
		return nil, fmt.Errorf("registering commands: %w", err)
	}

	return b, nil
}

func (b *DiscordBot) handleReady(s *discordgo.Session, e *discordgo.Ready) {
	b.log.Info("gateway ready")
}

func (b *DiscordBot) Open() error {
	return b.discord.Open()
}

func (b *DiscordBot) Close() error {
	return b.discord.Close()
}

func (b *DiscordBot) Run(ctx context.Context) error {
	defer utils.PanicRecovery(b.log)

	err := b.Open()
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	<-ctx.Done()

	err = b.Close()
	if err != nil {
		return fmt.Errorf("closing discord websocket: %w", err)
	}

	return nil
}

func (b *DiscordBot) isGuildInScope(guildID string) bool {
	_, ok := b.knownServers[guildID]
	return ok
}
