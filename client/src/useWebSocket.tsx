import { useRef, useEffect } from 'react';
import type { yourLobby, broadcastedLobby, Player, PageType, friendRequest, Friend, LobbyInvite, ChatMessage, TypingPlayer } from './structs';
import { MessageTypes, Page } from './structs';
import { toast } from 'sonner';

interface UseWebSocketProps {
  onSetPlayer: React.Dispatch<React.SetStateAction<Player>>;
  onSetLobby: React.Dispatch<React.SetStateAction<yourLobby>>;
  onSetLobbies: React.Dispatch<React.SetStateAction<broadcastedLobby[]>>;
  onSetPendingFriendRequests: React.Dispatch<React.SetStateAction<friendRequest[]>>;
  onSetFriendsList: React.Dispatch<React.SetStateAction<Friend[]>>;
  onSetPage: React.Dispatch<React.SetStateAction<PageType>>;
  onSetPendingInvites: React.Dispatch<React.SetStateAction<LobbyInvite[]>>;
  onSetChatMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
  onSetTypingPlayers: React.Dispatch<React.SetStateAction<TypingPlayer[]>>;
}

export default function useWebSocket({
  onSetPlayer,
  onSetLobby,
  onSetLobbies,
  onSetPendingFriendRequests,
  onSetFriendsList,
  onSetPage,
  onSetPendingInvites,
  onSetChatMessages,
  onSetTypingPlayers,
}: UseWebSocketProps) {

  const ws = useRef<WebSocket | null>(null);

  // Refs so toast buttons never close over stale function instances
  const joinLobbyRef = useRef<(lobbyID: string, password: string) => void>(() => {});
  const declineInviteRef = useRef<(lobbyID: string) => void>(() => {});

  const setAuthToken = (token: string) => localStorage.setItem('gameToken', token);
  const getAuthToken = () => localStorage.getItem('gameToken');
  const clearAuthToken = () => localStorage.removeItem('gameToken');

  useEffect(() => {
    const handleUnload = () => {
      if (!ws.current) return;
      ws.current.onclose = () => {};
      ws.current.close();
    };
    window.addEventListener('beforeunload', handleUnload);
    return () => {
      window.removeEventListener('beforeunload', handleUnload);
      ws.current?.close();
    };
  }, []);

  const connect = () => {
    if (
      ws.current &&
      (ws.current.readyState === WebSocket.CONNECTING ||
        ws.current.readyState === WebSocket.OPEN)
    ) return;

    if (ws.current) {
      ws.current.close();
      ws.current = null;
    }

    const token = getAuthToken();
    if (!token) {
      console.error('No auth token found. Cannot connect WebSocket.');
      onSetPage(Page.Auth);
      return;
    }

    const wsUrl = import.meta.env.REACT_APP_WS_URL || 'ws://localhost:4000/ws';
    ws.current = new WebSocket(wsUrl);

    ws.current.onopen = () => {
      if (ws.current && ws.current.readyState === WebSocket.OPEN) {
        ws.current.send(JSON.stringify({
          type: MessageTypes.RequestAuthentication,
          token: token,
        }));
      }
    };

    ws.current.onmessage = (event) => {
      const data = JSON.parse(event.data);
      console.log('WebSocket message received:', data);

      switch (data.type) {

        case MessageTypes.ResponseWelcome:
          onSetPlayer(data.player);
          onSetPage(Page.MainMenu);
          if (data.message) toast(data.message);
          if (data.lobbies) onSetLobbies(data.lobbies);
          if (data.pendingFriendRequests) onSetPendingFriendRequests(data.pendingFriendRequests);
          if (data.friendsList) onSetFriendsList(data.friendsList);
          break;

        case MessageTypes.ResponseLobbyList:
          onSetLobbies(data.lobbies);
          break;

        case MessageTypes.ResponseLobbyCreated:
          onSetLobby(data.lobby);
          onSetChatMessages([]);
          onSetTypingPlayers([]);
          onSetPage(Page.InLobby);
          toast('Lobby created successfully');
          break;

        case MessageTypes.ResponseJoinLobbySuccessful:
          onSetLobby(data.lobby);
          onSetChatMessages(data.chatHistory || []);
          onSetTypingPlayers([]);
          onSetPage(Page.InLobby);
          onSetPendingInvites(prev => prev.filter(i => i.lobbyID !== data.lobby.id));
          toast.dismiss();
          toast('Joined lobby successfully');
          break;

        case MessageTypes.ResponseJoinLobbyFailed:
          toast(data.message || 'Failed to join lobby');
          break;

        case MessageTypes.ResponseLobbyUpdated:
          onSetLobby(data.lobby);
          break;

        case MessageTypes.ResponseLobbyLeft:
          onSetLobby({} as yourLobby);
          onSetChatMessages([]);
          onSetTypingPlayers([]);
          onSetPage(Page.MainMenu);
          break;

        case MessageTypes.ResponsePendingFriendRequests:
          onSetPendingFriendRequests(data.pendingFriendRequests);
          break;

        case MessageTypes.ResponseFriendsList:
          onSetFriendsList(data.friendsList);
          break;

        case MessageTypes.ResponseFriendRequestSent:
          toast(data.message);
          break;

        case MessageTypes.ResponseFriendRequestReceived:
          onSetPendingFriendRequests(prev => [...prev, { id: data.player.id, name: data.player.name }]);
          toast(data.player.name + ' sent you a friend request');
          break;

        case MessageTypes.ResponseFriendRequestAccepted:
          onSetFriendsList(prev => [...prev, data.friend]);
          toast('Friend request accepted by ' + data.friend.name);
          break;

        case MessageTypes.ResponseFriendOnlineStatus:
          onSetFriendsList(prev =>
            prev.map(f =>
              f.id === data.friend.id
                ? { ...f, isOnline: data.friend.isOnline }
                : f
            )
          );
          toast(`${data.friend.name} is now ${data.friend.isOnline ? 'online' : 'offline'}`);
          break;

        case MessageTypes.ResponseInviteReceived: {
          const invite: LobbyInvite = {
            lobbyID: data.lobbyID,
            lobbyName: data.lobbyName,
            inviterID: data.inviterID,
            inviterName: data.inviterName,
          };
          onSetPendingInvites(prev => [...prev, invite]);

          const toastID = `invite-${data.lobbyID}`;
          toast.custom(
            (t) => (
              <div className="flex flex-col gap-2 bg-white border border-gray-200 rounded-xl shadow-lg px-4 py-3 min-w-[280px]">
                <p className="text-sm text-gray-800">
                  <strong>{data.inviterName}</strong> invited you to <strong>{data.lobbyName}</strong>
                </p>
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      joinLobbyRef.current(data.lobbyID, '');
                      toast.dismiss(t);
                    }}
                    className="flex-1 bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium px-3 py-1.5 rounded-lg transition"
                  >
                    Join
                  </button>
                  <button
                    onClick={() => {
                      declineInviteRef.current(data.lobbyID);
                      toast.dismiss(t);
                    }}
                    className="flex-1 bg-gray-100 hover:bg-gray-200 text-gray-700 text-sm font-medium px-3 py-1.5 rounded-lg transition"
                  >
                    Decline
                  </button>
                </div>
              </div>
            ),
            { id: toastID, duration: 15000 }
          );
          break;
        }

        case MessageTypes.ResponseInviteSent:
          toast(data.message);
          break;

        case MessageTypes.ResponseInviteDeclined:
          onSetPendingInvites(prev => prev.filter(i => i.lobbyID !== data.lobbyID));
          toast.dismiss(`invite-${data.lobbyID}`);
          break;

        case MessageTypes.ResponseBotJoined:
          onSetLobby(prev => ({
            ...prev,
            bots: [...(prev.bots || []), data.bot],
          }));
          break;

        case MessageTypes.ResponseBotLeft:
          onSetLobby(prev => ({
            ...prev,
            bots: (prev.bots || []).filter(b => b.id !== data.botID),
          }));
          break;

        case MessageTypes.ResponseLobbyChatMessage:
          onSetChatMessages(prev => [...prev, {
            senderID: data.senderID,
            senderName: data.senderName,
            content: data.content,
            isBot: data.isBot,
          }]);
          onSetTypingPlayers(prev => prev.filter(p => p.playerID !== data.senderID));
          break;

        case MessageTypes.ResponsePlayerTyping:
          if (data.isTyping) {
            onSetTypingPlayers(prev => {
              if (prev.find(p => p.playerID === data.playerID)) return prev;
              return [...prev, { playerID: data.playerID, playerName: data.playerName }];
            });
          } else {
            onSetTypingPlayers(prev => prev.filter(p => p.playerID !== data.playerID));
          }
          break;

        case MessageTypes.ResponseError:
          toast(data.error || 'An error occurred');
          break;

        default:
          console.warn('Unknown message type:', data.type);
      }
    };

    ws.current.onerror = (err) => {
      console.error('WebSocket error:', err);
      toast('Connection error');
    };

    ws.current.onclose = (event) => {
      console.log('WebSocket closed', event.code, event.reason);
      ws.current = null;
      if (event.code === 1008) {
        onSetPlayer({} as Player);
        onSetLobby({} as yourLobby);
        onSetLobbies([]);
        onSetPage(Page.Auth);
        toast('Logged in on another device');
      }
    };
  };

  const sendMessage = (msg: any) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(msg));
    } else {
      console.error('WebSocket is not connected');
      toast('Connection lost, please refresh');
    }
  };

  const createLobby = (lobbyName: string, maxPlayers: number, isPrivate: boolean, password: string) =>
    sendMessage({ type: MessageTypes.RequestCreateLobby, lobbyName, maxPlayers, isPrivate, password });

  const joinLobby = (lobbyID: string, password: string) =>
    sendMessage({ type: MessageTypes.RequestJoinLobby, lobbyID, password });

  const leaveLobby = (lobbyID: string) =>
    sendMessage({ type: MessageTypes.RequestLeaveLobby, lobbyID });

  const startGame = (lobbyID: string) =>
    sendMessage({ type: MessageTypes.RequestStartGame, lobbyID });

  const cancelGame = (lobbyID: string) =>
    sendMessage({ type: MessageTypes.RequestCancelGame, lobbyID });

  const addFriend = (friendName: string) =>
    sendMessage({ type: MessageTypes.RequestAddFriend, friendName });

  const acceptFriendRequest = (friendID: string, acceptRequest: boolean) =>
    sendMessage({ type: MessageTypes.RequestAcceptFriendRequest, friendID, acceptRequest });

  const inviteToLobby = (lobbyID: string, friendID: string) =>
    sendMessage({ type: MessageTypes.RequestInviteToLobby, lobbyID, friendID });

  const declineInvite = (lobbyID: string) =>
    sendMessage({ type: MessageTypes.RequestDeclineInvite, lobbyID });

  const addBotToLobby = (lobbyID: string) =>
    sendMessage({ type: MessageTypes.RequestAddBotToLobby, lobbyID });

  const removeBotFromLobby = (lobbyID: string, botID: string) =>
    sendMessage({ type: MessageTypes.RequestRemoveBotFromLobby, lobbyID, botID });

  const sendLobbyChatMessage = (lobbyID: string, message: string) =>
    sendMessage({ type: MessageTypes.RequestSendLobbyChatMessage, lobbyID, message });

  const startedTyping = (lobbyID: string) =>
    sendMessage({ type: MessageTypes.RequestStartedTyping, lobbyID, isTyping: true });

  const stoppedTyping = (lobbyID: string) =>
    sendMessage({ type: MessageTypes.RequestStartedTyping, lobbyID, isTyping: false });

  // Keep refs in sync so toast buttons always call the latest version
  joinLobbyRef.current = joinLobby;
  declineInviteRef.current = declineInvite;

  const logout = () => {
    clearAuthToken();
    ws.current?.close();
    onSetPlayer({} as Player);
    onSetLobby({} as yourLobby);
    onSetLobbies([]);
    onSetPage(Page.Auth);
  };

  useEffect(() => {
    const token = getAuthToken();
    if (token) connect();
  }, []);

  return {
    connect,
    logout,
    addFriend,
    acceptFriendRequest,
    createLobby,
    joinLobby,
    leaveLobby,
    startGame,
    cancelGame,
    inviteToLobby,
    declineInvite,
    addBotToLobby,
    removeBotFromLobby,
    sendLobbyChatMessage,
    startedTyping,
    stoppedTyping,
    getAuthToken,
    setAuthToken,
    clearAuthToken,
  };
}