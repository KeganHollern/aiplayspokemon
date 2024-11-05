package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const (
	actor_prompt = `You are playing Pokemon Yellow.
You will be given a description of the scene in the game.
You will also be given a goal to achieve.
Your job is to choose what sequence of buttons to press next to progress us towards the goal.`
	actor_model = openai.GPT4oMini
)

type Actor struct {
	client  *openai.Client
	history []openai.ChatCompletionMessage
}

func NewActorAgent() *Actor {
	return &Actor{
		client:  openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		history: []openai.ChatCompletionMessage{},
	}
}

type actions struct {
	Input []string `json:"input"`
}

func (p *Actor) Act(scene string, goal string) ([]string, error) {
	messages := p.history

	// if history too long drop 2nd oldest message and response
	if len(messages) > (historical_frames * 2) {
		messages = messages[2:]
	}

	// append scene to history
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: scene + "\n" + goal,
	})

	// get response from openai
	fmt.Printf("[DBG]: actor %d messages\n", len(messages))
	resp, err := p.client.CreateChatCompletion(
		context.TODO(),
		openai.ChatCompletionRequest{
			Model: actor_model,
			Messages: append([]openai.ChatCompletionMessage{
				// push initial system prompt to history
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: actor_prompt,
				},
			}, messages...),

			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
				JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
					Name: "inputs",
					Schema: &jsonschema.Definition{
						Type: jsonschema.Object,
						Properties: map[string]jsonschema.Definition{
							"input": {
								Type:        jsonschema.Array,
								Description: "sequence of buttons to press in order",
								Items: &jsonschema.Definition{
									Type:        jsonschema.String,
									Description: "the button to press",
									Enum:        []string{"UP", "DOWN", "LEFT", "RIGHT", "A", "B", "START", "SELECT", "NONE"},
								},
							},
						},
						Required:             []string{"input"},
						AdditionalProperties: false,
					},
					Strict: true,
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	// append response to history
	p.history = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: resp.Choices[0].Message.Content,
	})

	// return response
	var acts actions
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &acts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ai response; %w", err)
	}

	buttonsToPress := []string{}
	for _, act := range acts.Input {
		if act != "NONE" {
			buttonsToPress = append(buttonsToPress, act)
		}
	}
	return buttonsToPress, nil
}
