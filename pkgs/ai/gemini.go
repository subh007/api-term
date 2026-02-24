package ai

import (
	"context"

	"google.golang.org/genai"
)

// GeminiClient wraps the GenAI client
type GeminiClient struct {
	client *genai.Client
}

// NewGeminiClient creates a new Gemini Client
func NewGeminiClient(ctx context.Context) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &GeminiClient{client: client}, nil
}

// CreateChatSession initializes a chat session
func (g *GeminiClient) CreateChatSession(ctx context.Context, model string) (*genai.Chat, error) {
	chat, err := g.client.Chats.Create(ctx, model, nil, nil)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

// FormatContent extracts the text from a Candidate content slice.
func FormatContent(resp *genai.GenerateContentResponse) string {
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		var out string
		for _, part := range resp.Candidates[0].Content.Parts {
			out += part.Text
		}
		return out
	}
	return "No insights returned or unable to parse response."
}
