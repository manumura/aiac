package ollama

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofireflyio/aiac/v5/libaiac/types"
)

// Conversation is a struct used to converse with an Ollama chat model. It
// maintains all messages sent/received in order to maintain context.
type Conversation struct {
	// Messages is the list of all messages exchanged between the user and the
	// assistant.
	Messages []types.Message

	backend *Ollama
	model   string
}

type chatResponse struct {
	Message types.Message `json:"message"`
	Done    bool          `json:"done"`
}

// Chat initiates a conversation with an Ollama chat model. A conversation
// maintains context, allowing to send further instructions to modify the output
// from previous requests. The name of the model to use must be provided. Users
// can also supply zero or more "previous messages" that may have been exchanged
// in the past. This practically allows "loading" previous conversations and
// continuing them.
func (backend *Ollama) Chat(model string, msgs ...types.Message) types.Conversation {
	chat := &Conversation{
		backend: backend,
		model:   model,
	}

	if len(msgs) > 0 {
		chat.Messages = msgs
	}

	return chat
}

// Send sends the provided message to the API and returns a Response object.
// To maintain context, all previous messages (whether from you to the API or
// vice-versa) are sent as well, allowing you to ask the API to modify the
// code it already generated.
func (conv *Conversation) Send(ctx context.Context, prompt string) (
	res types.Response,
	err error,
) {
	var answer chatResponse

	conv.Messages = append(conv.Messages, types.Message{
		Role:    "user",
		Content: prompt,
	})

	err = conv.backend.NewRequest("POST", "/chat").
		JSONBody(map[string]interface{}{
			"model":    conv.model,
			"messages": conv.Messages,
			"options": map[string]interface{}{
				"temperature": 0.2,
			},
			"stream": false,
		}).
		Into(&answer).
		RunContext(ctx)
	if err != nil {
		return res, fmt.Errorf("failed sending prompt: %w", err)
	}

	conv.Messages = append(conv.Messages, answer.Message)

	res.FullOutput = strings.TrimSpace(answer.Message.Content)
	if answer.Done {
		res.StopReason = "done"
	} else {
		res.StopReason = "truncated"
	}

	var ok bool
	if res.Code, ok = types.ExtractCode(res.FullOutput); !ok {
		res.Code = res.FullOutput
	}

	return res, nil
}
