package media

import (
	"time"
)

const DefaultFFmpegBinary = "ffmpeg"
const DefaultFFprobeBinary = "ffprobe"

const DefaultCommandTimeout = time.Second * 30

type FFmpegOptions func(*FFmpeg)

type FFmpeg struct {
	ffmpegBinary   string
	ffprobeBinary  string
	commandTimeout time.Duration
}

func WithFFmpegBinary(ffmpegBinary string) FFmpegOptions {
	return func(f *FFmpeg) {
		f.ffmpegBinary = ffmpegBinary
	}
}

func WithFFprobeBinary(ffprobeBinary string) FFmpegOptions {
	return func(f *FFmpeg) {
		f.ffprobeBinary = ffprobeBinary
	}
}

func WithCommandTimeout(timeout time.Duration) FFmpegOptions {
	return func(f *FFmpeg) {
		f.commandTimeout = timeout
	}
}

func NewFFmpeg(options ...FFmpegOptions) *FFmpeg {
	ffmpeg := &FFmpeg{
		ffmpegBinary:   DefaultFFmpegBinary,
		ffprobeBinary:  DefaultFFprobeBinary,
		commandTimeout: DefaultCommandTimeout,
	}

	for _, option := range options {
		option(ffmpeg)
	}

	return ffmpeg
}
