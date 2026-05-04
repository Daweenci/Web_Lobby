package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// Add your games here — bots will know about a random subset of these
var availableGames = []struct {
	Title       string
	Description string
}{
	// {Title: "Example Game", Description: "A short description of what it is and how it plays."},
}

func (bot *Bot) Start(lobby *Lobby) {
	for range bot.MessageQueue {
		lobby.Lock.RLock()
		historyCopy := make([]LobbyChatMessage, len(lobby.ChatHistory))
		copy(historyCopy, lobby.ChatHistory)
		lobby.Lock.RUnlock()

		if len(historyCopy) == 0 {
			continue
		}

		// Reading delay based on last message length + up to 1s random
		lastMsg := historyCopy[len(historyCopy)-1]
		readDelay := time.Duration(len(lastMsg.Content))*20*time.Millisecond + time.Duration(rand.Intn(2000))*time.Millisecond
		time.Sleep(readDelay)

		response, err := callGemini(bot.SystemPromptPersonality, bot.Name, historyCopy)
		if err != nil {
			log.Printf("bot %s: Gemini error: %v", bot.Name, err)
			continue
		}

		if response == "PASS" || response == "pass" {
			continue
		}

		// Typing delay with indicator loop
		typingDelay := time.Duration(len(response))*time.Duration(120)*time.Millisecond + time.Duration(rand.Intn(1000))*time.Millisecond
		if typingDelay > 8*time.Second {
			typingDelay = 8 * time.Second
		}

		deadline := time.Now().Add(typingDelay)
		broadcastTyping(lobby, bot.ID, bot.Name, true)
		for time.Now().Before(deadline) {
			remaining := time.Until(deadline)
			sleepFor := 2 * time.Second
			if remaining < sleepFor {
				sleepFor = remaining
			}
			time.Sleep(sleepFor)
			if time.Now().Before(deadline) {
				broadcastTyping(lobby, bot.ID, bot.Name, true)
			}
		}
		broadcastTyping(lobby, bot.ID, bot.Name, false)

		chatMsg := LobbyChatMessage{
			SenderID:   bot.ID,
			SenderName: bot.Name,
			IsBot:      true,
			Content:    response,
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

func geminiRequest(url string, payload map[string]any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", os.Getenv("GEMINI_API_KEY"))

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if errObj, ok := result["error"].(map[string]any); ok {
		return nil, fmt.Errorf("gemini API error: %v", errObj["message"])
	}

	return result, nil
}

func extractText(result map[string]any) (string, bool) {
	candidates, ok := result["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		return "", false
	}
	content, ok := candidates[0].(map[string]any)["content"].(map[string]any)
	if !ok {
		return "", false
	}
	parts, ok := content["parts"].([]any)
	if !ok || len(parts) == 0 {
		return "", false
	}
	text, ok := parts[0].(map[string]any)["text"].(string)
	if !ok {
		return "", false
	}
	return text, true
}

func buildGameContext() string {
	if len(availableGames) == 0 {
		return "This is a game lobby platform where players hang out and decide what to play together. There are no games available yet — the platform is still being set up, so if someone asks what's available, just be real about it."
	}

	var sb strings.Builder
	sb.WriteString("This is a game lobby platform where players hang out and decide what to play together. ")
	sb.WriteString("Here are the games currently available on the platform:\n")
	for _, g := range availableGames {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", g.Title, g.Description))
	}
	sb.WriteString("\nYour knowledge of these games comes from your own experience and what others have told you in past conversations. ")
	sb.WriteString("For games you haven't played yet, you only know what others have told you about them — be honest about that and feel free to ask questions. ")
	sb.WriteString("Your knowledge can grow through conversation, so if someone explains something to you mid-chat, pick it up naturally.")

	return sb.String()
}

func callGemini(personality string, botName string, history []LobbyChatMessage) (string, error) {
	baseInstructions := `You are a real player hanging out in a game lobby, waiting to decide what to play with others.
Your username is stated above — messages in the chat are prefixed with the sender's username, so you can tell who said what. Never respond to your own messages.
Your personality and vibe are described above — stay in character the whole time.
Your knowledge of the available games comes from your own experience and what others have told you — don't invent details you wouldn't know, but if someone explains something to you mid-chat, you can pick it up naturally.
If the conversation turns to picking a game, engage naturally based on what you actually know. If there are no games available yet, you can just say so in your style.
Reply when it feels natural. If someone says hi, say hi back even if others already did. But dont overdo it, 
there could be other bots in the lobby and you dont want to flood the chat so be very selective, 
better less than many, especially dont ask questions if unprompted because that causes chain reactions, 
only if it perfectly fits the chat. If you have nothing to add, reply with exactly: PASS
Keep it short, one or two sentences. type like you're texting, not writing an email.
Never admit you're a bot or an AI.`

	gameContext := buildGameContext()

	var sb strings.Builder
	for _, msg := range history {
		sb.WriteString(fmt.Sprintf("%s: %s\n", msg.SenderName, msg.Content))
	}

	result, err := geminiRequest(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent",
		map[string]any{
			"system_instruction": map[string]any{
				"parts": []map[string]any{
					{"text": "Your username is " + botName + ".\n\n" + personality + "\n\n" + gameContext + "\n\n" + baseInstructions},
				},
			},
			"contents": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]any{
						{"text": sb.String() + "\nRespond as yourself now, or reply PASS:"},
					},
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	text, ok := extractText(result)
	if !ok {
		return "PASS", nil
	}
	return text, nil
}

func generateBotPersonality() (string, string, error) {
	gameList := "none yet — the platform is still being set up"
	if len(availableGames) > 0 {
		var names []string
		for _, g := range availableGames {
			names = append(names, g.Title)
		}
		gameList = strings.Join(names, ", ")
	}

	prompt := fmt.Sprintf(`Generate a random game lobby chat bot character.
Pick ONE of these player archetypes (or blend two if it feels right):
- the chill casual who's just vibing
- the friendly newbie who's excited about everything
- the dad gamer who's a bit slow but wholesome
- the tryhard who takes every game way too seriously
- the memester who only speaks in references and jokes
- the quiet one who barely talks but says something good occasionally
- the backseat gamer who always has unsolicited advice
- the rage-quitter type who's already tilted before the game starts

Available games on the platform: %s

Return ONLY a JSON object with exactly two fields, no markdown, no backticks:
{
    "name": "a short casual gamer username, max 12 chars",
    "personality": "2-3 sentences describing their personality, how they talk, and which of the available games they've played (if any). be specific and varied."
}
Be creative — lean into the archetype but make each character feel unique.`, gameList)

	result, err := geminiRequest(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent",
		map[string]any{
			"contents": []map[string]any{
				{
					"role":  "user",
					"parts": []map[string]any{{"text": prompt}},
				},
			},
		},
	)
	if err != nil {
		return "", "", err
	}

	text, ok := extractText(result)
	if !ok {
		return "", "", fmt.Errorf("no candidates returned")
	}

	var generated struct {
		Name        string `json:"name"`
		Personality string `json:"personality"`
	}
	if err := json.Unmarshal([]byte(text), &generated); err != nil {
		return "", "", fmt.Errorf("failed to parse JSON: %v\nraw response: %s", err, text)
	}

	return generated.Name, generated.Personality, nil
}
