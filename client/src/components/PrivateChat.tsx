import { useState, useRef, useEffect } from 'react';
import { useStore } from '@/store';
import { useShallow } from 'zustand/react/shallow';
import type { Store } from '@/store';
import type { Friend } from '@/structs';

// Per-friend chat message shape (local only for now)
type PrivateMessage = {
  senderID: string;
  content: string;
  timestamp: number;
};

type FriendChatState = {
  messages: PrivateMessage[];
  unread: number;
  lastActivity: number; // timestamp of most recent message either side
};

// We keep chat state in a ref-stable map outside of React state to avoid
// re-renders on every keystroke, but expose it through useState snapshots.
function useFriendChats(friendIDs: string[]) {
  const [chats, setChats] = useState<Record<string, FriendChatState>>(() => {
    const init: Record<string, FriendChatState> = {};
    for (const id of friendIDs) {
      init[id] = { messages: [], unread: 0, lastActivity: 0 };
    }
    return init;
  });

  // Make sure newly-added friends get a slot
  useEffect(() => {
    setChats((prev) => {
      let changed = false;
      const next = { ...prev };
      for (const id of friendIDs) {
        if (!next[id]) {
          next[id] = { messages: [], unread: 0, lastActivity: 0 };
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [friendIDs.join(',')]); // eslint-disable-line react-hooks/exhaustive-deps

  const sendMessage = (friendID: string, senderID: string, content: string) => {
    const msg: PrivateMessage = { senderID, content, timestamp: Date.now() };
    setChats((prev) => ({
      ...prev,
      [friendID]: {
        messages: [...(prev[friendID]?.messages ?? []), msg],
        unread: 0, // own messages don't add unread
        lastActivity: msg.timestamp,
      },
    }));
  };

  const markRead = (friendID: string) => {
    setChats((prev) => {
      if (!prev[friendID] || prev[friendID].unread === 0) return prev;
      return { ...prev, [friendID]: { ...prev[friendID], unread: 0 } };
    });
  };

  return { chats, sendMessage, markRead };
}

export function PrivateChat() {
  const { player, friendsList } = useStore(
    useShallow((state: Store) => ({
      player: state.player,
      friendsList: state.friendsList,
    }))
  );

  const friendIDs = friendsList.map((f) => f.id);
  const { chats, sendMessage, markRead } = useFriendChats(friendIDs);

  const [isOpen, setIsOpen] = useState(false);
  const [activeFriendID, setActiveFriendID] = useState<string | null>(null);
  const [inputValue, setInputValue] = useState('');
  const chatBottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Scroll to bottom when active chat messages change
  useEffect(() => {
    chatBottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [activeFriendID, chats]);

  // Focus input when switching chats
  useEffect(() => {
    if (activeFriendID) {
      inputRef.current?.focus();
    }
  }, [activeFriendID]);

  const totalUnread = Object.values(chats).reduce((acc, c) => acc + c.unread, 0);

  // Sort friends: online first, then offline. Within each group, sort by lastActivity desc.
  const sortedFriends = [...friendsList].sort((a, b) => {
    if (a.isOnline !== b.isOnline) return a.isOnline ? -1 : 1;
    const aActivity = chats[a.id]?.lastActivity ?? 0;
    const bActivity = chats[b.id]?.lastActivity ?? 0;
    return bActivity - aActivity;
  });

  const activeFriend = friendsList.find((f) => f.id === activeFriendID) ?? null;
  const activeChat = activeFriendID ? chats[activeFriendID] : null;

  const handleSelectFriend = (friend: Friend) => {
    setActiveFriendID(friend.id);
    markRead(friend.id);
    setInputValue('');
  };

  const handleSend = () => {
    const trimmed = inputValue.trim();
    if (!trimmed || !activeFriendID) return;
    sendMessage(activeFriendID, player.id, trimmed);
    setInputValue('');
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') handleSend();
  };

  const handleToggle = () => {
    setIsOpen((prev) => !prev);
    if (!isOpen) {
      // reset active friend when opening fresh
    }
  };

  const handleBack = () => {
    setActiveFriendID(null);
    setInputValue('');
  };

  return (
    <div className="fixed bottom-4 left-6 z-50 flex flex-col items-start gap-2">
      {/* Expanded panel */}
      {isOpen && (
        <div
          className="flex bg-white border-2 border-gray-200 rounded-2xl shadow-2xl overflow-hidden"
          style={{ width: 420, height: 480 }}
        >
          {/* Left column — friends list */}
          <div className="flex flex-col w-40 shrink-0 border-r border-gray-100 bg-gray-50">
            <div className="px-3 py-2 border-b border-gray-200">
              <span className="text-xs font-bold text-gray-500 uppercase tracking-wider">Friends</span>
            </div>
            <ul className="flex-1 overflow-y-auto">
              {sortedFriends.length === 0 && (
                <li className="px-3 py-3 text-xs text-gray-400 text-center">No friends yet</li>
              )}
              {sortedFriends.map((friend) => {
                const friendChat = chats[friend.id];
                const unread = friendChat?.unread ?? 0;
                const isActive = friend.id === activeFriendID;
                const lastMsg = friendChat?.messages[friendChat.messages.length - 1];

                return (
                  <li key={friend.id}>
                    <button
                      onClick={() => handleSelectFriend(friend)}
                      className={`w-full text-left px-3 py-2.5 flex items-center gap-2 transition-colors ${
                        isActive
                          ? 'bg-blue-50 border-r-2 border-blue-500'
                          : 'hover:bg-gray-100'
                      }`}
                    >
                      {/* Online dot */}
                      <span
                        className={`w-2 h-2 rounded-full shrink-0 ${
                          friend.isOnline ? 'bg-green-500' : 'bg-gray-300'
                        }`}
                      />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between gap-1">
                          <span
                            className={`text-xs font-semibold truncate ${
                              friend.isOnline ? 'text-gray-800' : 'text-gray-400'
                            }`}
                          >
                            {friend.name}
                          </span>
                          {unread > 0 && (
                            <span className="bg-red-500 text-white text-[10px] font-bold rounded-full h-4 w-4 flex items-center justify-center shrink-0">
                              {unread}
                            </span>
                          )}
                        </div>
                        {lastMsg && (
                          <p className="text-[10px] text-gray-400 truncate mt-0.5">
                            {lastMsg.senderID === player.id ? 'You: ' : ''}
                            {lastMsg.content}
                          </p>
                        )}
                      </div>
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>

          {/* Right column — chat area */}
          <div className="flex flex-col flex-1 min-w-0">
            {activeFriend ? (
              <>
                {/* Chat header */}
                <div className="flex items-center gap-2 px-3 py-2 border-b border-gray-100 bg-white shrink-0">
                  <button
                    onClick={handleBack}
                    className="text-gray-400 hover:text-gray-600 transition md:hidden"
                    title="Back"
                  >
                    ←
                  </button>
                  <span
                    className={`w-2 h-2 rounded-full shrink-0 ${
                      activeFriend.isOnline ? 'bg-green-500' : 'bg-gray-300'
                    }`}
                  />
                  <span className="text-sm font-semibold text-gray-800 truncate">{activeFriend.name}</span>
                  <span className="text-xs text-gray-400 ml-auto">
                    {activeFriend.isOnline ? 'Online' : 'Offline'}
                  </span>
                </div>

                {/* Messages */}
                <div className="flex-1 overflow-y-auto p-3 flex flex-col gap-1.5">
                  {activeChat?.messages.length === 0 && (
                    <p className="text-xs text-gray-400 text-center mt-6">
                      Say hi to {activeFriend.name}!
                    </p>
                  )}
                  {activeChat?.messages.map((msg, i) => {
                    const isOwn = msg.senderID === player.id;
                    return (
                      <div
                        key={i}
                        className={`flex ${isOwn ? 'justify-end' : 'justify-start'}`}
                      >
                        <div
                          className={`px-2.5 py-1.5 rounded-2xl text-xs max-w-[75%] break-words ${
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

                {/* Input */}
                <div className="flex gap-2 p-2.5 border-t border-gray-100 bg-white shrink-0">
                  <input
                    ref={inputRef}
                    type="text"
                    value={inputValue}
                    onChange={(e) => setInputValue(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder={`Message ${activeFriend.name}...`}
                    className="flex-1 border border-gray-200 rounded-xl px-2.5 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-blue-400"
                  />
                  <button
                    onClick={handleSend}
                    className="bg-blue-500 hover:bg-blue-600 text-white px-3 py-1.5 rounded-xl text-xs transition"
                  >
                    Send
                  </button>
                </div>
              </>
            ) : (
              <div className="flex-1 flex items-center justify-center">
                <p className="text-xs text-gray-400 text-center px-4">
                  Select a friend to start chatting
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Toggle button */}
      <button
        onClick={handleToggle}
        className={`relative flex items-center gap-2 px-4 py-2.5 rounded-2xl shadow-lg border-2 transition-all text-sm font-semibold ${
          isOpen
            ? 'bg-blue-500 border-blue-500 text-white hover:bg-blue-600'
            : 'bg-white border-gray-200 text-gray-700 hover:bg-gray-50'
        }`}
      >
        {/* Chat bubble icon */}
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="currentColor"
          className="w-4 h-4"
        >
          <path
            fillRule="evenodd"
            d="M4.804 21.644A6.707 6.707 0 006 21.75a6.721 6.721 0 003.583-1.029c.774.182 1.584.279 2.417.279 5.322 0 9.75-3.97 9.75-9 0-5.03-4.428-9-9.75-9s-9.75 3.97-9.75 9c0 2.409 1.025 4.587 2.674 6.192.232.226.277.428.254.543a3.73 3.73 0 01-.814 1.686.75.75 0 00.44 1.223 3.677 3.677 0 00-.001 0z"
            clipRule="evenodd"
          />
        </svg>
        <span>Chat</span>

        {/* Unread badge */}
        {totalUnread > 0 && (
          <span className="absolute -top-1.5 -right-1.5 bg-red-500 text-white text-[10px] font-bold rounded-full h-5 w-5 flex items-center justify-center shadow">
            {totalUnread > 99 ? '99+' : totalUnread}
          </span>
        )}
      </button>
    </div>
  );
}