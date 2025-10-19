package ai

import (
	"context"
	"fmt"
	"log"

	"github.com/KianoushAmirpour/notification_server/internal/config"
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

func (g GemeniClient) GenerateStory(ctx context.Context, preferences string) (string, error) {

	result, err := g.Client.Models.GenerateContent(
		ctx,
		g.Cfg.GemeniModel,
		genai.Text(fmt.Sprintf("Create a story about %s with fancey theme", preferences)),
		nil)

	if err != nil {
		return "", err
	}

	return result.Text(), nil

}
