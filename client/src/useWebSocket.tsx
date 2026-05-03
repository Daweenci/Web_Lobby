import { useRef, useEffect } from 'react';
import type { yourLobby, broadcastedLobby, Player, PageType, friendRequest, Friend, LobbyInvite, ChatMessage, TypingPlayer } from './structs';
import { MessageTypes, Page } from './structs';
import { toast } from 'sonner';
import { useStore } from './store';

interface UseWebSocketProps {
}

export default function useWebSocket() {

  const ws = useRef<WebSocket | null>(null);
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
      useStore.getState().setPage(Page.Auth);
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
          useStore.getState().setPlayer(data.player);
          useStore.getState().setPage(Page.MainMenu);
          if (data.message) toast(data.message);
          if (data.lobbies) useStore.getState().setBroadcastedLobbies(data.lobbies);
          if (data.pendingFriendRequests) useStore.getState().setPendingFriendRequests(data.pendingFriendRequests);
          if (data.friendsList) useStore.getState().setFriendsList(data.friendsList);
          break;

        case MessageTypes.ResponseLobbyList:
          useStore.getState().setBroadcastedLobbies(data.lobbies);
          break;

        case MessageTypes.ResponseLobbyCreated:
          useStore.getState().setLobby(data.lobby);
          useStore.getState().setChatMessages([]);
          useStore.getState().setTypingPlayers([]);
          useStore.getState().setPage(Page.InLobby);
          toast('Lobby created successfully');
          break;

        case MessageTypes.ResponseJoinLobbySuccessful:
          useStore.getState().setLobby(data.lobby);
          useStore.getState().setChatMessages(data.chatHistory || []);
          useStore.getState().setTypingPlayers([]);
          useStore.getState().setPage(Page.InLobby);
          useStore.getState().addInvite(data.invite);
          toast.dismiss();
          toast('Joined lobby successfully');
          break;

        case MessageTypes.ResponseJoinLobbyFailed:
          toast(data.message || 'Failed to join lobby');
          break;

        case MessageTypes.ResponseLobbyUpdated:
          useStore.getState().setLobby(data.lobby);
          break;

        case MessageTypes.ResponseLobbyLeft:
          useStore.getState().setLobby({} as yourLobby);
          useStore.getState().setChatMessages([]);
          useStore.getState().setTypingPlayers([]);
          useStore.getState().setPage(Page.MainMenu);
          break;

        case MessageTypes.ResponsePendingFriendRequests:
          useStore.getState().setPendingFriendRequests(data.pendingFriendRequests);
          break;

        case MessageTypes.ResponseFriendsList:
          useStore.getState().setFriendsList(data.friendsList);
          break;

        case MessageTypes.ResponseFriendRequestSent:
          toast(data.message);
          break;

        case MessageTypes.ResponseFriendRequestReceived: {
          const request = { id: data.player.id, name: data.player.name };
          useStore.getState().addPendingFriendRequest(request);

          const toastID = `friend-request-${data.player.id}`;

          toast.custom( //TODO: extract to its own file
            () => (
              <div className="flex flex-col gap-2 bg-white border border-gray-200 rounded-xl shadow-lg px-4 py-3 min-w-[280px]">
                <p className="text-sm text-gray-800">
                  <strong>{data.player.name}</strong> sent you a friend request
                </p>
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      acceptFriendRequest(data.player.id, true);
                      toast.dismiss(toastID);
                    }}
                    className="flex-1 bg-green-500 hover:bg-green-600 text-white text-sm font-medium px-3 py-1.5 rounded-lg transition"
                  >
                    Accept
                  </button>
                  <button
                    onClick={() => {
                      acceptFriendRequest(data.player.id, false);
                      toast.dismiss(toastID);
                    }}
                    className="flex-1 bg-gray-100 hover:bg-gray-200 text-gray-700 text-sm font-medium px-3 py-1.5 rounded-lg transition"
                  >
                    Decline
                  </button>
                </div>
              </div>
            ),
            { id: toastID, duration: Infinity }
          );
          break;
        }

        case MessageTypes.ResponseFriendRequestAccepted:
          useStore.getState().addFriend(data.friend);
          toast('Friend request accepted by ' + data.friend.name);
          break;

        case MessageTypes.ResponseFriendOnlineStatus:
          useStore.getState().updateFriendStatus(data.friend);
          toast(`${data.friend.name} is now ${data.friend.isOnline ? 'online' : 'offline'}`);
          break;

        case MessageTypes.ResponseInviteReceived: {
          const invite: LobbyInvite = {
            lobbyID: data.lobbyID,
            lobbyName: data.lobbyName,
            inviterID: data.inviterID,
            inviterName: data.inviterName,
          };
          useStore.getState().addInvite(invite);

          const toastID = `invite-${data.lobbyID}`;

          toast.custom( //TODO: extract to its own file
            (t) => (
              <div className="flex flex-col gap-2 bg-white border border-gray-200 rounded-xl shadow-lg px-4 py-3 min-w-[280px]">
                <p className="text-sm text-gray-800">
                  <strong>{data.inviterName}</strong> invited you to <strong>{data.lobbyName}</strong>
                </p>
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      joinLobbyRef.current(data.lobbyID, '');
                      toast.dismiss(toastID);
                    }}
                    className="flex-1 bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium px-3 py-1.5 rounded-lg transition"
                  >
                    Join
                  </button>
                  <button
                    onClick={() => {
                      declineInviteRef.current(data.lobbyID);
                      toast.dismiss(toastID);
                    }}
                    className="flex-1 bg-gray-100 hover:bg-gray-200 text-gray-700 text-sm font-medium px-3 py-1.5 rounded-lg transition"
                  >
                    Decline
                  </button>
                </div>
              </div>
            ),
            { id: toastID, duration: 4000 }
          );
          break;
        }

        case MessageTypes.ResponseInviteSent:
          toast(data.message);
          break;

        case MessageTypes.ResponseInviteDeclined:
          useStore.getState().removeInvite(data.lobbyID);
          toast.dismiss(`invite-${data.lobbyID}`);
          break;

        case MessageTypes.ResponseBotJoined:
          useStore.getState().addBot(data.bot);
          break;

        case MessageTypes.ResponseBotLeft:
          useStore.getState().removeBot(data.botID);
          break;

        case MessageTypes.ResponseLobbyChatMessage:
          useStore.getState().addChatMessage(data.message);
          useStore.getState().removeTypingPlayer(data.message.senderID);
          break;

        case MessageTypes.ResponsePlayerTyping:
          if (data.isTyping) {
            useStore.getState().addTypingPlayer({ playerID: data.playerID, playerName: data.playerName });
          } else {
            useStore.getState().removeTypingPlayer(data.playerID);
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
        useStore.getState().setPlayer({} as Player);
        useStore.getState().setLobby({} as yourLobby);
        useStore.getState().setBroadcastedLobbies([]);
        useStore.getState().setPage(Page.Auth);
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

  const sendLobbyChatMessage = (lobbyID: string, content: string) =>
    sendMessage({ type: MessageTypes.RequestSendLobbyChatMessage, lobbyID, content });

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
    useStore.getState().setPlayer({} as Player);
    useStore.getState().setLobby({} as yourLobby);
    useStore.getState().setBroadcastedLobbies([]);
    useStore.getState().setPage(Page.Auth);
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