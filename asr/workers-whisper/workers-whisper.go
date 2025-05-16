package workerswhisper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/K3das/orange/asr"
)

// used for the model name in the database
const apiPrefix = "workers_whisper-"

type CloudflareResponse[T any] struct {
	Result   *T    `json:"result"`
	Success  bool  `json:"success"`
	Errors   []any `json:"errors"`
	Messages []any `json:"messages"`
}

type SpeechRecognitionResponse struct {
	// The transcription
	Text      string  `json:"text"`
	Vtt       string  `json:"vtt"`
	WordCount float64 `json:"word_count"`
	// HACK: https://discord.com/channels/595317990191398933/1105477009964027914/1370918133308854303
	// Words     []Word  `json:"words"`
}

type Word struct {
	// The ending second when the word completes
	End float64 `json:"end"`
	// The second this word begins in the recording
	Start float64 `json:"start"`
	Word  string  `json:"word"`
}

type WorkersWhisperClient struct {
	account string
	token   string
	model   string

	http *http.Client
}

type WorkersWhisperClientOptions struct {
	Account   string `env:"CF_ACCOUNT_ID,required"`
	Token     string `env:"CF_TOKEN,required"`
	ModelName string `env:"CF_MODEL_NAME,required"`
}

func NewWorkersWhisperClient(options WorkersWhisperClientOptions) *WorkersWhisperClient {
	return &WorkersWhisperClient{
		account: options.Account,
		token:   options.Token,
		model:   options.ModelName,
		http:    http.DefaultClient,
	}
}

func (w *WorkersWhisperClient) runCF(ctx context.Context, data []byte) (*CloudflareResponse[SpeechRecognitionResponse], error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/run/%s", w.account, w.model), bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.token)

	resp, err := w.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok http response: [%d] %s", resp.StatusCode, resp.Status)
	}

	var cfResp *CloudflareResponse[SpeechRecognitionResponse]
	err = json.NewDecoder(resp.Body).Decode(&cfResp)
	if err != nil {
		return nil, fmt.Errorf("decoding response json: %w", err)
	}

	return cfResp, nil
}

func (w *WorkersWhisperClient) Run(ctx context.Context, data []byte) (*asr.ASROutput, error) {
	resp, err := w.runCF(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("request unsuccessful")
	}
	if resp.Result == nil {
		return nil, fmt.Errorf("nil result")
	}

	return &asr.ASROutput{
		ModelName: apiPrefix + w.model,
		Text:      resp.Result.Text,
	}, nil
}
