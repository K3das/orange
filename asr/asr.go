package asr

import "context"

type SpeechRecognitionAPI interface {
	Run(ctx context.Context, data []byte) (*ASROutput, error)
}

type ASROutput struct {
	Text      string
	ModelName string
}
