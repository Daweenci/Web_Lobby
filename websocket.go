package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	lobbies     = make(map[string]*Lobby)
	lobbiesLock sync.RWMutex

	activePlayers     = make(map[string]*Player)
	activePlayersLock sync.RWMutex

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

func sendErrorToPlayer(player *Player, errorMsg string) {
	sendResponse(player, ErrorResponse{
		BaseResponse: newBaseResponse(ResponseError),
		Error:        errorMsg,
	})
}

func sendErrorToConn(conn *websocket.Conn, errorMsg string) {
	err := conn.WriteJSON(ErrorResponse{
		BaseResponse: newBaseResponse(ResponseError),
		Error:        errorMsg,
	})
	if err != nil {
		log.Printf("Failed to send error message to connection %s: %v", conn.RemoteAddr(), err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("NEW WEBSOCKET CONNECTION from %s", r.RemoteAddr)
	// Upgrade to WebSocket immediately (no auth check yet)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}

	// Create temporary player for initial connection
	var player *Player
	authenticated := false

	defer func() {
		conn.Close()

		if player != nil {
			activePlayersLock.RLock()
			current, exists := activePlayers[player.ID]
			activePlayersLock.RUnlock()

			if exists && current.Conn == conn {
				disconnectPlayer(player.ID)
				broadcastLobbies()
				pingAllFriendsOnlineStatusHandler(player.ID, false) //isOnline = false
			}

		}
	}()

	// Wait for authentication message
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var base struct {
			Type  MessageType `json:"type"`
			Token string      `json:"token,omitempty"`
		}
		if err := json.Unmarshal(msgBytes, &base); err != nil {
			sendErrorToConn(conn, "Invalid message format")
			continue
		}

		// If not authenticated yet, only accept authenticate messages
		if !authenticated {
			if base.Type != RequestAuthentication || base.Token == "" {
				sendErrorToConn(conn, "Authentication required")
				continue
			}

			// Validate JWT token
			playerID, err := parseJWT(base.Token)
			if err != nil {
				sendErrorToConn(conn, "Invalid or expired token")
				continue
			}

			// Get player from database
			dbPlayer, err := getPlayerByID(playerID)
			if err != nil {
				sendErrorToConn(conn, "Player not found")
				continue
			}

			// Create authenticated player
			player = &Player{
				ID:   dbPlayer.ID,
				Name: dbPlayer.Username,
				Conn: conn,
				Send: make(chan []byte, 256),
			}

			// Add to active players
			var oldPlayer *Player
			activePlayersLock.RLock()
			if p, exists := activePlayers[player.ID]; exists {
				oldPlayer = p
			}
			activePlayersLock.RUnlock()

			if oldPlayer != nil {
				log.Printf("Player %s already connected, disconnecting old connection", player.ID)

				oldPlayer.Conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(1008, "Duplicate login"),
					time.Now().Add(time.Second),
				)

				disconnectPlayer(oldPlayer.ID)
			}
			activePlayersLock.Lock()
			activePlayers[player.ID] = player
			activePlayersLock.Unlock()

			go player.writePump()

			authenticated = true

			// Send welcome message
			welcomeResponse := WelcomeResponse{
				BaseResponse: newBaseResponse(ResponseWelcome),
				Player: PlayerDTO{
					ID:   player.ID,
					Name: player.Name,
				},
				Message:               "Welcome back, " + player.Name + "!",
				Lobbies:               getLobbiesList(),
				PendingFriendRequests: getPendingFriendRequests(player.ID),
				FriendsList:           getFriendsWithOnlineStatus(player.ID),
			}
			sendResponse(player, welcomeResponse)
			log.Printf("Player %s authenticated successfully", player.ID)
			pingAllFriendsOnlineStatusHandler(player.ID, true) //isOnline = true
			continue
		}

		// Handle authenticated messages
		switch base.Type {
		case RequestJoinLobby:
			var msg JoinLobbyRequest
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid join_lobby message")
				continue
			}
			msg.PlayerID = player.ID
			msg.Name = player.Name
			joinLobbyHandler(msg)

		case RequestLeaveLobby:
			var msg LeaveLobbyRequest
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid leave_lobby message")
				continue
			}
			msg.PlayerID = player.ID
			leaveLobbyHandler(msg)

		case RequestCreateLobby:
			var msg CreateLobbyRequest
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid create_lobby message")
				continue
			}
			msg.PlayerID = player.ID
			msg.PlayerName = player.Name
			createLobbyHandler(msg)

		case RequestStartGame:
			var msg StartGame
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid start_game message")
				continue
			}
			msg.PlayerID = player.ID
			startGameHandler(msg)

		case RequestCancelGame:
			var msg CancelGame
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid cancel_game message")
				continue
			}
			msg.PlayerID = player.ID
			cancelGameHandler(msg)

		case RequestAddFriend:
			var msg AddFriendRequest
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid add_friend message")
				continue
			}
			msg.PlayerID = player.ID
			sendFriendRequestHandler(msg)

		case RequestAcceptFriendRequest:
			var msg AcceptFriendRequestRequest
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid accept_friend_request message")
				continue
			}
			msg.PlayerID = player.ID
			acceptFriendRequestHandler(msg)

		case RequestSendLobbyChatMessage:
			var msg SendLobbyChatMessageRequest
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid send_lobby_chat_message message")
				continue
			}
			msg.PlayerID = player.ID
			sendLobbyChatMessageHandler(msg)

		case RequestStartedTyping:
			var msg TypingRequest
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid started_typing message")
				continue
			}
			msg.PlayerID = player.ID
			typingHandler(msg)

		case RequestAddBotToLobby:
			var msg AddBotToLobbyRequest
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid add_bot_to_lobby message")
				continue
			}
			addBotToLobbyHandler(msg)

		case RequestRemoveBotFromLobby:
			var msg RemoveBotFromLobbyRequest
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendErrorToPlayer(player, "Invalid remove_bot_from_lobby message")
				continue
			}
			removeBotFromLobbyHandler(msg)

		default:
			sendErrorToPlayer(player, "Unknown message type")
		}
	}

}
