import { create } from 'zustand';
import type { broadcastedLobby, ChatMessage, Friend, friendRequest, LobbyInvite, PageType, Player, TypingPlayer, yourLobby } from './structs';
import { Page } from './structs';

export interface Store {
    player: Player;
    broadcastedLobbies: broadcastedLobby[];
    lobby: yourLobby;
    pendingFriendRequests: friendRequest[];
    friendsList: Friend[];
    currentPage: PageType;
    pendingInvites: LobbyInvite[];
    chatMessages: ChatMessage[];
    typingPlayers: TypingPlayer[];
    botCreating: boolean;

    setPlayer: (player: Player) => void;
    setBroadcastedLobbies: (lobbies: broadcastedLobby[]) => void;
    setLobby: (lobby: yourLobby) => void;
    setPendingFriendRequests: (requests: friendRequest[]) => void;
    setFriendsList: (friends: Friend[]) => void;
    setPage: (page: PageType) => void;
    setPendingInvites: (invites: LobbyInvite[]) => void;
    setChatMessages: (messages: ChatMessage[]) => void;
    setTypingPlayers: (players: TypingPlayer[]) => void;

    enterLobby: (data: { lobby: yourLobby; chatMessages?: ChatMessage[] }) => void;
    resetLobbyState: (page: PageType) => void;

    addPendingFriendRequest: (request: friendRequest) => void;
    removePendingFriendRequest: (id: string) => void;

    addFriend: (friend: Friend) => void;
    updateFriendStatus: (friend: Friend) => void;

    addInvite: (invite: LobbyInvite) => void;
    removeInvite: (lobbyID: string) => void;

    addChatMessage: (msg: ChatMessage) => void;
    clearChat: () => void;

    addTypingPlayer: (player: TypingPlayer) => void;
    removeTypingPlayer: (playerID: string) => void;
    clearTypingPlayers: () => void;

    setBotCreating: (creating: boolean) => void;
    addBot: (bot: any) => void;
    removeBot: (botID: string) => void;

    removeFriend: (friendID: string) => void;
}

export const useStore = create<Store>((set) => ({
    player: {} as Player,
    broadcastedLobbies: [],
    lobby: {} as yourLobby,
    pendingFriendRequests: [],
    friendsList: [],
    currentPage: Page.Auth,
    pendingInvites: [],
    chatMessages: [],
    typingPlayers: [],
    botCreating: false,

    setPlayer: (player) => set({ player }),
    setBroadcastedLobbies: (lobbies) => set({ broadcastedLobbies: lobbies }),
    setLobby: (lobby) => set({ lobby }),
    setPendingFriendRequests: (requests) => set({ pendingFriendRequests: requests }),
    setFriendsList: (friends) => set({ friendsList: friends }),
    setPage: (page) => set({ currentPage: page }),
    setPendingInvites: (invites) => set({ pendingInvites: invites }),
    setChatMessages: (messages) => set({ chatMessages: messages }),
    setTypingPlayers: (players) => set({ typingPlayers: players }),

    enterLobby: ({
        lobby,
        chatMessages,
        }: {
        lobby: yourLobby;
        chatMessages?: ChatMessage[];
        }) =>
        set({
            lobby,
            chatMessages: chatMessages ?? [],
            typingPlayers: [],
            botCreating: false,
            currentPage: Page.InLobby,
        }),

    resetLobbyState: (page: PageType) =>
        set({
            lobby: {} as yourLobby,
            chatMessages: [],
            typingPlayers: [],
            botCreating: false,
            currentPage: page,
        }),

    addPendingFriendRequest: (request) =>
        set((state) => ({
        pendingFriendRequests: [...state.pendingFriendRequests, request],
        })),

    removePendingFriendRequest: (id) =>
        set((state) => ({
        pendingFriendRequests: state.pendingFriendRequests.filter((r) => r.id !== id),
        })),

    addFriend: (friend) =>
        set((state) => ({
        friendsList: [...state.friendsList, friend],
        })),

    updateFriendStatus: (friend) =>
        set((state) => ({
        friendsList: state.friendsList.map((f) =>
            f.id === friend.id ? { ...f, isOnline: friend.isOnline } : f
        ),
        })),

    addInvite: (invite) =>
        set((state) => ({
        pendingInvites: [...state.pendingInvites, invite],
        })),

    removeInvite: (lobbyID) =>
        set((state) => ({
        pendingInvites: state.pendingInvites.filter((i) => i.lobbyID !== lobbyID),
        })),

    addChatMessage: (msg) =>
        set((state) => ({
        chatMessages: [...state.chatMessages, msg],
        })),

    clearChat: () => set({ chatMessages: [] }),

    addTypingPlayer: (player) =>
        set((state) => {
        if (state.typingPlayers.find((p) => p.playerID === player.playerID)) {
            return state;
        }
        return {
            typingPlayers: [...state.typingPlayers, player],
        };
        }),

    removeTypingPlayer: (playerID) =>
        set((state) => ({
        typingPlayers: state.typingPlayers.filter((p) => p.playerID !== playerID),
        })),

    clearTypingPlayers: () => set({ typingPlayers: [] }),

    setBotCreating: (creating) => set({ botCreating: creating }),

    addBot: (bot) =>
        set((state) => {
        if (!state.lobby) return state;
        return {
            lobby: {
            ...state.lobby,
            bots: [...state.lobby.bots, bot],
            },
        };
        }),

    removeBot: (botID) =>
        set((state) => {
            if (!state.lobby) return state;
            return {
                lobby: {
                ...state.lobby,
                bots: state.lobby.bots.filter((b) => b.id !== botID),
                },
            };
        }),

    removeFriend: (friendID) =>
        set((state) => {
            const friendExists = state.friendsList.some((f) => f.id === friendID);
            if (!friendExists) return state;
            return {
                friendsList: state.friendsList.filter((f) => f.id !== friendID),
            };
        }),
}));