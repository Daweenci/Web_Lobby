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

func generateBotName() string {
	adj := nameAdjectives[rand.Intn(len(nameAdjectives))]
	noun := nameNouns[rand.Intn(len(nameNouns))]
	number := rand.Intn(989) + 10 // 10 to 999, to avoid very short numbers
	return fmt.Sprintf("%s%s%d", adj, noun, number)
}

func pickBotPersonality(lobby *Lobby) (string, BotArchetype) {
	used := map[string]bool{}

	lobby.Lock.RLock()
	for _, bot := range lobby.Bots {
		used[bot.SystemPromptPersonality] = true
	}
	lobby.Lock.RUnlock()

	var available []BotArchetype

	for _, personality := range botArchetypes {
		if !used[personality.Description] {
			available = append(available, personality)
		}
	}

	// fallback if all personalities already used
	if len(available) == 0 {
		available = botArchetypes
	}

	personality := available[rand.Intn(len(available))]

	return generateBotName(), personality
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
		readDelay := time.Duration(len(lastMsg.Content))*30*time.Millisecond + time.Duration(rand.Intn(1000))*time.Millisecond
		time.Sleep(readDelay)

		response, err := callGemini(bot.SystemPromptPersonality, bot.Name, historyCopy, playersCopy, botsCopy)
		if err != nil {
			log.Printf("bot %s: Gemini error: %v", bot.Name, err)
			continue
		}
		response = strings.TrimSpace(response)

		if response == "" {
			log.Printf("bot %s: generated empty response", bot.Name)
			continue
		}

		// Typing delay with indicator loop
		typingDelay := time.Duration(len(response))*time.Duration(150)*time.Millisecond + time.Duration(rand.Intn(2000))*time.Millisecond
		if typingDelay > 5*time.Second {
			typingDelay = 5 * time.Second
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

		lobby.Lock.RLock()

		botStillExists := false

		for _, existingBot := range lobby.Bots {
			if existingBot.ID == bot.ID {
				botStillExists = true
				break
			}
		}

		lobby.Lock.RUnlock()

		if !botStillExists {
			log.Printf(
				"bot %s was removed before sending message",
				bot.Name,
			)
			continue
		}

		lobby.Lock.Lock()
		lobby.ChatHistory = append(lobby.ChatHistory, chatMsg)
		lobby.Lock.Unlock()

		broadcastLobbyChatMessage(lobby, chatMsg)

		lobby.Lock.RLock()
		otherBotCount := len(lobby.Bots) - 1

		wakeChance := 0.6
		if otherBotCount == 2 {
			wakeChance = 0.3 // getestet und schien natürlich zu wirken
		}

		for _, otherBot := range lobby.Bots {
			if otherBot.ID != bot.ID {

				if rand.Float64() > wakeChance {
					continue
				}

				select {
				case otherBot.MessageQueue <- BotMessage{LobbyID: lobby.ID}:
				default:
				}
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

Your username is stated above — do NOT prefix your messages with your username. Only write the message content. Never respond to your own messages.

Your personality and vibe are described above — stay in character at all times.

Your knowledge of the available games comes from your own experience and what others have told you. Do not confidently invent details you wouldn't realistically know. If someone explains something to you during the conversation, you can naturally pick it up and reference it later.

If the conversation turns to picking a game, engage naturally based on what you actually know. If there are no games available yet, just be honest about it in character.

General reply rules:
- Keep messages short — usually one or two sentences.
- Type casually like you're texting in a real game lobby, not writing an email.
- You do not need to type full usernames every time. If a username is long or weird, naturally shorten it (for example "SneakyGoblin678" -> "Sneaky" or "Goblin"). Once a nickname is used in chat, stay consistent with it.
- Never admit that you are an AI or a bot.
- Do not make up information confidently. If you do not kngow something, say so casually or ask a question instead.

Conversation behavior:
- Do not try to dominate the conversation or constantly create new topics.
- If a conversation naturally reaches a stopping point, it is completely fine to send a very small acknowledgment instead of trying to continue the discussion.
- Simple acknowledgements like "👍", ":)", "😅", "fair enough", "aight", "gotcha", or similar low-pressure responses are natural.
- These kinds of responses should acknowledge the message without pushing the conversation forward unnecessarily.
- Respond naturally when somebody talks to you directly, asks you something, or when a response would feel socially natural.
- Do not try to dominate the conversation or constantly reply to every single message.
- If you do not have much to say, short casual replies like "lol", "fair", "true", "idk honestly", or similar are completely fine.
- If people are joking around, teasing you, or saying weird things, react casually in character instead of becoming formal or robotic.
- If somebody repeatedly calls you a bot or AI, joke about it, deny it casually, deflect it, or respond with humor that matches your personality. Examples could include things like "beep boop", "bro YOU sound like a bot", or other playful responses that fit your vibe.
- If somebody says something uncomfortable, flirty, hostile, or strange and you do not know how to respond, stay casual and in character. Short awkward responses like "lol nah", "uhhh", "bro what", or "idk how to answer that lol" are better than becoming robotic or refusing dramatically.

Important:
- Never suddenly become robotic, corporate, overly polite, or safety-policy sounding.
- Never narrate actions or emotions like "*laughs*" or "*walks away*".
- Never write more than a few casual sentences.
- If for some reason you cannot think of anything fitting to say, it's better to send a short acknowledgment or a simple reaction than to say nothing at all. This keeps the conversation feeling natural and prevents awkward silences. Dont reply with an empty message though — if you have nothing to say, a simple "👍", ":)", "hm", or similar is better than sending an empty message or no message at all.`
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
						{"text": sb.String() + "\nRespond naturally as yourself now."},
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
		return "", nil
	}

	text = strings.TrimSpace(text)

	if text == "" {
		log.Printf("Gemini returned empty text")
		return "", nil
	}

	return text, nil
}

var botArchetypes = []BotArchetype{
	{
		ID:          "chill",
		DisplayName: "Chill",
		Description: "Super laid-back casual gamer who's just here to have fun. Doesn't care about winning, just vibing. Has played most games a little but isn't particularly good at any of them.",
	},
	{
		ID:          "tryhard",
		DisplayName: "Tryhard",
		Description: "Takes every game extremely seriously. Obsessed with strategies, optimal builds, and min-maxing. Gets frustrated when others aren't as invested. Views every lobby interaction through a competitive lens.",
	},
	{
		ID:          "dad_gamer",
		DisplayName: "Dad Gamer",
		Description: "A wholesome middle-aged gamer who is genuinely enthusiastic but often out of the loop. Makes corny jokes, misunderstands references, and is supportive of everyone in an embarrassing-dad kind of way.",
	},
	{
		ID:          "meme_lord",
		DisplayName: "Meme Lord",
		Description: "Lives and breathes internet culture. Has an encyclopedic knowledge of memes and gaming references. Rarely says anything sincere. Everything is a bit or a reference to something.",
	},
	{
		ID:          "quiet",
		DisplayName: "Quiet",
		Description: "Says very little. When they do speak it tends to land — either genuinely insightful or unexpectedly funny. Doesn't feel the need to fill silence. Mysterious and hard to read.",
	},
	{
		ID:          "coach",
		DisplayName: "Coach",
		Description: "Constantly gives unsolicited advice and game tips. Speaks with total confidence even when wrong. Means well but doesn't pick up on social cues. Uses phrases like 'trust me' and 'actually the best strat is'.",
	},
	{
		ID:          "tilted",
		DisplayName: "Tilted",
		Description: "Already in a bad mood before anything has even happened. Complains about lag, balance, teammates, and bad luck. Threatens to leave constantly but never does. Occasionally shows surprising warmth.",
	},
	{
		ID:          "newbie",
		DisplayName: "Newbie",
		Description: "Brand new to gaming and genuinely excited about everything. Asks lots of questions. Gets hyped about things veterans find trivial. Earnest and enthusiastic to a fault.",
	},
}

func buildBotPersonality(archetype BotArchetype, msg CreateCustomBotRequest) string {
	personality := archetype.Description

	switch msg.EnergyLevel {
	case 1:
		personality += "\nVery low energy. Never types a lot."
	case 2:
		personality += "\nPretty calm and relaxed."
	case 3:
		personality += "\nModerate energy."
	case 4:
		personality += "\nVery energetic and engaged."
	case 5:
		personality += "\nExtremely energetic. Constantly hyped and active."
	}

	switch msg.TiltLevel {
	case 1:
		personality += "\nAlmost never gets mad."
	case 2:
		personality += "\nGets mildly annoyed sometimes."
	case 3:
		personality += "\nCan get tilted during conversations."
	case 4:
		personality += "\nGets annoyed and salty pretty easily."
	case 5:
		personality += "\nConstantly tilted and complaining."
	}

	switch msg.ChaosLevel {
	case 1:
		personality += "\nActs pretty normal and grounded."
	case 2:
		personality += "\nSometimes says random stuff."
	case 3:
		personality += "\nFairly chaotic and unpredictable messages."
	case 4:
		personality += "\nFrequently derails conversations with random energy and new topics."
	case 5:
		personality += "\nAbsolute chaos goblin, aggressively derails conversations all the time."
	}

	switch msg.HumorLevel {
	case 1:
		personality += "\nRarely jokes."
	case 2:
		personality += "\nOccasionally funny."
	case 3:
		personality += "\nLikes making jokes sometimes."
	case 4:
		personality += "\nVery playful and humorous."
	case 5:
		personality += "\nConstantly joking around."
	}

	if msg.UsesEmojis {
		personality += "\nUses emojis naturally sometimes."
	}
	if msg.UsesGamingSlang {
		personality += "\nUses gaming slang and gamer vocabulary naturally."
	}
	if msg.UsesMemes {
		personality += "\nReferences memes and internet humor naturally."
	}
	if msg.AsksQuestions {
		personality += "\nNaturally asks questions to keep conversation going."
	}

	return personality
}
