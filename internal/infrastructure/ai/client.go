package ai

import (
	"context"
	"fmt"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"google.golang.org/genai"
)

// var StoryGenerationPrompt = `
// You are a story-generation assistant. Create a story based on the user's preferences by following below steps.

// Instructions:
// 1. Generate a roght plot:
// Using user's input, produce a compact outline that includes: Main characters, Setting, Central conflict, Key turning points, Ending direction (not fully detailed)
// this plot should be up to 3 bullet points. Each one is limited to 50 words.

// 2. Expand into Scenes
// Turn the plot into a scene-by-scene breakdown. For each scene, include: Purpose of the scene, What characters do or feel, Sensory or atmospheric notes and Important narrative developments.
// Include up to 5 scenes depending on story complexity. Each scene is limited to 200 words.

// 3. Output the final story. You are allowed to use maximum 1000 words per story.

// user preferences: %s
// `
var StoryGenerationPrompt = `
You are a story-generation assistant. Create a story based on the user's preferences..
You are allowed to use maximum 1000 words per story.

user preferences: %s
`

type GemeniClient struct {
	Client *genai.Client
	Model  string
}

func NewGemeniClient(ctx context.Context, apikey, model string) (*GemeniClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apikey})
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to create gemeni client", err)
	}
	return &GemeniClient{Client: client, Model: model}, nil
}

func (g GemeniClient) GenerateStory(ctx context.Context, preferences string) (string, error) {
	prompt := fmt.Sprintf(StoryGenerationPrompt, preferences)
	result, err := g.Client.Models.GenerateContent(
		ctx,
		g.Model,
		genai.Text(prompt),
		nil)

	if err != nil {
		return "", domain.NewDomainError(domain.ErrCodeExternal, "failed to generate story from ai model", err)
	}

	return result.Text(), nil

}
