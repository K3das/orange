package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	workerswhisper "github.com/K3das/orange/asr/workers-whisper"
	"github.com/K3das/orange/discord"
	"github.com/K3das/orange/messages"
	"github.com/K3das/orange/store"
	"github.com/caarlos0/env/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

var CommitHash = ""

type config struct {
	PostgresDSN string `env:"POSTGRES_DSN,required"`

	DiscordToken string   `env:"DISCORD_TOKEN,required"`
	Servers      []string `env:"SERVERS,required"`

	WorkersWhisperOptions workerswhisper.WorkersWhisperClientOptions `envPrefix:"ASR_WORKERS_WHISPER_"`
}

const environmentPrefix = "ORANGE_"
const logLevelEnvKey = environmentPrefix + "LOG_LEVEL"

func createLog() *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = ""

	logLevelValue := os.Getenv(logLevelEnvKey)
	logLevel, logLevelErr := zapcore.ParseLevel(logLevelValue)

	if logLevelErr != nil {
		logLevel = zapcore.InfoLevel
	}

	rawLog := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		logLevel,
	)).Named("orange")

	if CommitHash != "" {
		rawLog = rawLog.With(zap.String("commit", CommitHash))
	}

	if logLevelErr != nil && logLevelValue != "" {
		rawLog.With(zap.String(logLevelEnvKey, logLevelValue)).Warn("unable to parse log level, using INFO")
	}

	return rawLog
}

func main() {
	parentLogger := createLog()
	defer parentLogger.Sync()

	log := parentLogger.Named("main")
	log.With(zap.String("min_log_level", parentLogger.Level().String())).Info("starting")

	cfg := config{}
	if err := env.ParseWithOptions(&cfg, env.Options{
		Prefix: environmentPrefix,
	}); err != nil {
		log.Fatal("failed to parse config", zap.Error(err))
	}

	s := store.NewStore(context.Background(), parentLogger)
	err := s.Connect(context.Background(), cfg.PostgresDSN)
	if err != nil {
		log.Fatal("failed to connect store", zap.Error(err))
	}

	messageProvider, err := messages.NewMessageProvider()
	if err != nil {
		log.Fatal("failed to create message provider", zap.Error(err))
	}

	asrClient := workerswhisper.NewWorkersWhisperClient(cfg.WorkersWhisperOptions)

	discordBot, err := discord.NewDiscordBot(context.Background(), discord.DiscordBotOptions{
		Token:        cfg.DiscordToken,
		Servers:      cfg.Servers,
		ParentLogger: parentLogger,
		Store:        s,
		Messages:     messageProvider,
		ASR:          asrClient,
	})
	if err != nil {
		log.Fatal("failed to create discord bot", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := errgroup.Group{}

	// Discord bot
	g.Go(func() error {
		defer cancel()

		return discordBot.Run(ctx)
	})

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-shutdownSignal:
		cancel()
		log.Info("received signal, shutting down")
	case <-ctx.Done():
		log.Info("context done, shutting down")
	}

	err = g.Wait()
	if err != nil {
		log.Fatal("error group error", zap.Error(err))
	}
}
