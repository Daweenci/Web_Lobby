// useGameWebSocket.ts
import { useRef, useEffect } from 'react';
import type { yourLobby, broadcastedLobby, Player, PageType, friendRequest, Friend, LobbyInvite } from './structs';
import { MessageTypes, Page } from './structs';
import { toast } from 'sonner';

interface UseWebSocketProps {
  onSetPlayer: (player: Player) => void;
  onSetLobby: (lobby: yourLobby) => void;
  onSetLobbies: (lobbies: broadcastedLobby[]) => void;
  onSetPendingFriendRequests: (pendingFriendRequests: friendRequest[]) => void;
  onSetFriendsList: React.Dispatch<React.SetStateAction<Friend[]>>;
  onSetPage: (page: PageType) => void;
  onSetPendingInvites: (pendingInvites: LobbyInvite[]) => void;
}

export default function useWebSocket({
  onSetPlayer,
  onSetLobby,
  onSetLobbies,
  onSetPendingFriendRequests,
  onSetFriendsList,
  onSetPage,
}: UseWebSocketProps) {

  const ws = useRef<WebSocket | null>(null);

  const setAuthToken = (token: string) => {
    localStorage.setItem('gameToken', token);
  };

  const getAuthToken = () => {
    return localStorage.getItem('gameToken');
  };

  const clearAuthToken = () => {
    localStorage.removeItem('gameToken');
  };

  // Close socket if tab closes
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

    // Prevent duplicate connections
    if (
      ws.current &&
      (ws.current.readyState === WebSocket.CONNECTING ||
       ws.current.readyState === WebSocket.OPEN)
    ) {
      return;
    }

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
          token: token
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
          onSetPage(Page.InLobby);
          toast('Lobby created successfully');
          break;

        case MessageTypes.ResponseJoinLobbySuccessful:
          onSetLobby(data.lobby);
          onSetPage(Page.InLobby);
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
          toast(data.player.name + " sent you a friend request");
          break;

        case MessageTypes.ResponseFriendRequestAccepted:
          toast("Friend request accepted by " + data.friend.name);
          break;

        case MessageTypes.ResponseFriendOnlineStatus: {

          onSetFriendsList(prev =>
            prev.map(f =>
              f.id === data.friend.id
                ? { ...f, isOnline: data.friend.isOnline }
                : f
            )
          );

          toast(`${data.friend.name} is now ${data.friend.isOnline ? "online" : "offline"}`);
          break;
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
        toast("Logged in on another device");
      } else {
        console.log("Connection closed");
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

  const createLobby = (
    lobbyName: string,
    maxPlayers: number,
    isPrivate: boolean,
    password: string
  ) =>
    sendMessage({
      type: MessageTypes.RequestCreateLobby,
      lobbyName,
      maxPlayers,
      isPrivate,
      password
    });

  const joinLobby = (lobbyID: string, password: string) =>
    sendMessage({
      type: MessageTypes.RequestJoinLobby,
      lobbyID,
      password
    });

  const leaveLobby = (lobbyID: string) =>
    sendMessage({
      type: MessageTypes.RequestLeaveLobby,
      lobbyID
    });

  const startGame = (lobbyID: string) =>
    sendMessage({
      type: MessageTypes.RequestStartGame,
      lobbyID
    });

  const cancelGame = (lobbyID: string) =>
    sendMessage({
      type: MessageTypes.RequestCancelGame,
      lobbyID
    });

  const logout = () => {
    clearAuthToken();
    ws.current?.close();
    onSetPlayer({} as Player);
    onSetLobby({} as yourLobby);
    onSetLobbies([]);
    onSetPage(Page.Auth);
  };

  const addFriend = (friendName: string) => {
    sendMessage({
      type: MessageTypes.RequestAddFriend,
      friendName
    });
  }

  const acceptFriendRequest = (friendID: string, acceptRequest: boolean) => {
    sendMessage({
      type: MessageTypes.RequestAcceptFriendRequest,
      friendID, //person who requested 
      acceptRequest: acceptRequest
    });
  }

  // Auto-connect if token exists
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
    getAuthToken,
    setAuthToken,
    clearAuthToken,
  };
}