package messages

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/google/go-jsonnet"
)

//go:embed jsonnet/*
var messages embed.FS

type MessageProvider struct {
	vm *jsonnet.VM
}

func NewMessageProvider() (*MessageProvider, error) {
	m := &MessageProvider{
		vm: jsonnet.MakeVM(),
	}

	imports := make(map[string]jsonnet.Contents)
	fs.WalkDir(messages, ".", func(path string, d fs.DirEntry, err error) error {
		if d != nil && !d.IsDir() {
			content, _ := messages.ReadFile(path)
			imports[strings.TrimPrefix(path, "jsonnet/")] = jsonnet.MakeContentsRaw(content)
		}
		return nil
	})

	m.vm.Importer(&jsonnet.MemoryImporter{
		Data: imports,
	})

	_, _, err := m.vm.ImportData("anonymous", "index.jsonnet")
	if err != nil {
		return nil, fmt.Errorf("importing index: %w", err)
	}

	return m, nil
}

func (m *MessageProvider) ExecuteMessage(messageName string, data any) (string, error) {
	m.vm.TLAVar("message_key", messageName)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshaling data: %w", err)
	}
	m.vm.TLACode("data", string(jsonData))

	defer m.vm.TLAReset()

	jsonOut, err := m.vm.EvaluateAnonymousSnippet("anonymous", "function(message_key, data) (import 'index.jsonnet')[message_key](data)")
	if err != nil {
		return "", fmt.Errorf("evaluating jsonnet: %w", err)
	}

	return jsonOut, nil
}
