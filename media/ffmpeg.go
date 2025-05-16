package media

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/K3das/orange/utils"
)

// FFmpegResampleAudioFromFile resamples the input to 48kHz and encodes as aac, outputting the data as bytes
func (f *FFmpeg) FFmpegResampleAudioFromFile(ctx context.Context, filePath string, maxSize int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, f.commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		f.ffmpegBinary,
		"-i", filePath,
		"-c:a", "aac",
		"-ar:a", "48000",
		"-ac:a", "1",
		"-f", "adts",
		"-",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("starting ffmpeg: %w", err)
	}

	output, err := utils.ReadAllLimit(stdout, maxSize)
	if err != nil {
		return nil, fmt.Errorf("reading output: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("running ffmpeg: %w", err)
	}

	return output, nil
}
