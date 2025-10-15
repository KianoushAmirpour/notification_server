package adapters

import (
	"context"

	"github.com/KianoushAmirpour/notification_server/internal/config"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"google.golang.org/genai"
)

type GemeniClient struct {
	Client *genai.Client
	Cfg    *config.Config
}

func NewGemeniClient(client *genai.Client, cfg *config.Config) *GemeniClient {
	return &GemeniClient{Client: client, Cfg: cfg}
}

func (g GemeniClient) GenerateImages(ctx context.Context, cfg *config.Config) (*genai.GenerateImagesResponse, error) {
	resp, err := g.Client.Models.GenerateImages(
		ctx, cfg.GemeniModel,
		domain.ImageGenerationPrompt,
		&genai.GenerateImagesConfig{NumberOfImages: int32(g.Cfg.NumOfImages), AspectRatio: g.Cfg.AspectRatio})

	if err != nil {
		return nil, err
	}

	return resp, nil

}
