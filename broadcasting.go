package main

func broadcastLobbyUpdate(lobby *Lobby) {
	lobby.Lock.RLock()
	// Copying mutable fields, leaving immutable ones out
	playersCopy := make([]*Player, len(lobby.Players))
	copy(playersCopy, lobby.Players)
	gameStartCopy := make([]PlayerStarted, len(lobby.GameStart))
	copy(gameStartCopy, lobby.GameStart)
	botsCopy := make([]*Bot, len(lobby.Bots))
	copy(botsCopy, lobby.Bots)
	lobby.Lock.RUnlock()
	updatedLobby := LobbyDTO{
		ID:         lobby.ID,
		Name:       lobby.Name,
		MaxPlayers: lobby.MaxPlayers,
		IsPrivate:  lobby.IsPrivate,
		Players:    toPlayerResponses(playersCopy),
		Bots:       toBotResponses(botsCopy),
		GameStart:  gameStartCopy,
	}

	for _, player := range playersCopy {
		lobbyUpdatedResponse := LobbyUpdatedResponse{
			BaseResponse: newBaseResponse(ResponseLobbyUpdated),
			Lobby:        updatedLobby,
		}
		sendResponse(player, lobbyUpdatedResponse)
	}
}

func broadcastLobbies() {
	lobbiesLock.RLock()
	lobbiesCopy := make([]*Lobby, 0, len(lobbies))
	for _, l := range lobbies {
		lobbiesCopy = append(lobbiesCopy, l)
	}
	lobbiesLock.RUnlock()

	lobbiesResponse := make([]LobbyDTO, 0, len(lobbiesCopy))
	for _, lobby := range lobbiesCopy {
		lobby.Lock.RLock()
		playersCopy := make([]*Player, len(lobby.Players))
		copy(playersCopy, lobby.Players)
		gameStartCopy := make([]PlayerStarted, len(lobby.GameStart))
		copy(gameStartCopy, lobby.GameStart)
		botsCopy := make([]*Bot, len(lobby.Bots))
		copy(botsCopy, lobby.Bots)
		lobbiesResponse = append(lobbiesResponse, LobbyDTO{
			ID:         lobby.ID,
			Name:       lobby.Name,
			MaxPlayers: lobby.MaxPlayers,
			IsPrivate:  lobby.IsPrivate,
			Players:    toPlayerResponses(playersCopy),
			Bots:       toBotResponses(botsCopy),
			GameStart:  gameStartCopy,
		})
		lobby.Lock.RUnlock()
	}

	activePlayersLock.RLock()
	activePlayersCopy := make([]*Player, 0, len(activePlayers))
	for _, p := range activePlayers {
		activePlayersCopy = append(activePlayersCopy, p)
	}
	activePlayersLock.RUnlock()
	for _, player := range activePlayersCopy {
		lobbiesUpdateResponse := LobbiesUpdateResponse{
			BaseResponse: newBaseResponse(ResponseLobbyList),
			Lobbies:      lobbiesResponse,
		}
		sendResponse(player, lobbiesUpdateResponse)
	}
}

func broadcastLobbyChatMessage(lobby *Lobby, msg LobbyChatMessage) {
	lobby.Lock.RLock()
	playersCopy := make([]*Player, len(lobby.Players))
	copy(playersCopy, lobby.Players)
	lobby.Lock.RUnlock()

	response := LobbyChatMessageResponse{
		BaseResponse: newBaseResponse(ResponseLobbyChatMessage),
		LobbyID:      lobby.ID,
		SenderName:   msg.SenderName,
		SenderID:     msg.SenderID,
		Content:      msg.Message,
		IsBot:        msg.IsBot,
	}

	for _, player := range playersCopy {
		sendResponse(player, response)
	}
}

func broadcastTyping(lobby *Lobby, playerID string, playerName string, isTyping bool) {
	lobby.Lock.RLock()
	playersCopy := make([]*Player, len(lobby.Players))
	copy(playersCopy, lobby.Players)
	lobby.Lock.RUnlock()

	response := PlayerTypingResponse{
		BaseResponse: newBaseResponse(ResponsePlayerTyping),
		PlayerID:     playerID,
		PlayerName:   playerName,
		IsTyping:     isTyping,
	}

	for _, player := range playersCopy {
		// Don't send to the player who is typing
		if player.ID == playerID {
			continue
		}
		sendResponse(player, response)
	}
}

func broadcastBotJoined(lobby *Lobby, bot *Bot) {
	lobby.Lock.RLock()
	playersCopy := make([]*Player, len(lobby.Players))
	copy(playersCopy, lobby.Players)
	lobby.Lock.RUnlock()

	for _, p := range playersCopy {
		sendResponse(p, BotJoinedResponse{
			BaseResponse: newBaseResponse(ResponseBotJoined),
			Bot:          BotDTO{ID: bot.ID, Name: bot.Name},
		})
	}
}

func broadcastBotLeft(lobby *Lobby, botID string) {
	lobby.Lock.RLock()
	playersCopy := make([]*Player, len(lobby.Players))
	copy(playersCopy, lobby.Players)
	lobby.Lock.RUnlock()

	for _, p := range playersCopy {
		sendResponse(p, BotLeftResponse{
			BaseResponse: newBaseResponse(ResponseBotLeft),
			BotID:        botID,
		})
	}
}
