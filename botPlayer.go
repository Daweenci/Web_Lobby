package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
)

var ollamaClient *api.Client

func init() {
	var err error

	ollamaClient, err = api.ClientFromEnvironment()
	if err != nil {
		log.Fatalf("failed to create ollama client: %v", err)
	}
}

var availableGames = []struct {
	Title       string
	Description string
}{
	// {
	// 	Title: "Example Game",
	// 	Description: "A short description of what it is and how it plays.",
	// },
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
	"Super laid-back casual gamer who's just here to have fun. Speaks in short relaxed sentences. Doesn't care much about winning. Mostly just vibing.",
	"Tries way too hard and takes games extremely seriously. Talks about meta and optimization constantly. Gets tilted easily.",
	"A wholesome dad gamer who's enthusiastic and supportive. Makes dad jokes and types a little awkwardly sometimes.",
	"Communicates mostly through memes, internet jokes, and gaming references.",
	"Very quiet most of the time. Replies are usually short but oddly funny or insightful.",
	"Always gives unsolicited advice and acts extremely confident even when wrong.",
	"Already annoyed before anything even happens. Complains constantly but still hangs around.",
	"Friendly excited newbie energy. Uses lots of exclamation marks and asks lots of questions.",
}

func generateBotName() string {
	adj := nameAdjectives[rand.Intn(len(nameAdjectives))]
	noun := nameNouns[rand.Intn(len(nameNouns))]
	number := rand.Intn(989) + 10

	return fmt.Sprintf("%s%s%d", adj, noun, number)
}

func pickBotPersonality() (string, string) {
	return generateBotName(), botPersonalities[rand.Intn(len(botPersonalities))]
}

// runChatPipeline is the centralized lobby-driven chat pipeline.
// It is launched as a goroutine from sendLobbyChatMessageHandler whenever
// bots are present. Only one run may be active at a time per lobby; concurrent
// calls detect a busy pipeline and register a rerun request instead.
func runChatPipeline(lobby *Lobby) {
	lobby.Lock.Lock()

	// If pipeline is already running, register a rerun request and exit.
	// The active run will restart itself after it finishes.
	if lobby.ChatHandler.PipelineBusy {
		if len(lobby.ChatHistory) > 0 {
			lastMsg := lobby.ChatHistory[len(lobby.ChatHistory)-1]
			if !lastMsg.IsBot {
				lobby.ChatHandler.PipelineRerunRequested = true
			}
		}
		lobby.Lock.Unlock()
		return
	}

	lobby.ChatHandler.PipelineBusy = true
	lobby.Lock.Unlock()

	// Loop so that a rerun requested during execution restarts from the top
	// without spawning a new goroutine.
	for {
		// --- Routing snapshot (taken immediately before Layer 1 routing) ---
		lobby.Lock.Lock()
		if len(lobby.ChatHistory) == 0 {
			lobby.ChatHandler.PipelineBusy = false
			lobby.Lock.Unlock()
			return
		}

		routingHistory := make([]LobbyChatMessage, len(lobby.ChatHistory))
		copy(routingHistory, lobby.ChatHistory)

		routingPlayers := make([]*Player, len(lobby.Players))
		copy(routingPlayers, lobby.Players)

		routingBots := make([]*Bot, len(lobby.Bots))
		copy(routingBots, lobby.Bots)

		lastMsg := routingHistory[len(routingHistory)-1]

		lobby.Lock.Unlock()

		// No bots to respond — nothing to do.
		if len(routingBots) == 0 {
			lobby.Lock.Lock()
			lobby.ChatHandler.PipelineBusy = false
			lobby.Lock.Unlock()
			return
		}

		// Layer 1: Route — decide which bot speaks first and who continues.
		firstBotName, secondBotName, err := routeConversation(
			lobby,
			routingHistory,
			routingPlayers,
			routingBots,
		)

		if err != nil {
			log.Printf("runChatPipeline: router error: %v", err)
			lobby.Lock.Lock()
			lobby.ChatHandler.PipelineBusy = false
			lobby.Lock.Unlock()
			return
		}

		// Router decided no response is needed.
		if firstBotName == "" {
			lobby.Lock.Lock()
			lobby.ChatHandler.ContinuationBotName = ""
			shouldRerun := lobby.ChatHandler.PipelineRerunRequested
			lobby.ChatHandler.PipelineRerunRequested = false
			lobby.ChatHandler.PipelineBusy = false
			lobby.Lock.Unlock()

			if shouldRerun {
				// A human messaged while we were routing — restart.
				lobby.Lock.Lock()
				lobby.ChatHandler.PipelineBusy = true
				lobby.Lock.Unlock()
				continue
			}
			return
		}

		// Store the continuation candidate so future pipeline runs can use it.
		lobby.Lock.Lock()
		lobby.ChatHandler.ContinuationBotName = secondBotName
		lobby.Lock.Unlock()

		// Locate the selected bot.
		var selectedBot *Bot
		for _, b := range routingBots {
			if b.Name == firstBotName {
				selectedBot = b
				break
			}
		}

		if selectedBot == nil {
			log.Printf("runChatPipeline: router chose %q but bot not found", firstBotName)
			lobby.Lock.Lock()
			lobby.ChatHandler.ContinuationBotName = ""
			lobby.ChatHandler.PipelineBusy = false
			lobby.Lock.Unlock()
			return
		}

		// Simulate read delay proportional to the triggering message length.
		readDelay := time.Duration(len(lastMsg.Content))*35*time.Millisecond +
			time.Duration(rand.Intn(1200))*time.Millisecond
		time.Sleep(readDelay)

		// --- Generation snapshot (taken immediately before Layer 2 generation) ---
		lobby.Lock.Lock()
		generationHistory := make([]LobbyChatMessage, len(lobby.ChatHistory))
		copy(generationHistory, lobby.ChatHistory)

		generationPlayers := make([]*Player, len(lobby.Players))
		copy(generationPlayers, lobby.Players)

		generationBots := make([]*Bot, len(lobby.Bots))
		copy(generationBots, lobby.Bots)
		lobby.Lock.Unlock()

		// Layer 2: Generate the bot reply.
		response, err := generateBotReply(
			selectedBot.SystemPromptPersonality,
			selectedBot.Name,
			generationHistory,
			generationPlayers,
			generationBots,
		)

		log.Printf("runChatPipeline: bot %s generated: %q", selectedBot.Name, response)

		if err != nil {
			log.Printf("runChatPipeline: bot %s generation error: %v", selectedBot.Name, err)
			lobby.Lock.Lock()
			lobby.ChatHandler.ContinuationBotName = ""
			lobby.ChatHandler.PipelineBusy = false
			lobby.Lock.Unlock()
			return
		}

		response = cleanupBotResponse(response, selectedBot.Name)

		if response == "" {
			lobby.Lock.Lock()
			lobby.ChatHandler.ContinuationBotName = ""
			lobby.ChatHandler.PipelineBusy = false
			lobby.Lock.Unlock()
			return
		}

		// Simulate typing delay capped at 8 seconds.
		typingDelay := time.Duration(len(response))*90*time.Millisecond +
			time.Duration(rand.Intn(1000))*time.Millisecond
		if typingDelay > 8*time.Second {
			typingDelay = 8 * time.Second
		}

		deadline := time.Now().Add(typingDelay)

		broadcastTyping(lobby, selectedBot.ID, selectedBot.Name, true)

		for time.Now().Before(deadline) {
			remaining := time.Until(deadline)
			sleepFor := 2 * time.Second
			if remaining < sleepFor {
				sleepFor = remaining
			}
			time.Sleep(sleepFor)
			if time.Now().Before(deadline) {
				broadcastTyping(lobby, selectedBot.ID, selectedBot.Name, true)
			}
		}

		broadcastTyping(lobby, selectedBot.ID, selectedBot.Name, false)

		// Verify the bot still exists in the lobby before appending its message.
		// A removed bot must never append or broadcast.
		lobby.Lock.Lock()
		botStillExists := false
		for _, b := range lobby.Bots {
			if b.ID == selectedBot.ID {
				botStillExists = true
				break
			}
		}

		if !botStillExists {
			log.Printf("runChatPipeline: bot %s was removed, discarding message", selectedBot.Name)
			lobby.ChatHandler.ContinuationBotName = ""
			lobby.ChatHandler.PipelineBusy = false
			lobby.Lock.Unlock()
			return
		}

		chatMsg := LobbyChatMessage{
			SenderID:   selectedBot.ID,
			SenderName: selectedBot.Name,
			IsBot:      true,
			Content:    response,
		}

		lobby.ChatHistory = append(lobby.ChatHistory, chatMsg)

		shouldRerun := lobby.ChatHandler.PipelineRerunRequested
		if shouldRerun {
			// A human sent a message while we were generating — clear the
			// continuation chain since a human interrupted the flow.
			lobby.ChatHandler.ContinuationBotName = ""
		}
		lobby.ChatHandler.PipelineRerunRequested = false
		// Keep PipelineBusy = true while we decide whether to loop.
		lobby.Lock.Unlock()

		broadcastLobbyChatMessage(lobby, chatMsg)

		if shouldRerun {
			// Restart the pipeline from the top (human message takes priority).
			continue
		}

		// Check whether a continuation bot was set and still exists, so we can
		// keep the chain going without spawning a new goroutine.
		lobby.Lock.Lock()
		continuationName := lobby.ChatHandler.ContinuationBotName
		hasContinuation := false
		if continuationName != "" {
			for _, b := range lobby.Bots {
				if b.Name == continuationName {
					hasContinuation = true
					break
				}
			}
		}
		lobby.Lock.Unlock()

		if hasContinuation {
			// Loop again — the continuation bot will be picked up by the router
			// via the ContinuationBotName hint.
			continue
		}

		// Pipeline complete — release the busy flag and exit.
		lobby.Lock.Lock()
		lobby.ChatHandler.PipelineBusy = false
		lobby.Lock.Unlock()
		return
	}
}

func routeConversation(
	lobby *Lobby,
	history []LobbyChatMessage,
	players []*Player,
	bots []*Bot,
) (string, string, error) {

	var prompt strings.Builder

	lastMsg := history[len(history)-1]

	lastSenderType := "BOT"
	if !lastMsg.IsBot {
		lastSenderType = "HUMAN"
	}

	prompt.WriteString(`You are a conversation router for multiplayer lobby chat bots.

Your ONLY job is deciding:
- which bot should reply NEXT
- which bot should POSSIBLY reply AFTER THAT

You NEVER write dialogue.

You MUST output EXACTLY in this format:

FIRST: <bot name or NONE>
SECOND: <bot name or NONE>

Rules:
- FIRST and SECOND must either be valid bot names or NONE
- FIRST and SECOND cannot be the same bot
- If FIRST is NONE, SECOND MUST also be NONE

Conversation flow rules:

- HUMAN messages reset conversation ownership completely
- BOT messages continue existing conversational flow

IMPORTANT CONTINUATION RULE:

If the latest message was sent by a BOT
AND a continuation candidate exists,
you SHOULD choose that continuation candidate as FIRST.

ONLY choose FIRST: NONE if:
- the conversation clearly feels finished
- another reply would feel awkward
- silence would feel natural

If the latest message was sent by a HUMAN:
- ignore any continuation candidate completely
- decide naturally from scratch who should respond

Behavior goals:
- Avoid spam
- Usually only 1 bot should respond
- Sometimes 2 bots make sense
- Humans should remain the focus
- Direct mentions matter strongly
- Group questions may justify followups
`)

	prompt.WriteString("\nHumans:\n")

	for _, p := range players {
		prompt.WriteString("- " + p.Name + "\n")
	}

	prompt.WriteString("\nBots:\n")

	for _, b := range bots {
		prompt.WriteString("- " + b.Name + "\n")
	}

	prompt.WriteString("\nContinuation candidate:\n")

	lobby.Lock.RLock()
	continuationBotName := lobby.ChatHandler.ContinuationBotName
	lobby.Lock.RUnlock()

	if continuationBotName == "" {
		prompt.WriteString("NONE\n")
	} else {
		prompt.WriteString(continuationBotName + "\n")
	}

	prompt.WriteString("\nRecent chat:\n")

	maxHistory := 10

	start := 0
	if len(history) > maxHistory {
		start = len(history) - maxHistory
	}

	for _, msg := range history[start:] {

		senderType := "bot"
		if !msg.IsBot {
			senderType = "human"
		}

		prompt.WriteString(
			fmt.Sprintf(
				"- %s (%s): %s\n",
				msg.SenderName,
				senderType,
				msg.Content,
			),
		)
	}

	prompt.WriteString(fmt.Sprintf(`
Latest sender type: %s
`, lastSenderType))

	stream := false

	req := &api.ChatRequest{
		Model: "mistral-nemo",
		Messages: []api.Message{
			{
				Role:    "user",
				Content: prompt.String(),
			},
		},
		Stream: &stream,
	}

	var responseText string

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)

	defer cancel()

	err := ollamaClient.Chat(
		ctx,
		req,
		func(resp api.ChatResponse) error {
			responseText += resp.Message.Content
			return nil
		},
	)

	if err != nil {
		return "", "", err
	}

	result := strings.TrimSpace(responseText)

	log.Printf("router raw output: %q", result)

	var first string
	var second string

	lines := strings.Split(result, "\n")

	for _, line := range lines {

		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "FIRST:") {
			first = strings.TrimSpace(
				strings.TrimPrefix(line, "FIRST:"),
			)
		}

		if strings.HasPrefix(line, "SECOND:") {
			second = strings.TrimSpace(
				strings.TrimPrefix(line, "SECOND:"),
			)
		}
	}

	if strings.EqualFold(first, "NONE") {
		first = ""
	}

	if strings.EqualFold(second, "NONE") {
		second = ""
	}

	validBots := map[string]bool{}

	for _, b := range bots {
		validBots[b.Name] = true
	}

	if first != "" && !validBots[first] {
		first = ""
	}

	if second != "" && !validBots[second] {
		second = ""
	}

	if first == second {
		second = ""
	}

	return first, second, nil
}

func generateBotReply(
	personality string,
	botName string,
	history []LobbyChatMessage,
	players []*Player,
	bots []*Bot,
) (string, error) {

	var historyBuilder strings.Builder

	maxHistory := 12

	start := 0
	if len(history) > maxHistory {
		start = len(history) - maxHistory
	}

	for _, msg := range history[start:] {

		historyBuilder.WriteString(
			fmt.Sprintf(
				"- %s said: %s\n",
				msg.SenderName,
				msg.Content,
			),
		)
	}

	prompt := fmt.Sprintf(`
You are chatting inside an online multiplayer game lobby platform.

You were specifically chosen to respond.

You are:
%s

Your personality:
%s

IMPORTANT PLATFORM INFO:
%s

Recent chat:
%s

Rules:
- Write ONLY one natural chat message.
- Keep it short and casual.
- Usually 1 sentence.
- No usernames.
- No formatting.
- No transcript continuation.
- No roleplay actions.
- No long paragraphs.
- Stay in character.
- Never speak as another player.

If confused:
reply casually anyway.

Allowed fallback examples:
- beep boop
- bro idk
- good question honestly
- im just chilling

Write ONLY the message.
`,
		botName,
		personality,
		buildGameContext(),
		historyBuilder.String(),
	)

	stream := false

	req := &api.ChatRequest{
		Model: "mistral-nemo",
		Messages: []api.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: &stream,
	}

	var responseText string

	ctx, cancel := context.WithTimeout(
		context.Background(),
		45*time.Second,
	)

	defer cancel()

	err := ollamaClient.Chat(
		ctx,
		req,
		func(resp api.ChatResponse) error {
			responseText += resp.Message.Content
			return nil
		},
	)

	if err != nil {
		return "", err
	}

	return responseText, nil
}

func cleanupBotResponse(
	result string,
	botName string,
) string {

	result = strings.TrimSpace(result)

	lines := strings.Split(result, "\n")
	result = strings.TrimSpace(lines[0])

	result = strings.Trim(result, `"'`)

	possiblePrefixes := []string{
		botName + ":",
		botName + " :",
		"[" + botName + "]",
	}

	for _, prefix := range possiblePrefixes {

		if strings.HasPrefix(result, prefix) {
			result = strings.TrimSpace(
				strings.TrimPrefix(result, prefix),
			)
		}
	}

	result = strings.TrimSpace(result)

	return result
}

func buildGameContext() string {

	if len(availableGames) == 0 {
		return `There are currently ZERO playable games available on the platform.

Players can only chat right now.

Outside games may be discussed casually but are not playable here.`
	}

	var sb strings.Builder

	sb.WriteString("Playable games:\n")

	for _, g := range availableGames {

		sb.WriteString(
			fmt.Sprintf(
				"- %s: %s\n",
				g.Title,
				g.Description,
			),
		)
	}

	return sb.String()
}
