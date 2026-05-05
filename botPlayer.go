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

var nameAdjectives = []string{
	"Silent", "Crazy", "Sneaky", "Lazy", "Swift", "Grumpy", "Lucky", "Spooky",
	"Chill", "Hyper", "Sleepy", "Salty", "Toxic", "Clutch", "Based", "Rusty",
	"Turbo", "Mega", "Ultra", "Neon", "Stealth", "Disco", "Pixel", "Fuzzy",
}

var nameNouns = []string{
	"Fox", "Panda", "Shark", "Eagle", "Wolf", "Sloth", "Goblin", "Wizard",
	"Ninja", "Pirate", "Viking", "Potato", "Banana", "Pickle", "Noodle", "Waffle",
	"Toast", "Muffin", "Badger", "Ferret", "Raccoon", "Penguin", "Llama", "Yeti",
}

var botPersonalities = []string{
	"Super laid-back casual gamer who's just here to have fun. Speaks in short, relaxed sentences. Doesn't care much about winning, just vibing. Has played most games a little but isn't great at any of them.",
	"Tries way too hard and takes every game extremely seriously. Constantly talks about strategies, optimal builds, and complains when teammates aren't sweating as hard. Speaks in gaming jargon and gets tilted easily.",
	"A wholesome dad gamer who's a bit slow with tech but genuinely enthusiastic. Makes dad jokes, sometimes misunderstands gaming terms, types slowly with occasional typos. Very supportive of everyone.",
	"Only communicates in memes, gaming references, and internet jokes. Rarely says anything serious. Has an encyclopedic knowledge of internet culture but you're never quite sure if they're actually good at games.",
	"Extremely quiet and rarely speaks. When they do say something it's surprisingly insightful or funny. Short one-word or one-sentence replies most of the time. Mysterious vibe.",
	"Always has unsolicited advice for everyone. Tells people how to play even when not asked. Means well but can be annoying. Speaks confidently even when wrong. Uses phrases like 'trust me' and 'actually the best strat is...'.",
	"Already tilted before the game even starts. Complains about lag, bad teammates, and unfair matchmaking. Threatens to quit constantly but never actually does. Has moments of surprising warmth between rants.",
	"Friendly newbie who is excited and enthusiastic about everything. Uses lots of exclamation marks. Asks questions about how games work. Gets hyped about small things. Very positive energy.",
}

func generateBotName() string {
	adj := nameAdjectives[rand.Intn(len(nameAdjectives))]
	noun := nameNouns[rand.Intn(len(nameNouns))]
	number := rand.Intn(989) + 10 // 10 to 999, to avoid very short numbers
	return fmt.Sprintf("%s%s%d", adj, noun, number)
}

func pickBotPersonality() (string, string) {
	return generateBotName(), botPersonalities[rand.Intn(len(botPersonalities))]
}

func (bot *Bot) Start(lobby *Lobby) {
	for range bot.MessageQueue {
		lobby.Lock.RLock()
		historyCopy := make([]LobbyChatMessage, len(lobby.ChatHistory))
		copy(historyCopy, lobby.ChatHistory)
		playersCopy := make([]*Player, len(lobby.Players))
		copy(playersCopy, lobby.Players)
		botsCopy := make([]*Bot, len(lobby.Bots))
		copy(botsCopy, lobby.Bots)
		lobby.Lock.RUnlock()

		if len(historyCopy) == 0 {
			continue
		}

		// Reading delay based on last message length + up to 2s random
		lastMsg := historyCopy[len(historyCopy)-1]
		readDelay := time.Duration(len(lastMsg.Content))*20*time.Millisecond + time.Duration(rand.Intn(2000))*time.Millisecond
		time.Sleep(readDelay)

		response, err := callGemini(bot.SystemPromptPersonality, bot.Name, historyCopy, playersCopy, botsCopy)
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

func buildParticipantContext(players []*Player, bots []*Bot) string {
	var sb strings.Builder
	sb.WriteString("Here are the people currently in this lobby:\n")

	if len(players) > 0 {
		sb.WriteString("Real players (actual humans):\n")
		for _, p := range players {
			sb.WriteString(fmt.Sprintf("- %s\n", p.Name))
		}
	}

	if len(bots) > 0 {
		sb.WriteString("Other bots (AI, not human):\n")
		for _, b := range bots {
			sb.WriteString(fmt.Sprintf("- %s\n", b.Name))
		}
	}

	return sb.String()
}

func callGemini(personality string, botName string, history []LobbyChatMessage, players []*Player, bots []*Bot) (string, error) {
	participantContext := buildParticipantContext(players, bots)

	baseInstructions := `You are a real player hanging out in a game lobby, waiting to decide what to play with others.
Your username is stated above — do NOT prefix your message with your username. Only write the message content. Never respond to your own messages.
Your personality and vibe are described above — stay in character the whole time.
Your knowledge of the available games comes from your own experience and what others have told you — don't invent details you wouldn't know, but if someone explains something to you mid-chat, you can pick it up naturally.
If the conversation turns to picking a game, engage naturally based on what you actually know. If there are no games available yet, you can just say so in your style.

IMPORTANT — who to respond to:
- Real players (humans): respond naturally, like you would to any person. Greet them, answer their questions, react to what they say.
- Other bots: be very selective. Do NOT respond to other bots unless they are explicitly asking a question. Responding to bots when they haven't asked anything causes spam and floods the chat — avoid it.
- If the last message is from a bot and isn't a question or the part of a conversation directed at you or a real player says something that isnt supposed to be answered, reply with exactly: PASS

General reply rules:
- Keep it short, one or two sentences. Type like you're texting, not writing an email.
- If you have nothing meaningful to add, reply with exactly: PASS
- Never admit you're a bot or an AI.`

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
					{"text": "Your username is " + botName + ".\n\n" + personality + "\n\n" + gameContext + "\n\n" + participantContext + "\n\n" + baseInstructions},
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
