package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

const (
	planner_prompt = `You are helping the user play the game Pokemon Yellow.
Your job is to guide the journey. 
The user will describe to you the scene.
The user will also provide the last input they gave the game. 
Start your reply by describing what has likely just taken place.
Finish your reply by 1-2 sentences of what the user should do next.
Use the users last input to correct their actions or change your plan.
Be very explicit and concise.`
	planner_model                 = openai.GPT4oLatest
	planner_max_completion_tokens = 200
)

type Planner struct {
	client  *openai.Client
	history []openai.ChatCompletionMessage
}

func NewPlannerAgent() *Planner {
	return &Planner{
		client:  openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		history: []openai.ChatCompletionMessage{},
	}
}

func (p *Planner) Plan(scene string, last_input string) (string, error) {

	messages := p.history
	// if history too long drop 2nd oldest message and response
	if len(messages) > (historical_frames * 2) {
		messages = messages[2:]
	}

	// append scene to history
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: scene + "\nLast input: " + last_input,
	})

	// get response from openai
	fmt.Printf("[DBG]: planner %d messages\n", len(messages))
	resp, err := p.client.CreateChatCompletion(
		context.TODO(),
		openai.ChatCompletionRequest{
			Model: planner_model,
			Messages: append([]openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: planner_prompt,
				},
			}, messages...),
			MaxCompletionTokens: planner_max_completion_tokens,
		},
	)
	if err != nil {
		return "", err
	}

	// append response to history
	p.history = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: resp.Choices[0].Message.Content,
	})

	// return response
	return resp.Choices[0].Message.Content, nil
}