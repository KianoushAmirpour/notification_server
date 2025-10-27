package ai

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KianoushAmirpour/notification_server/internal/config"
	"google.golang.org/genai"
)

type GemeniClient struct {
	Client *genai.Client
	Cfg    *config.Config
	Logger *slog.Logger
}

func NewGemeniClient(ctx context.Context, cfg *config.Config, logger *slog.Logger) *GemeniClient {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: cfg.GemeniAPI})
	if err != nil {
		logger.Error("failed to create gemeni client", slog.String("reason", err.Error()))
	}
	return &GemeniClient{Client: client, Cfg: cfg, Logger: logger}
}

func (g GemeniClient) GenerateStory(ctx context.Context, preferences string) (string, error) {

	result, err := g.Client.Models.GenerateContent(
		ctx,
		g.Cfg.GemeniModel,
		genai.Text(fmt.Sprintf("Create a story about %s with fancey theme", preferences)),
		nil)

	if err != nil {
		g.Logger.Error("failed to generate story", slog.String("reason", err.Error()))
		return "", err
	}

	g.Logger.Info("story was generated")
	return result.Text(), nil

}
