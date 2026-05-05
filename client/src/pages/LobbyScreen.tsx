import React, { useState, useRef, useEffect, useCallback } from 'react';
import type { Friend } from '@/structs';
import inviteIcon from "@/assets/invite.svg";
import { useStore, type Store } from '@/store';
import { useShallow } from 'zustand/react/shallow';
import UserProfile from '@/components/UserProfile';

type LobbyScreenProps = {
  startGame: () => void;
  cancelGame: () => void;
  leaveLobby: () => void;
  inviteToLobby: (friendID: string) => void;
  addBotToLobby: () => void;
  removeBotFromLobby: (botID: string) => void;
  sendLobbyChatMessage: (content: string) => void;
  startedTyping: () => void;
  stoppedTyping: () => void;
  logout: () => void;
  addFriend: (friendName: string) => void;
  acceptFriendRequest: (friendID: string, accept: boolean) => void;
  declineInvite: (lobbyID: string) => void;
  joinLobby: (lobbyID: string, joinPassword: string) => void;
};

export default function LobbyScreen({
  startGame,
  cancelGame,
  leaveLobby,
  inviteToLobby,
  addBotToLobby,
  removeBotFromLobby,
  sendLobbyChatMessage,
  startedTyping,
  stoppedTyping,
  logout,
  addFriend,
  acceptFriendRequest,
  declineInvite,
  joinLobby,
}: LobbyScreenProps) {
  const { lobby, currentPlayer, chatMessages, typingPlayers, botCreating, friendsList } = useStore(
    useShallow((state: Store) => ({
      lobby: state.lobby,
      currentPlayer: state.player,
      chatMessages: state.chatMessages,
      typingPlayers: state.typingPlayers,
      botCreating: state.botCreating,
      friendsList: state.friendsList,
    }))
  );
  const [gameStarting, setGameStarting] = useState(false);
  const [showFriends, setShowFriends] = useState(false);
  const [chatInput, setChatInput] = useState('');
  const chatBottomRef = useRef<HTMLDivElement>(null);
  const typingTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isTypingRef = useRef(false);

  // Scroll to bottom when new messages arrive, not sure wether to keep
  useEffect(() => {
    chatBottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [chatMessages]);

  const handleToggleGameStart = () => {
    if (gameStarting) {
      cancelGame();
      setGameStarting(false);
    } else {
      startGame();
      setGameStarting(true);
    }
  };

  const handleSendMessage = () => {
    const trimmed = chatInput.trim();
    if (!trimmed) return;
    sendLobbyChatMessage(trimmed);
    setChatInput('');
    // Clear typing state immediately on send
    if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
    if (isTypingRef.current) {
      stoppedTyping();
      isTypingRef.current = false;
    }
  };

  const handleChatKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') handleSendMessage();
  };

  const handleChatInput = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setChatInput(e.target.value);

    // Send startedTyping if not already typing
    if (!isTypingRef.current) {
      startedTyping();
      isTypingRef.current = true;
    }

    // Reset the debounce timer
    if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
    typingTimeoutRef.current = setTimeout(() => {
      stoppedTyping();
      isTypingRef.current = false;
    }, 3000);
  }, [startedTyping, stoppedTyping]);

  // Clean up typing timer on unmount
  useEffect(() => {
    return () => {
      if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
    };
  }, []);

  const totalOccupied = (lobby.players?.length ?? 0) + (lobby.bots?.length ?? 0);

  const typingText = typingPlayers.length === 0
    ? null
    : typingPlayers.length === 1
      ? `${typingPlayers[0].playerName} is typing...`
      : typingPlayers.length === 2
        ? `${typingPlayers[0].playerName} and ${typingPlayers[1].playerName} are typing...`
        : 'Several people are typing...';

  return (
    <div className="flex items-center justify-center min-h-screen p-4">
      <div className="absolute top-4 right-6">
        <UserProfile
          onLogout={logout}
          onAddFriend={addFriend}
          onAcceptFriendRequest={acceptFriendRequest}
          onDeclineInvite={declineInvite}
          onJoinLobby={(lobbyID) => joinLobby(lobbyID, '')}
        />
      </div>
      <div className="flex gap-6 w-full max-w-5xl">

        {/* Left panel — lobby info */}
        <div className="flex flex-col gap-4 w-64 shrink-0">
          <div className="border-2 border-gray-300 rounded-2xl p-4 flex flex-col gap-3">
            <h2 className="text-xl font-bold">{lobby.name}</h2>
            <p className="text-sm text-gray-500">
              {totalOccupied}/{lobby.maxPlayers} players
            </p>

            {/* Player + bot list */}
            <ul className="flex flex-col gap-1">
              {lobby.players?.map((player) => (
                <li key={player.id} className="flex items-center gap-2 text-sm">
                  <span className="w-2 h-2 rounded-full bg-green-500 shrink-0" />
                  <span className={player.id === currentPlayer.id ? 'font-semibold' : ''}>
                    {player.name}
                  </span>
                </li>
              ))}
              {lobby.bots?.map((bot) => (
                <li key={bot.id} className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                        <span className="w-2 h-2 rounded-full bg-purple-400 shrink-0" />
                        <span>{bot.name}</span>
                    </div>
                    <button
                        onClick={() => removeBotFromLobby(bot.id)}
                        className="text-gray-400 hover:text-red-500 transition text-xs ml-2"
                        title="Remove bot"
                    >
                        ✕
                    </button>
                </li>
            ))}
            {botCreating && (
                <li className="flex items-center gap-2 text-sm text-purple-400">
                    <svg className="w-4 h-4 animate-spin shrink-0" viewBox="0 0 24 24" fill="none">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z" />
                    </svg>
                    <span>Adding bot...</span>
                </li>
            )}
            </ul>

            {lobby.password && (
              <p className="text-xs text-gray-400">
                Password: <span className="font-mono">{lobby.password}</span>
              </p>
            )}
          </div>

          {/* Actions */}
          <div className="flex flex-col gap-2">
            {/* Invite friends */}
            <div className="relative">
              <button
                onClick={() => setShowFriends(prev => !prev)}
                className="w-full flex items-center justify-center gap-2 border-2 border-gray-300 rounded-xl px-3 py-2 hover:bg-gray-100 transition text-sm"
              >
                <span>Invite</span>
                <img src={inviteIcon} alt="Invite" className="w-5 h-5" />
              </button>
              {showFriends && (
                <div className="absolute left-0 top-full mt-1 w-full bg-white border border-gray-200 rounded-xl shadow-lg z-10 overflow-hidden">
                  <FriendsList
                    friendsList={friendsList}
                    onInvite={(friendID) => {
                      inviteToLobby(friendID);
                      setShowFriends(false);
                    }}
                  />
                </div>
              )}
            </div>

            {/* Add bot */}
            <button
                onClick={addBotToLobby}
                disabled={botCreating}
                className="w-full border-2 border-purple-300 text-purple-600 rounded-xl px-3 py-2 hover:bg-purple-50 transition text-sm disabled:opacity-50 disabled:cursor-not-allowed"
            >
                + Add Bot
            </button>

            {/* Start / Cancel */}
            <button
              onClick={handleToggleGameStart}
              className={`w-full rounded-xl px-3 py-2 text-white text-sm transition ${
                gameStarting
                  ? 'bg-orange-500 hover:bg-orange-600'
                  : 'bg-blue-500 hover:bg-blue-600'
              }`}
            >
              {gameStarting ? 'Cancel Start' : 'Start Game'} ({lobby.gameStart?.length ?? 0}/{lobby.maxPlayers})
            </button>

            {/* Leave */}
            <button
              onClick={leaveLobby}
              className="w-full bg-red-500 text-white rounded-xl px-3 py-2 hover:bg-red-600 transition text-sm"
            >
              Leave Lobby
            </button>
          </div>
        </div>

        {/* Right panel — chat */}
        <div className="flex flex-col flex-1 border-2 border-gray-300 rounded-2xl overflow-hidden">
          {/* Message list */}
          <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-2 min-h-0 max-h-[600px]">
            {chatMessages.length === 0 && (
              <p className="text-sm text-gray-400 text-center mt-4">No messages yet. Say hi!</p>
            )}
            {chatMessages.map((msg, index) => {
              const isOwn = msg.senderID === currentPlayer.id;
              return (
                <div
                  key={index}
                  className={`flex flex-col ${isOwn ? 'items-end' : 'items-start'}`}
                >
                  <span className="text-xs text-gray-400 mb-0.5 px-1">{msg.senderName}</span>
                  <div
                    className={`px-3 py-2 rounded-2xl text-sm max-w-xs break-words ${
                      isOwn
                        ? 'bg-blue-500 text-white rounded-br-sm'
                        : 'bg-gray-100 text-gray-800 rounded-bl-sm'
                    }`}
                  >
                    {msg.content}
                  </div>
                </div>
              );
            })}
            <div ref={chatBottomRef} />
          </div>

          {/* Typing indicator */}
          <div className="px-4 h-5 flex items-center">
            {typingText && (
              <span className="text-xs text-gray-400 italic">{typingText}</span>
            )}
          </div>

          {/* Input */}
          <div className="flex gap-2 p-3 border-t border-gray-200">
            <input
              type="text"
              value={chatInput}
              onChange={handleChatInput}
              onKeyDown={handleChatKeyDown}
              placeholder="Type a message..."
              className="flex-1 border border-gray-300 rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
            <button
              onClick={handleSendMessage}
              className="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-xl text-sm transition"
            >
              Send
            </button>
          </div>
        </div>

      </div>
    </div>
  );
}

function FriendsList({
  friendsList,
  onInvite,
}: {
  friendsList: Friend[];
  onInvite: (friendID: string) => void;
}) {
  const onlineFriends = friendsList.filter(f => f.isOnline);

  if (onlineFriends.length === 0) {
    return (
      <div className="px-3 py-2 text-sm text-gray-500">
        No friends online
      </div>
    );
  }

  return (
    <ul className="max-h-60 overflow-y-auto divide-y divide-gray-100 m-0 p-0 list-none">
      {onlineFriends.map(friend => (
        <li
          key={friend.id}
          className="flex items-center justify-between px-3 py-2 text-sm hover:bg-gray-50 transition-colors"
        >
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-green-500" />
            <span className="text-gray-800">{friend.name}</span>
          </div>
          <button
            onClick={() => onInvite(friend.id)}
            className="text-xs bg-blue-500 hover:bg-blue-600 text-white px-2 py-1 rounded-lg transition"
          >
            Invite
          </button>
        </li>
      ))}
    </ul>
  );
}