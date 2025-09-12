// internal/domain/game/character.go (updated)
package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tahcohcat/gofigure-web/internal/llm"
	"github.com/tahcohcat/gofigure-web/internal/logger"
)

type Message struct {
	Role      string    `json:"role,omitempty"`
	Content   string    `json:"content,omitempty" json:"content,omitempty"`
	Emotions  string    `json:"emotions,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type TTS struct {
	Engine string `json:"engine,omitempty"`
	Model  string `json:"model,omitempty"`
}

// Character in the game
type Character struct {
	Name        string   `json:"name"`
	Personality string   `json:"personality"`
	Sprite      string   `json:"sprite,omitempty"`
	Knowledge   []string `json:"knowledge"`
	Reliable    bool     `json:"reliable"`
	TTS         []TTS    `json:"tts"`

	Conversation []*Message
}

func (c *Character) GetCharacterResponse(ctx context.Context, prompt string, llmClient llm.LLM) (*llm.CharacterReply, error) {

	resp, err := llmClient.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var reply llm.CharacterReply
	if err := json.Unmarshal([]byte(resp), &reply); err != nil {
		logger.New().Warn(fmt.Sprintf("failed to unmarshal response. [response:%s, prompt:%s]", resp, prompt))

		// Try to extract JSON from the response if it's embedded in text
		if extractedReply, extractErr := c.extractJSONFromResponse(resp); extractErr == nil {
			return extractedReply, nil
		}

		// Fallback: create a valid reply from the raw response
		return &llm.CharacterReply{
			Response: resp,
			Emotion:  "neutral", // Default emotion
		}, nil
	}
	return &reply, nil
}

// AskQuestion using Ollama client for character interaction
func (c *Character) AskQuestion(ctx context.Context, question string, murder Murder, llmClient llm.LLM) (*llm.CharacterReply, error) {

	c.addQuestion(question, murder)

	prompt := c.serialiseConversation()

	resp, err := c.GetCharacterResponse(ctx, prompt, llmClient)
	if err != nil {
		logger.New().WithError(err).Warn("could not generate character response")
		return &llm.CharacterReply{}, err
	}

	c.Conversation = append(c.Conversation, &Message{

		// openai supperted types:  ['system', 'assistant', 'user', 'function', 'tool', and 'developer']",
		Role:      "assistant",
		Content:   resp.Response,
		Emotions:  resp.Emotion,
		Timestamp: time.Now(),
	})

	return resp, nil
}

func (c *Character) addQuestion(question string, murder Murder) {
	reliabilityNote := "You are generally truthful and helpful."
	if !c.Reliable {
		reliabilityNote = "You might hide some facts, be evasive, or provide misleading information. Stay in character."
	}

	latest := fmt.Sprintf("Detective's follow up question: %s\n\nIMPORTANT: You MUST respond in this exact JSON format: {\"response\": \"your character response here\", \"emotion\": \"your emotional state\"}", question)

	if c.IsInitialMessage() {
		scenario := fmt.Sprintf(`You are roleplaying as %s in a murder mystery game.

CHARACTER PROFILE:
- Name: %s
- Personality: %s
- %s

MURDER SCENARIO:
- Victim found in: %s
- Murder weapon: %s  
- Actual killer: %s
- Your knowledge about the case: %v

CRITICAL INSTRUCTIONS:
- Stay completely in character
- Answer the detective's question based on your personality and knowledge
- Keep responses concise but engaging
- Don't break character or mention this is a game
- If you don't know something, say so in character
- You MUST respond in valid JSON format only
- Reply in this EXACT JSON structure: {"response": "your character response here", "emotion": "your emotional state"}
- Do NOT include any text before or after the JSON
- Valid emotions: happy, sad, angry, nervous, confident, suspicious, worried, neutral, etc.

Detective's question: "%s"

Your JSON response as %s:`,
			c.Name, c.Name, c.Personality, reliabilityNote,
			murder.Location, murder.Weapon, murder.Killer, c.Knowledge,
			question, c.Name)

		c.Conversation = []*Message{
			{Role: "system", Content: fmt.Sprintf("%s", scenario), Timestamp: time.Now()},
		}

		latest = fmt.Sprintf("Detective's question: %s", question)
	}

	c.Conversation = append(c.Conversation, &Message{Role: "user", Content: latest, Timestamp: time.Now()})
}

func (c *Character) IsInitialMessage() bool {
	return len(c.Conversation) == 0
}

func (c *Character) serialiseConversation() string {
	s, err := json.Marshal(c.Conversation)
	if err != nil {
		logger.New().Error(err.Error())
		return ""
	}

	return string(s)
}

// extractJSONFromResponse tries to find and extract JSON from a text response
func (c *Character) extractJSONFromResponse(response string) (*llm.CharacterReply, error) {
	// Look for JSON patterns in the response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start != -1 && end != -1 && end > start {
		jsonStr := response[start : end+1]
		var reply llm.CharacterReply
		if err := json.Unmarshal([]byte(jsonStr), &reply); err == nil {
			return &reply, nil
		}
	}

	return nil, fmt.Errorf("no valid JSON found in response")
}
