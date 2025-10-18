package ai

import (
	"context"
	"fmt"
	"log"

	"github.com/KianoushAmirpour/notification_server/internal/config"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"google.golang.org/genai"
)

type GemeniClient struct {
	Client *genai.Client
	Cfg    *config.Config
}

func NewGemeniClient(ctx context.Context, cfg *config.Config) *GemeniClient {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: cfg.GemeniAPI})
	if err != nil {
		log.Fatal(err)
	}
	return &GemeniClient{Client: client, Cfg: cfg}
}

func (g GemeniClient) GenerateStory(ctx context.Context, preferences []string) (*genai.GenerateContentResponse, error) {
	// resp, err := g.Client.Models.GenerateImages(
	// 	ctx,
	// 	g.Cfg.GemeniModel,
	// 	fmt.Sprintf("Create a picture about %s with %s", preferences, domain.ImageGenerationPrompt),
	// 	&genai.GenerateImagesConfig{NumberOfImages: int32(numofimages), AspectRatio: ratio})

	result, err := g.Client.Models.GenerateContent(
		ctx,
		g.Cfg.GemeniModel,
		genai.Text(fmt.Sprintf("Create a story about %s with %s", preferences, domain.StoryGenerationThemePrompt)),
		nil)

	if err != nil {
		return nil, err
	}

	return result, nil

}
