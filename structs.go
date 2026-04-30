package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type MessageType string

const (
	Online  string = "online"
	Offline string = "offline"

	RequestAuthentication       MessageType = "authenticate"
	RequestLogin                MessageType = "login"
	RequestRegister             MessageType = "register"
	RequestCreateLobby          MessageType = "create_lobby"
	RequestJoinLobby            MessageType = "join_lobby"
	RequestLeaveLobby           MessageType = "leave_lobby"
	RequestStartGame            MessageType = "start_game"
	RequestCancelGame           MessageType = "cancel_game"
	RequestAddFriend            MessageType = "add_friend"
	RequestAcceptFriendRequest  MessageType = "accept_friend_request"
	RequestInviteToLobby        MessageType = "invite_to_lobby"
	RequestDeclineInvite        MessageType = "decline_invite"
	RequestAddBotToLobby        MessageType = "add_bot_to_lobby"
	RequestRemoveBotFromLobby   MessageType = "remove_bot_from_lobby"
	RequestAddBotFriend         MessageType = "add_bot_friend"    //TODO
	RequestRemoveBotFriend      MessageType = "remove_bot_friend" //TODO
	RequestInviteBotFriend      MessageType = "invite_bot_friend" //TODO
	RequestStartedTyping        MessageType = "started_typing"
	RequestSendLobbyChatMessage MessageType = "send_lobby_chat_message"

	ResponseWelcome               MessageType = "welcome"
	ResponseLoginSuccessful       MessageType = "login_successful"
	ResponseLoginFailed           MessageType = "login_failed"
	ResponseRegisterSuccessful    MessageType = "register_successful"
	ResponseRegisterFailed        MessageType = "register_failed"
	ResponseLobbyCreated          MessageType = "lobby_created"
	ResponseLobbyList             MessageType = "lobby_list"
	ResponseLobbyUpdated          MessageType = "lobby_updated"
	ResponseJoinLobbySuccessful   MessageType = "join_lobby_successful"
	ResponseJoinLobbyFailed       MessageType = "join_lobby_failed"
	ResponseLobbyLeft             MessageType = "lobby_left"
	ResponseFriendRequestSent     MessageType = "friend_request_sent"
	ResponsePendingFriendRequests MessageType = "pending_friend_requests"
	ResponseFriendRequestReceived MessageType = "friend_request_received"
	ResponseFriendRequestAccepted MessageType = "friend_request_accepted"
	ResponseFriendOnlineStatus    MessageType = "friend_online_status"
	ResponseInviteReceived        MessageType = "invite_received"    //TODO
	ResponseInviteSent            MessageType = "invite_sent"        //TODO
	ResponseBotJoined             MessageType = "bot_joined"         //TODO
	ResponseBotLeft               MessageType = "bot_left"           //TODO
	ResponseBotFriendAdded        MessageType = "bot_friend_added"   //TODO
	ResponseBotFriendRemoved      MessageType = "bot_friend_removed" //TODO
	ResponsePlayerTyping          MessageType = "player_typing"
	ResponseLobbyChatMessage      MessageType = "lobby_chat_message"
	ResponseFriendsList           MessageType = "friends_list"
	ResponseError                 MessageType = "error"
)

type Bot struct {
	ID                      string
	Name                    string
	SystemPromptPersonality string
	MessageQueue            chan BotMessage
}

type BotDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type BotMessage struct { //To the bot, not from the bot
	LobbyID string
}

type LobbyChatMessage struct {
	SenderID   string `json:"senderID"`
	SenderName string `json:"senderName"`
	IsBot      bool   `json:"isBot"`
	Message    string `json:"message"`
}

type Player struct {
	ID   string
	Name string
	Conn *websocket.Conn
	Send chan []byte
}

type PlayerDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FriendDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsOnline bool   `json:"isOnline"`
}

type PlayerStarted struct {
	ID string `json:"playerID"`
}

type LoginRequest struct {
	Type     MessageType `json:"type"`
	Name     string      `json:"name"`
	Password string      `json:"password"`
}

type RegisterRequest struct {
	Type     MessageType `json:"type"`
	Name     string      `json:"name"`
	Password string      `json:"password"`
}

type JoinLobbyRequest struct {
	Type     MessageType `json:"type"`
	LobbyID  string      `json:"lobbyID"`
	PlayerID string      `json:"playerID"`
	Name     string      `json:"name"`
	Password string      `json:"password"`
}

type CreateLobbyRequest struct {
	Type       MessageType `json:"type"`
	LobbyName  string      `json:"lobbyName"`
	MaxPlayers int         `json:"maxPlayers"`
	IsPrivate  bool        `json:"isPrivate"`
	Password   string      `json:"password"`
	PlayerID   string      `json:"playerID"`
	PlayerName string      `json:"playerName"`
}

type LeaveLobbyRequest struct {
	Type     MessageType `json:"type"`
	LobbyID  string      `json:"lobbyID"`
	PlayerID string      `json:"playerID"`
}

type AddFriendRequest struct {
	Type       MessageType `json:"type"`
	FriendName string      `json:"friendName"`
	PlayerID   string      `json:"playerID"`
}

type GetPendingFriendRequestsRequest struct {
	Type     MessageType `json:"type"`
	PlayerID string      `json:"playerID"`
}

type AcceptFriendRequestRequest struct {
	Type          MessageType `json:"type"`
	FriendID      string      `json:"friendID"` //person that requested in the first place
	AcceptRequest bool        `json:"acceptRequest"`
	PlayerID      string      `json:"playerID"`
}

type SendLobbyChatMessageRequest struct {
	Type     MessageType `json:"type"`
	LobbyID  string      `json:"lobbyID"`
	PlayerID string      `json:"playerID"`
	Message  string      `json:"message"`
}

type InviteToLobbyRequest struct {
	Type     MessageType `json:"type"`
	LobbyID  string      `json:"lobbyID"`
	PlayerID string      `json:"playerID"`
	FriendID string      `json:"friendID"`
}

type DeclineInviteRequest struct {
	Type     MessageType `json:"type"`
	LobbyID  string      `json:"lobbyID"`
	PlayerID string      `json:"playerID"`
}

type TypingRequest struct {
	Type     MessageType `json:"type"`
	LobbyID  string      `json:"lobbyID"`
	PlayerID string      `json:"playerID"`
}

type StartGame struct { //TODO: Why StartGame not response or request?
	Type     MessageType `json:"type"`
	LobbyID  string      `json:"lobbyID"`
	PlayerID string      `json:"playerID"`
}

type CancelGame struct { //TODO: Why CancelGame not response or request?
	Type     MessageType `json:"type"`
	LobbyID  string      `json:"lobbyID"`
	PlayerID string      `json:"playerID"`
}

type Lobby struct {
	ID          string
	Name        string
	MaxPlayers  int
	IsPrivate   bool
	Password    string
	Players     []*Player
	Bots        []*Bot
	GameStart   []PlayerStarted
	ChatHistory []LobbyChatMessage
	Lock        sync.RWMutex
}

type Response interface {
	GetType() MessageType
}

type BaseResponse struct {
	Type MessageType `json:"type"`
}

func newBaseResponse(t MessageType) BaseResponse {
	return BaseResponse{Type: t}
}

func (r BaseResponse) GetType() MessageType {
	return r.Type
}

type WelcomeResponse struct {
	BaseResponse
	Player                PlayerDTO   `json:"player"`
	Message               string      `json:"message"`
	Lobbies               []LobbyDTO  `json:"lobbies"`
	PendingFriendRequests []PlayerDTO `json:"pendingFriendRequests"`
	FriendsList           []FriendDTO `json:"friendsList"`
}

type LobbyDTO struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	MaxPlayers int             `json:"maxPlayers"`
	IsPrivate  bool            `json:"isPrivate"`
	Players    []PlayerDTO     `json:"players"`
	Bots       []BotDTO        `json:"bots"`
	GameStart  []PlayerStarted `json:"gameStart"`
}

type LobbyUpdatedResponse struct {
	BaseResponse
	Lobby LobbyDTO `json:"lobby"`
}

type LobbiesUpdateResponse struct {
	BaseResponse
	Lobbies []LobbyDTO `json:"lobbies"`
}

type LobbyJoinFailedResponse struct {
	BaseResponse
	Message string `json:"message"`
}

type AddBotToLobbyRequest struct {
	Type    MessageType `json:"type"`
	LobbyID string      `json:"lobbyID"`
}

type RemoveBotFromLobbyRequest struct {
	Type    MessageType `json:"type"`
	LobbyID string      `json:"lobbyID"`
	BotID   string      `json:"botID"`
}

type SuccessfulJoinLobbyResponse struct {
	BaseResponse
	Lobby LobbyDTO `json:"lobby"`
}

type CreateLobbyResponse struct {
	BaseResponse
	Lobby LobbyDTO `json:"lobby"`
}

type LobbyLeftResponse struct {
	BaseResponse
}

type FriendRequestSentResponse struct {
	BaseResponse
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type PendingFriendRequestsResponse struct {
	BaseResponse
	PendingFriendRequests []PlayerDTO `json:"pendingFriendRequests"`
}

type FriendRequestReceivedResponse struct {
	BaseResponse
	Player PlayerDTO `json:"player"`
}

type FriendRequestAcceptedResponse struct {
	BaseResponse
	Friend FriendDTO `json:"friend"`
}

type FriendsListResponse struct {
	BaseResponse
	FriendsList []FriendDTO `json:"friendsList"`
}

type FriendOnlineStatusResponse struct {
	BaseResponse
	Friend FriendDTO `json:"friend"`
}

type BotJoinedResponse struct {
	BaseResponse
	Bot BotDTO `json:"bot"`
}

type BotLeftResponse struct {
	BaseResponse
	BotID string `json:"botID"`
}

type LobbyChatMessageResponse struct {
	BaseResponse
	LobbyID    string `json:"lobbyID"`
	SenderName string `json:"senderName"`
	SenderID   string `json:"senderID"`
	IsBot      bool   `json:"isBot"`
	Content    string `json:"content"`
}

type PlayerTypingResponse struct {
	BaseResponse
	PlayerID   string `json:"playerID"`
	PlayerName string `json:"playerName"`
	IsTyping   bool   `json:"isTyping"`
}

type InviteSentResponse struct {
	BaseResponse
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type InviteReceivedResponse struct {
	BaseResponse
	LobbyID     string `json:"lobbyID"`
	LobbyName   string `json:"lobbyName"`
	InviterID   string `json:"inviterID"`
	InviterName string `json:"inviterName"`
}

type ErrorResponse struct {
	BaseResponse
	Error string `json:"error"`
}
