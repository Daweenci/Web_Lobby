package main

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

func joinLobbyHandler(msg JoinLobbyRequest) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		log.Println("joinLobbyHandler: Lobby not found")
		return
	}

	activePlayersLock.RLock()
	player, ok := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !ok {
		log.Println("joinLobbyHandler: Player not found")
		disconnectPlayer(msg.PlayerID)
		return
	}

	lobby.Lock.Lock()
	for _, p := range lobby.Players {
		if p.ID == msg.PlayerID {
			lobby.Lock.Unlock()
			return
		}
	}
	lobby.Lock.Unlock()

	// Leave current lobby if in one
	lobbiesLock.RLock()
	for _, l := range lobbies {
		if l.ID == msg.LobbyID {
			continue
		}
		l.Lock.Lock()
		for i, p := range l.Players {
			if p.ID == msg.PlayerID {
				l.Players = append(l.Players[:i], l.Players[i+1:]...)
				l.Lock.Unlock()
				lobbiesLock.RUnlock()
				if len(l.Players) == 0 {
					lobbiesLock.Lock()
					delete(lobbies, l.ID)
					lobbiesLock.Unlock()
				} else {
					broadcastLobbyUpdate(l)
				}
				broadcastLobbies()
				goto doneLeaving
			}
		}
		l.Lock.Unlock()
	}
	lobbiesLock.RUnlock()
doneLeaving:

	// Check if player has a valid invite for this lobby
	hasInvite := false
	pendingInvitesLock.RLock()
	for _, lobbyID := range pendingInvites[msg.PlayerID] {
		if lobbyID == msg.LobbyID {
			hasInvite = true
			break
		}
	}
	pendingInvitesLock.RUnlock()

	// Only check password if no valid invite
	if !hasInvite && lobby.Password != msg.Password {
		sendResponse(player, LobbyJoinFailedResponse{
			BaseResponse: newBaseResponse(ResponseJoinLobbyFailed),
			Message:      "Incorrect password",
		})
		return
	}

	lobby.Lock.Lock()
	// Check lobby capacity
	if len(lobby.Players)+len(lobby.Bots) >= lobby.MaxPlayers {
		sendResponse(player, LobbyJoinFailedResponse{
			BaseResponse: newBaseResponse(ResponseJoinLobbyFailed),
			Message:      "Lobby is full",
		})
		lobby.Lock.Unlock()
		return
	}

	// Add player
	lobby.Players = append(lobby.Players, player)
	lobbyResponse := LobbyDTO{
		ID:         lobby.ID,
		Name:       lobby.Name,
		MaxPlayers: lobby.MaxPlayers,
		IsPrivate:  lobby.IsPrivate,
		Players:    toPlayerResponses(lobby.Players),
		Bots:       toBotResponses(lobby.Bots),
		GameStart:  lobby.GameStart,
	}
	lobby.Lock.Unlock()

	// Always clean up any pending invite for this lobby
	pendingInvitesLock.Lock()
	invites := pendingInvites[msg.PlayerID]
	for i, lobbyID := range invites {
		if lobbyID == msg.LobbyID {
			pendingInvites[msg.PlayerID] = append(invites[:i], invites[i+1:]...)
			break
		}
	}
	if len(pendingInvites[msg.PlayerID]) == 0 {
		delete(pendingInvites, msg.PlayerID)
	}
	pendingInvitesLock.Unlock()

	sendResponse(player, SuccessfulJoinLobbyResponse{
		BaseResponse: newBaseResponse(ResponseJoinLobbySuccessful),
		Lobby:        lobbyResponse,
		ChatHistory:  lobby.ChatHistory,
	})
	broadcastLobbyUpdate(lobby)
	broadcastLobbies()
}

func leaveLobbyHandler(msg LeaveLobbyRequest) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		log.Println("leaveLobbyHandler: Lobby not found")
		return
	}

	activePlayersLock.RLock()
	player, ok := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !ok {
		log.Println("leaveLobbyHandler: Player not found")
		disconnectPlayer(player.ID)
		return
	}

	lobby.Lock.Lock()
	for i := len(lobby.GameStart) - 1; i >= 0; i-- {
		if lobby.GameStart[i].ID == player.ID {
			lobby.GameStart = append(lobby.GameStart[:i], lobby.GameStart[i+1:]...)
			break
		}
	}

	for i := len(lobby.Players) - 1; i >= 0; i-- {
		if lobby.Players[i].ID == player.ID {
			lobby.Players = append(lobby.Players[:i], lobby.Players[i+1:]...)
			break
		}
	}
	lobby.Lock.Unlock()

	lobbyDeleted := false
	if len(lobby.Players) == 0 {
		lobbiesLock.Lock()
		delete(lobbies, lobby.ID)
		lobbiesLock.Unlock()
		lobbyDeleted = true
	}
	if !lobbyDeleted {
		broadcastLobbyUpdate(lobby)
	}

	sendResponse(player, LobbyLeftResponse{
		BaseResponse: newBaseResponse(ResponseLobbyLeft),
	})
	broadcastLobbies()
}

func createLobbyHandler(msg CreateLobbyRequest) {
	activePlayersLock.RLock()
	player, ok := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !ok {
		log.Println("createLobbyHandler: Player not found")
		disconnectPlayer(msg.PlayerID)
		return
	}
	lobbyID := uuid.New().String()

	newLobby := &Lobby{
		ID:         lobbyID,
		Name:       msg.LobbyName,
		MaxPlayers: msg.MaxPlayers,
		IsPrivate:  msg.IsPrivate,
		Password:   msg.Password,
		Players:    []*Player{player},
		GameStart:  []PlayerStarted{},
	}

	lobbiesLock.Lock()
	lobbies[lobbyID] = newLobby
	lobbiesLock.Unlock()

	newLobbyResponse := LobbyDTO{
		ID:         newLobby.ID,
		Name:       newLobby.Name,
		MaxPlayers: newLobby.MaxPlayers,
		IsPrivate:  newLobby.IsPrivate,
		Players:    toPlayerResponses(newLobby.Players),
		GameStart:  newLobby.GameStart,
	}

	createLobbyResponse := CreateLobbyResponse{
		BaseResponse: newBaseResponse(ResponseLobbyCreated),
		Lobby:        newLobbyResponse,
	}
	sendResponse(player, createLobbyResponse)
	broadcastLobbies()
}

func startGameHandler(msg StartGame) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		log.Println("StartGameHandler: Lobby not found")
		return
	}

	activePlayersLock.RLock()
	player, ok := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !ok {
		log.Println("StartGameHandler: Player not found")
		disconnectPlayer(msg.PlayerID)
		return
	}

	lobby.Lock.Lock()
	alreadyStarted := false
	for _, p := range lobby.GameStart {
		if p.ID == player.ID {
			alreadyStarted = true
			break
		}
	}
	if !alreadyStarted {
		lobby.GameStart = append(lobby.GameStart, PlayerStarted{ID: player.ID})
	}
	lobby.Lock.Unlock()

	broadcastLobbyUpdate(lobby)
}

func cancelGameHandler(msg CancelGame) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		log.Println("cancelGameHandler: Lobby not found")
		return
	}

	activePlayersLock.RLock()
	player, ok := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !ok {
		log.Println("cancelGameHandler: Player not found")
		disconnectPlayer(msg.PlayerID)
		return
	}

	lobby.Lock.Lock()
	for i := len(lobby.GameStart) - 1; i >= 0; i-- {
		if lobby.GameStart[i].ID == player.ID {
			lobby.GameStart = append(lobby.GameStart[:i], lobby.GameStart[i+1:]...)
			break
		}
	}
	lobby.Lock.Unlock()

	broadcastLobbyUpdate(lobby)
}

func sendFriendRequestHandler(msg AddFriendRequest) {
	activePlayersLock.RLock()
	player, playerOK := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !playerOK {
		log.Println("sendFriendRequestHandler: Player not found")
		disconnectPlayer(msg.PlayerID)
		return
	}

	friendID, friendOK := getPlayerIdByName(msg.FriendName)
	if !friendOK {
		sendFriendRequestResult(player, false, "Player not found")
		return
	}

	if msg.PlayerID == friendID {
		sendFriendRequestResult(player, false, "You cannot add yourself")
		return
	}

	if areFriends(msg.PlayerID, friendID) {
		sendFriendRequestResult(player, false, "You are already friends")
		return
	}

	err := createFriendRequest(msg.PlayerID, friendID)
	if err != nil {
		if errors.Is(err, ErrFriendRequestExists) {
			sendFriendRequestResult(player, false, "Friend request already sent")
			return
		}

		log.Printf("sendFriendRequestHandler: %v", err)
		sendFriendRequestResult(player, false, "Error creating friend request")
		return
	}

	sendFriendRequestResult(player, true, "Friend request sent")

	activePlayersLock.RLock()
	friend, ok := activePlayers[friendID]
	activePlayersLock.RUnlock()
	if !ok {
		log.Println("sendFriendRequestHandler: Friend not online")
		return
	}

	sendResponse(friend, FriendRequestReceivedResponse{
		BaseResponse: newBaseResponse(ResponseFriendRequestReceived),
		Player: PlayerDTO{
			ID:   player.ID,
			Name: player.Name,
		},
	})
}

func sendFriendRequestResult(player *Player, success bool, message string) {
	sendResponse(player, FriendRequestSentResponse{
		BaseResponse: newBaseResponse(ResponseFriendRequestSent),
		Success:      success,
		Message:      message,
	})
}

func acceptFriendRequestHandler(msg AcceptFriendRequestRequest) {
	friendID := msg.FriendID
	playerID := msg.PlayerID
	acceptRequest := msg.AcceptRequest
	err := handleFriendRequest(friendID, playerID, acceptRequest)

	if err != nil {
		activePlayersLock.RLock()
		player, ok := activePlayers[playerID]
		activePlayersLock.RUnlock()

		if ok {
			sendErrorToPlayer(player, "Error handling friend request")
		}

		log.Printf("acceptFriendRequestHandler: %v", err)
		return
	}

	activePlayersLock.RLock()
	player, playerOk := activePlayers[msg.PlayerID]
	friend, friendOk := activePlayers[friendID]
	activePlayersLock.RUnlock()
	if !playerOk {
		log.Println("acceptFriendRequestHandler: Player not found")
		disconnectPlayer(msg.PlayerID)
		return
	}
	pendingFriendRequestsPlayer := getPendingFriendRequests(playerID)
	pendingFriendRequestsResponsePlayer := PendingFriendRequestsResponse{
		BaseResponse:          newBaseResponse(ResponsePendingFriendRequests),
		PendingFriendRequests: pendingFriendRequestsPlayer,
	}
	sendResponse(player, pendingFriendRequestsResponsePlayer)
	if acceptRequest {
		friendsListResponse := FriendsListResponse{
			BaseResponse: newBaseResponse(ResponseFriendsList),
			FriendsList:  getFriendsWithOnlineStatus(playerID),
		}
		sendResponse(player, friendsListResponse)
	}

	if !friendOk {
		log.Println("acceptFriendRequestHandler: Friend not online")
		return
	}
	pendingFriendRequestsFriend := getPendingFriendRequests(friendID)
	pendingFriendRequestsResponseFriend := PendingFriendRequestsResponse{
		BaseResponse:          newBaseResponse(ResponsePendingFriendRequests),
		PendingFriendRequests: pendingFriendRequestsFriend,
	}
	sendResponse(friend, pendingFriendRequestsResponseFriend)

	if acceptRequest && friendOk {
		playerInfo, err := getPlayerByID(playerID)
		if err != nil {
			log.Printf("acceptFriendRequestHandler: Error getting player info: %v", err)
			return
		}
		playerName := playerInfo.Username
		friendRequestAcceptedResponse := FriendRequestAcceptedResponse{
			BaseResponse: newBaseResponse(ResponseFriendRequestAccepted),
			Friend: FriendDTO{
				ID:       player.ID,
				Name:     playerName,
				IsOnline: true,
			},
		}
		sendResponse(friend, friendRequestAcceptedResponse)
	}
}

func pingAllFriendsOnlineStatusHandler(playerID string, isOnline bool) {
	player, err := getPlayerByID(playerID)
	if err != nil {
		log.Printf("pingAllFriendsHandler: Error getting player by ID: %v", err)
		return
	}
	friendsList := getFriendsWithOnlineStatus(playerID)
	friendCameOnline := FriendOnlineStatusResponse{
		BaseResponse: newBaseResponse(ResponseFriendOnlineStatus),
		Friend:       FriendDTO{ID: player.ID, Name: player.Username, IsOnline: isOnline},
	}

	for _, friend := range friendsList {
		activePlayersLock.RLock()
		friend, ok := activePlayers[friend.ID]
		activePlayersLock.RUnlock()
		if ok {
			sendResponse(friend, friendCameOnline)
		}
	}
}

func getFriendsWithOnlineStatus(playerID string) []FriendDTO {
	friendsList := getFriendsList(playerID)
	friendsListWithOnlineStatus := make([]FriendDTO, len(friendsList))
	for i, f := range friendsList {
		activePlayersLock.RLock()
		_, isOnline := activePlayers[f.ID]
		activePlayersLock.RUnlock()
		friendsListWithOnlineStatus[i] = FriendDTO{
			ID:       f.ID,
			Name:     f.Name,
			IsOnline: isOnline,
		}
	}
	return friendsListWithOnlineStatus
}

func sendLobbyChatMessageHandler(msg SendLobbyChatMessageRequest) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		log.Println("sendLobbyChatMessageHandler: Lobby not found")
		return
	}

	activePlayersLock.RLock()
	player, ok := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !ok {
		log.Println("sendLobbyChatMessageHandler: Player not found")
		return
	}

	lobby.Lock.RLock()
	inLobby := false
	for _, p := range lobby.Players {
		if p.ID == player.ID {
			inLobby = true
			break
		}
	}
	lobby.Lock.RUnlock()
	if !inLobby {
		sendErrorToPlayer(player, "You are not in this lobby")
		return
	}

	chatMsg := LobbyChatMessage{
		SenderID:   player.ID,
		SenderName: player.Name,
		IsBot:      false,
		Content:    msg.Content,
	}
	lobby.Lock.Lock()
	lobby.ChatHistory = append(lobby.ChatHistory, chatMsg)
	lobby.Lock.Unlock()

	// Broadcast to all players in lobby
	broadcastLobbyChatMessage(lobby, chatMsg)

	stopTyping(player.ID, lobby)

	lobby.Lock.RLock()
	botsCopy := make([]*Bot, len(lobby.Bots))
	copy(botsCopy, lobby.Bots)
	lobby.Lock.RUnlock()

	rand.Shuffle(len(botsCopy), func(i, j int) {
		botsCopy[i], botsCopy[j] = botsCopy[j], botsCopy[i]
	})

	go func() {
		for i, bot := range botsCopy {

			if i > 0 {
				time.Sleep(2500 * time.Millisecond) // WOW das hat alles so viel besser gemacht unglaublich
			}

			select {
			case bot.MessageQueue <- BotMessage{LobbyID: lobby.ID}:
				log.Printf("Queued bot %s after delay", bot.Name)
			default:
				log.Printf("Skipped bot %s because queue already full", bot.Name)
			}
		}
	}()
}

var (
	typingTimers     = make(map[string]*time.Timer) // playerID -> timer
	typingTimersLock sync.Mutex
)

func typingHandler(msg TypingRequest) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		return
	}

	activePlayersLock.RLock()
	player, ok := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !ok {
		return
	}

	if !msg.IsTyping {
		stopTyping(msg.PlayerID, lobby)
		return
	}

	broadcastTyping(lobby, msg.PlayerID, player.Name, true)

	typingTimersLock.Lock()
	if timer, exists := typingTimers[msg.PlayerID]; exists {
		timer.Stop()
	}
	typingTimers[msg.PlayerID] = time.AfterFunc(5*time.Second, func() { //Fallback falls Frontend nicht selbst ein Signal zum Stoppen schickt
		stopTyping(msg.PlayerID, lobby)
	})
	typingTimersLock.Unlock()
}

func stopTyping(playerID string, lobby *Lobby) {
	typingTimersLock.Lock()
	if timer, exists := typingTimers[playerID]; exists {
		timer.Stop()
		delete(typingTimers, playerID)
	}
	typingTimersLock.Unlock()

	activePlayersLock.RLock()
	player, ok := activePlayers[playerID]
	activePlayersLock.RUnlock()
	if !ok {
		return
	}

	broadcastTyping(lobby, playerID, player.Name, false)
}

func addBotToLobbyHandler(msg AddBotToLobbyRequest) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		log.Println("addBotToLobbyHandler: Lobby not found")
		return
	}

	lobby.Lock.Lock()
	if len(lobby.Players)+len(lobby.Bots) >= lobby.MaxPlayers {
		lobby.Lock.Unlock()
		broadcastToLobbyPlayers(lobby, BotCreationFailedResponse{
			BaseResponse: newBaseResponse(ResponseBotCreationFailed),
			Message:      "Lobby is full",
		})
		return
	}
	if lobby.BotCreating {
		lobby.Lock.Unlock()
		broadcastToLobbyPlayers(lobby, BotCreationFailedResponse{
			BaseResponse: newBaseResponse(ResponseBotCreationFailed),
			Message:      "A bot is already being created",
		})
		return
	}
	lobby.BotCreating = true
	lobby.Lock.Unlock()

	broadcastToLobbyPlayers(lobby, BotCreatingResponse{
		BaseResponse: newBaseResponse(ResponseBotCreating),
	})

	name, archetype := pickBotPersonality(lobby)

	bot := &Bot{
		ID:                      uuid.New().String(),
		Name:                    name,
		SystemPromptPersonality: archetype.Description,
		MessageQueue:            make(chan BotMessage, 1),
	}

	lobby.Lock.Lock()
	if len(lobby.Players)+len(lobby.Bots) >= lobby.MaxPlayers {
		lobby.BotCreating = false
		lobby.Lock.Unlock()
		broadcastToLobbyPlayers(lobby, BotCreationFailedResponse{
			BaseResponse: newBaseResponse(ResponseBotCreationFailed),
			Message:      "Lobby is full",
		})
		return
	}
	lobby.Bots = append(lobby.Bots, bot)
	lobby.BotCreating = false
	lobby.Lock.Unlock()

	go bot.Start(lobby)

	broadcastBotJoined(lobby, bot)
	broadcastLobbyUpdate(lobby)
	broadcastLobbies()
	log.Println("Bot created with personality:", archetype.Description)
}

func removeBotFromLobbyHandler(msg RemoveBotFromLobbyRequest) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		log.Println("removeBotFromLobbyHandler: Lobby not found")
		return
	}

	lobby.Lock.Lock()
	removed := false
	botName := ""
	for i := len(lobby.Bots) - 1; i >= 0; i-- {
		if lobby.Bots[i].ID == msg.BotID {
			botName = lobby.Bots[i].Name
			close(lobby.Bots[i].MessageQueue)
			lobby.Bots = append(lobby.Bots[:i], lobby.Bots[i+1:]...)
			removed = true
			break
		}
	}
	lobby.Lock.Unlock()

	if !removed {
		log.Println("removeBotFromLobbyHandler: Bot not found")
		return
	}

	broadcastBotLeft(lobby, msg.BotID, botName)
	broadcastLobbyUpdate(lobby)
	broadcastLobbies()
}

func declineInviteHandler(msg DeclineInviteRequest) {
	pendingInvitesLock.Lock()
	invites := pendingInvites[msg.PlayerID]
	for i, lobbyID := range invites {
		if lobbyID == msg.LobbyID {
			pendingInvites[msg.PlayerID] = append(invites[:i], invites[i+1:]...)
			break
		}
	}
	if len(pendingInvites[msg.PlayerID]) == 0 {
		delete(pendingInvites, msg.PlayerID)
	}
	pendingInvitesLock.Unlock()

	activePlayersLock.RLock()
	player, ok := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !ok {
		return
	}

	sendResponse(player, InviteDeclinedResponse{
		BaseResponse: newBaseResponse(ResponseInviteDeclined),
		LobbyID:      msg.LobbyID,
	})
}

func inviteToLobbyHandler(msg InviteToLobbyRequest) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		log.Println("inviteToLobbyHandler: Lobby not found")
		return
	}

	activePlayersLock.RLock()
	player, ok := activePlayers[msg.PlayerID]
	activePlayersLock.RUnlock()
	if !ok {
		log.Println("inviteToLobbyHandler: Player not found")
		disconnectPlayer(msg.PlayerID)
		return
	}

	// Check lobby is not full
	lobby.Lock.RLock()
	isFull := len(lobby.Players)+len(lobby.Bots) >= lobby.MaxPlayers
	lobby.Lock.RUnlock()
	if isFull {
		sendResponse(player, InviteSentResponse{
			BaseResponse: newBaseResponse(ResponseInviteSent),
			Success:      false,
			Message:      "Lobby is full",
		})
		return
	}

	// Check invitee exists
	activePlayersLock.RLock()
	friend, friendOnline := activePlayers[msg.FriendID]
	activePlayersLock.RUnlock()
	if !friendOnline {
		sendResponse(player, InviteSentResponse{
			BaseResponse: newBaseResponse(ResponseInviteSent),
			Success:      false,
			Message:      "Player is not online",
		})
		return
	}

	// Check if friend is already in this lobby
	lobby.Lock.RLock()
	for _, p := range lobby.Players {
		if p.ID == msg.FriendID {
			lobby.Lock.RUnlock()
			sendResponse(player, InviteSentResponse{
				BaseResponse: newBaseResponse(ResponseInviteSent),
				Success:      false,
				Message:      "Player is already in this lobby",
			})
			return
		}
	}
	lobby.Lock.RUnlock()

	// Check if invite already pending
	pendingInvitesLock.RLock()
	for _, lobbyID := range pendingInvites[msg.FriendID] {
		if lobbyID == msg.LobbyID {
			pendingInvitesLock.RUnlock()
			sendResponse(player, InviteSentResponse{
				BaseResponse: newBaseResponse(ResponseInviteSent),
				Success:      false,
				Message:      "Invite already sent",
			})
			return
		}
	}
	pendingInvitesLock.RUnlock()

	// Add invite to map
	pendingInvitesLock.Lock()
	pendingInvites[msg.FriendID] = append(pendingInvites[msg.FriendID], msg.LobbyID)
	pendingInvitesLock.Unlock()

	// Tell inviter it worked
	sendResponse(player, InviteSentResponse{
		BaseResponse: newBaseResponse(ResponseInviteSent),
		Success:      true,
		Message:      "Invite sent",
	})

	// Tell invitee they got an invite
	sendResponse(friend, InviteReceivedResponse{
		BaseResponse: newBaseResponse(ResponseInviteReceived),
		LobbyID:      lobby.ID,
		LobbyName:    lobby.Name,
		InviterID:    msg.PlayerID,
		InviterName:  player.Name,
	})
}

func createCustomBotHandler(msg CreateCustomBotRequest) {
	lobbiesLock.RLock()
	lobby, ok := lobbies[msg.LobbyID]
	lobbiesLock.RUnlock()
	if !ok {
		log.Println("createCustomBotHandler: Lobby not found")
		return
	}

	lobby.Lock.Lock()

	if len(lobby.Players)+len(lobby.Bots) >= lobby.MaxPlayers {
		lobby.Lock.Unlock()

		broadcastToLobbyPlayers(lobby, BotCreationFailedResponse{
			BaseResponse: newBaseResponse(ResponseBotCreationFailed),
			Message:      "Lobby is full",
		})

		return
	}

	if lobby.BotCreating {
		lobby.Lock.Unlock()

		broadcastToLobbyPlayers(lobby, BotCreationFailedResponse{
			BaseResponse: newBaseResponse(ResponseBotCreationFailed),
			Message:      "A bot is already being created",
		})

		return
	}

	lobby.BotCreating = true
	lobby.Lock.Unlock()

	broadcastToLobbyPlayers(lobby, BotCreatingResponse{
		BaseResponse: newBaseResponse(ResponseBotCreating),
	})

	var archetype *BotArchetype

	for _, a := range botArchetypes {
		if a.ID == msg.ArchetypeID {
			copy := a
			archetype = &copy
			break
		}
	}

	if archetype == nil {
		lobby.Lock.Lock()
		lobby.BotCreating = false
		lobby.Lock.Unlock()

		broadcastToLobbyPlayers(lobby, BotCreationFailedResponse{
			BaseResponse: newBaseResponse(ResponseBotCreationFailed),
			Message:      "Invalid archetype",
		})

		return
	}

	personality := archetype.Description

	switch msg.EnergyLevel {
	case 1:
		personality += "\nVery low energy. Rarely types a lot."
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
		personality += "\nFairly chaotic and unpredictable."
	case 4:
		personality += "\nFrequently derails conversations with random energy."
	case 5:
		personality += "\nAbsolute chaos goblin."
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

	botName := generateBotName()

	bot := &Bot{
		ID:                      uuid.New().String(),
		Name:                    botName,
		SystemPromptPersonality: personality,
		MessageQueue:            make(chan BotMessage, 1),
	}

	lobby.Lock.Lock()

	if len(lobby.Players)+len(lobby.Bots) >= lobby.MaxPlayers {
		lobby.BotCreating = false
		lobby.Lock.Unlock()

		broadcastToLobbyPlayers(lobby, BotCreationFailedResponse{
			BaseResponse: newBaseResponse(ResponseBotCreationFailed),
			Message:      "Lobby is full",
		})

		return
	}

	lobby.Bots = append(lobby.Bots, bot)
	lobby.BotCreating = false
	lobby.Lock.Unlock()

	go bot.Start(lobby)

	broadcastBotJoined(lobby, bot)
	broadcastLobbyUpdate(lobby)
	broadcastLobbies()

	log.Printf(
		"Custom bot created: %s (%s)",
		bot.Name,
		archetype.DisplayName,
	)
}
