package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func (bot *Bot) Start(lobby *Lobby) {
	for range bot.MessageQueue {
		delay := time.Duration(1000+rand.Intn(5000)) * time.Millisecond
		time.Sleep(delay)

		lobby.Lock.RLock()
		historyCopy := make([]LobbyChatMessage, len(lobby.ChatHistory))
		copy(historyCopy, lobby.ChatHistory)
		lobby.Lock.RUnlock()

		response, err := callGemini(bot.SystemPromptPersonality, historyCopy)
		if err != nil {
			log.Printf("bot %s: Gemini error: %v", bot.Name, err)
			continue
		}

		if response == "PASS" {
			continue
		}

		chatMsg := LobbyChatMessage{
			SenderID:   bot.ID,
			SenderName: bot.Name,
			IsBot:      true,
			Message:    response,
		}

		lobby.Lock.Lock()
		lobby.ChatHistory = append(lobby.ChatHistory, chatMsg)
		lobby.Lock.Unlock()

		broadcastLobbyChatMessage(lobby, chatMsg)

		lobby.Lock.RLock()
		for _, otherBot := range lobby.Bots {
			if otherBot.ID != bot.ID {
				otherBot.MessageQueue <- BotMessage{LobbyID: lobby.ID}
			}
		}
		lobby.Lock.RUnlock()
	}
}

func callGemini(personality string, history []LobbyChatMessage) (string, error) {
	baseInstructions := `You are in a game lobby chat.
You do NOT have to respond to every message — only if you genuinely have something to add.
If you have nothing to say, reply with exactly: PASS
Keep responses short, one or two sentences max.
Type in lowercase mostly, like a real person texting.
Never break character. You are a real player, not an AI.`

	fullPrompt := personality + "\n\n" + baseInstructions

	contents := make([]map[string]any, len(history))
	for i, msg := range history {
		contents[i] = map[string]any{
			"role": "user",
			"parts": []map[string]any{
				{"text": fmt.Sprintf("%s: %s", msg.SenderName, msg.Message)},
			},
		}
	}

	body, err := json.Marshal(map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]any{
				{"text": fullPrompt},
			},
		},
		"contents": contents,
	})
	if err != nil {
		return "", err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + os.Getenv("GEMINI_API_KEY")
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	candidates, ok := result["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		return "PASS", nil
	}

	content, ok := candidates[0].(map[string]any)["content"].(map[string]any)
	if !ok {
		return "PASS", nil
	}

	parts, ok := content["parts"].([]any)
	if !ok || len(parts) == 0 {
		return "PASS", nil
	}

	text, ok := parts[0].(map[string]any)["text"].(string)
	if !ok {
		return "PASS", nil
	}

	return text, nil
}

func generateBotPersonality() (string, string, error) {
	prompt := `Generate a random game lobby chat bot character.
Return ONLY a JSON object with exactly two fields, no markdown, no backticks:
{
    "name": "a short casual gamer username, max 12 chars",
    "personality": "2-3 sentences describing their personality, playstyle, mood today, and how they talk. be specific and varied."
}
Be creative and varied — different moods, archetypes, speaking styles each time.`

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": prompt},
				},
			},
		},
	})
	if err != nil {
		return "", "", err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + os.Getenv("GEMINI_API_KEY")
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	candidates, ok := result["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		return "", "", fmt.Errorf("no candidates returned")
	}

	text, ok := candidates[0].(map[string]any)["content"].(map[string]any)["parts"].([]any)[0].(map[string]any)["text"].(string)
	if !ok {
		return "", "", fmt.Errorf("failed to parse response")
	}

	var generated struct {
		Name        string `json:"name"`
		Personality string `json:"personality"`
	}
	if err := json.Unmarshal([]byte(text), &generated); err != nil {
		return "", "", fmt.Errorf("failed to parse JSON: %v", err)
	}

	return generated.Name, generated.Personality, nil
}
