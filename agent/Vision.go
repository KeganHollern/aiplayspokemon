package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"sync"

	"github.com/sashabaranov/go-openai"
)

const (
	vision_prompt = `You are helping the user play the game Pokemon Yellow.
You will be given an image from the game.
Your job is to describe the image to the user.
Do not suggest actions for the user.
Describe details like doors, sprites, and directions where the map continues off screen.
Describe if the player is up against a wall or the edge of the map.
When in a room, the space outside of the room will appear dark gray or black.
Be sure to describe when the exit of the room is clearly visible, sometimes this will be shown as a carpet along the bottom wall.
Use directions like UP DOWN LEFT and RIGHT to describe the direction of things from the player.
Also specify if the player is facing anything important.
Estimate distance of objects from the player.`
	vision_model                  = openai.GPT4oLatest
	vision_max_description_tokens = 1000
)

type Vision struct {
	// frame holds the last rendered from from the emulator
	// use getImage() to safely acquire a copy of the frame
	frame *image.RGBA
	lock  sync.Mutex

	client  *openai.Client
	history []openai.ChatCompletionMessage
}

func NewVisionAgent() *Vision {
	return &Vision{
		client:  openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		history: []openai.ChatCompletionMessage{},
	}
}

func (ai *Vision) SetImage(img *image.RGBA) {
	// NOTE: we don't want to block the render thread, so we try to lock the mutex.
	if !ai.lock.TryLock() {
		return
	}
	defer ai.lock.Unlock()

	ai.frame = img
}

func (ai *Vision) getImage() *image.RGBA {
	// NOTE: we DO want to block the ai thread
	ai.lock.Lock()
	defer ai.lock.Unlock()

	return ai.frame
}

func img2jpgb64(img *image.RGBA) (string, error) {
	if img == nil {
		return "", errors.New("no image")
	}

	var buf bytes.Buffer

	// create a new base64 encoder that writes to our buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	defer encoder.Close()

	// encode our image as jpeg into our base64 encoder losslessly
	if err := jpeg.Encode(encoder, img, &jpeg.Options{Quality: 100}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (ai *Vision) DescribeScene() (string, error) {
	img := ai.getImage()
	if img == nil {
		return "", errors.New("no scene")
	}

	// IMG to Base64 for prompt
	b64, err := img2jpgb64(img)
	if err != nil {
		return "", fmt.Errorf("failed to b64 encode scene; %w", err)
	}
	b64url := fmt.Sprintf("data:image/jpeg;base64,%s", b64)

	// if in iterm2 we can render the image we captured for debugging
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		fmt.Printf("Scene:\n")
		fmt.Printf("\033]1337;File=name=image.jpg;inline=1:%s\a\n", b64)
	} else {
		fmt.Println("Running outside iTerm2; image not displayed.")
	}

	// restrict history
	messages := ai.history
	if len(messages) > (historical_frames * 2) {
		messages = messages[2:]
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser,
		MultiContent: []openai.ChatMessagePart{
			{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL:    b64url,
					Detail: openai.ImageURLDetailLow,
				},
			},
		},
	})
	fmt.Printf("[DBG]: vision %d messages\n", len(messages))
	res, err := ai.client.CreateChatCompletion(context.TODO(), openai.ChatCompletionRequest{
		Model: vision_model,
		Messages: append(
			[]openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: vision_prompt,
			},
			}, messages...),
		MaxCompletionTokens: vision_max_description_tokens,
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe scene; %w", err)
	}

	if len(res.Choices) == 0 {
		return "", errors.New("no choices")
	}

	// update historical messages
	ai.history = append(messages, res.Choices[0].Message)

	return res.Choices[0].Message.Content, nil
}
