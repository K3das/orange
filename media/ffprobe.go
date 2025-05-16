package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

var ErrFFprobeDurationInvalid = fmt.Errorf("got no packets from ffprobe, likely a bad file")

type Packet struct {
	CodecType          string  `json:"codec_type"`
	StreamIndex        int     `json:"stream_index"`
	Pts                int     `json:"pts"`
	PtsTime            string  `json:"pts_time"`
	Dts                int     `json:"dts"`
	DtsTime            string  `json:"dts_time"`
	Duration           int     `json:"duration"`
	DurationTime       string  `json:"duration_time"`
	Size               string  `json:"size"`
	Pos                string  `json:"pos"`
	Flags              string  `json:"flags"`
	ParsedPtsTime      float64 `json:"-"`
	ParsedDtsTime      float64 `json:"-"`
	ParsedDurationTime float64 `json:"-"`
}

type FFprobePacketsOutput struct {
	Packets []Packet `json:"packets"`
}

func (f *FFmpeg) ffprobeGetPacketsFromFile(ctx context.Context, filePath string) ([]Packet, error) {
	cmd := exec.CommandContext(ctx,
		f.ffprobeBinary,
		"-i", filePath,
		"-v", "error",
		"-print_format", "json",
		"-show_packets",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running ffprobe: %w", err)
	}

	var response FFprobePacketsOutput
	err = json.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("parsing ffprobe json response: %w", err)
	}

	for i := range response.Packets {
		packet := &response.Packets[i]

		packet.ParsedDtsTime, err = strconv.ParseFloat(packet.DtsTime, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing DtsTime: %w", err)
		}
		packet.ParsedPtsTime, err = strconv.ParseFloat(packet.PtsTime, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing PtsTime: %w", err)
		}
		packet.ParsedDurationTime, err = strconv.ParseFloat(packet.DurationTime, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing DurationTime: %w", err)
		}
	}

	return response.Packets, nil
}

// FFprobeDurationFromFile gets the duration of the input file using ffprobe
//
// Parses packet metadata to determine length: `max pts time + duration time`.
// Returns ErrFFprobeDurationInvalid if no packets.
//
// This uses packet metadata because some containers don't really include duration
// metadata (like the MediaRecorder API's output), and it's more accurate to
// what is processed by the model.
func (f *FFmpeg) FFprobeDurationFromFile(ctx context.Context, filePath string) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, f.commandTimeout)
	defer cancel()

	packets, err := f.ffprobeGetPacketsFromFile(ctx, filePath)
	if err != nil {
		return 0, fmt.Errorf("getting packets: %w", err)
	}

	if len(packets) == 0 {
		return 0, ErrFFprobeDurationInvalid
	}

	var maxPacket Packet
	for _, packet := range packets {
		if packet.ParsedPtsTime > maxPacket.ParsedPtsTime {
			maxPacket = packet
		}
	}

	return maxPacket.ParsedPtsTime + maxPacket.ParsedDurationTime, nil
}
