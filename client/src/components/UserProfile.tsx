import { useState, useRef } from "react";
import profileIcon from "@/assets/user-profile-icon.svg";
import { useStore } from "@/store";
import { useShallow } from "zustand/react/shallow";
import type { Store } from "@/store";


type Props = {
  onLogout: () => void;
  onAddFriend: (friendName: string) => void;
  onAcceptFriendRequest: (friendID: string, accept: boolean) => void;
  onDeclineInvite: (lobbyID: string) => void;
  onJoinLobby: (lobbyID: string) => void;
};

export default function UserProfile({
  onLogout,
  onAddFriend,
  onAcceptFriendRequest,
  onDeclineInvite,
  onJoinLobby,
}: Props) {
  const { player, pendingFriendRequests, friendsList, pendingInvites } = useStore(
    useShallow((state: Store) => ({
      player: state.player,
      pendingFriendRequests: state.pendingFriendRequests,
      friendsList: state.friendsList,
      pendingInvites: state.pendingInvites,
    }))
  );
  type View = "menu" | "friends" | "invites" | "settings" | null;
  const [view, setView] = useState<View>(null);
  const friendInputRef = useRef<HTMLInputElement>(null);

  const toggleDropdown = () => {
    setView(prev => (prev ? null : "menu"));
  };

  const totalNotifications = pendingFriendRequests.length + pendingInvites.length;

  return (
    <div className="relative">
      <button
        onClick={toggleDropdown}
        className="px-4 py-2 bg-gray-200 rounded hover:bg-gray-300"
      >
        <span className="text-lg font-semibold">{player.name}</span>
        <img src={profileIcon} alt="Profile Icon" className="inline w-8 h-8 ml-1" />
        {totalNotifications > 0 && (
          <span className="absolute top-0 right-0 bg-red-500 text-white text-xs rounded-full h-5 w-5 flex items-center justify-center">
            {totalNotifications}
          </span>
        )}
      </button>

      {view && (
        <div className="absolute right-0 mt-2 w-40 bg-white border rounded shadow">

          {view === "menu" && (
            <>
              <button
                onClick={() => setView("friends")}
                className="w-full text-left px-4 py-2 hover:bg-gray-100 flex items-center justify-between"
              >
                <span>Friends</span>
                {pendingFriendRequests.length > 0 && (
                  <span className="bg-red-500 text-white text-xs rounded-full h-4 w-4 flex items-center justify-center">
                    {pendingFriendRequests.length}
                  </span>
                )}
              </button>
              <button
                onClick={() => setView("invites")}
                className="w-full text-left px-4 py-2 hover:bg-gray-100 flex items-center justify-between"
              >
                <span>Invites</span>
                {pendingInvites.length > 0 && (
                  <span className="bg-blue-500 text-white text-xs rounded-full h-4 w-4 flex items-center justify-center">
                    {pendingInvites.length}
                  </span>
                )}
              </button>
              <button
                onClick={() => setView("settings")}
                className="w-full text-left px-4 py-2 hover:bg-gray-100"
              >
                Settings
              </button>
              <button
                onClick={onLogout}
                className="w-full text-left px-4 py-2 hover:bg-gray-100"
              >
                Logout
              </button>
            </>
          )}

          {view === "friends" && (
            <div className="absolute right-0 mt-2 w-72 bg-white border rounded shadow">
              <div className="p-3 flex flex-col gap-2 items-start">
                <button
                  onClick={() => setView("menu")}
                  className="text-sm border border-blue-500 text-blue-500 rounded px-3 py-1 hover:bg-blue-500 hover:text-white transition duration-200"
                >
                  ← Back
                </button>

                <div className="flex gap-2 w-full">
                  <input
                    ref={friendInputRef}
                    type="text"
                    placeholder="Enter player name..."
                    className="flex-1 min-w-0 border border-gray-300 rounded px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <button
                    onClick={() => onAddFriend(friendInputRef.current?.value || '')}
                    className="bg-blue-500 text-white px-3 py-1 rounded hover:bg-blue-600 transition duration-200 text-sm whitespace-nowrap"
                  >
                    Add
                  </button>
                </div>

                <h3 className="text-sm font-semibold mt-2 text-gray-700">Pending Requests</h3>
                {pendingFriendRequests.length === 0 ? (
                  <p className="text-sm text-gray-400">No pending requests</p>
                ) : (
                  <ul className="w-full flex flex-col gap-1">
                    {pendingFriendRequests.map((req) => (
                      <li key={req.id} className="flex items-center justify-between gap-2">
                        <span className="text-sm text-gray-600">{req.name}</span>
                        <div className="flex gap-1">
                          <button
                            className="bg-green-500 text-white text-xs px-2 py-1 rounded hover:bg-green-600 transition"
                            onClick={() => onAcceptFriendRequest(req.id, true)}
                          >
                            ✓
                          </button>
                          <button
                            className="bg-red-500 text-white text-xs px-2 py-1 rounded hover:bg-red-600 transition"
                            onClick={() => onAcceptFriendRequest(req.id, false)}
                          >
                            ✕
                          </button>
                        </div>
                      </li>
                    ))}
                  </ul>
                )}

                <h3 className="text-sm font-semibold mt-2 text-gray-700">Friends</h3>
                {friendsList.length === 0 ? (
                  <p className="text-sm text-gray-400">No friends yet</p>
                ) : (
                  <ul className="w-full flex flex-col gap-1">
                    {friendsList.map((friend) => (
                      <li key={friend.id} className="flex items-center gap-2">
                        <span className={`inline-block w-2 h-2 rounded-full ${friend.isOnline ? 'bg-green-500' : 'bg-gray-400'}`} />
                        <span className="text-sm text-gray-700">{friend.name}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          )}

          {view === "invites" && (
            <div className="absolute right-0 mt-2 w-72 bg-white border rounded shadow">
              <div className="p-3 flex flex-col gap-2 items-start">
                <button
                  onClick={() => setView("menu")}
                  className="text-sm border border-blue-500 text-blue-500 rounded px-3 py-1 hover:bg-blue-500 hover:text-white transition duration-200"
                >
                  ← Back
                </button>

                <h3 className="text-sm font-semibold text-gray-700">Pending Invites</h3>
                {pendingInvites.length === 0 ? (
                  <p className="text-sm text-gray-400">No pending invites</p>
                ) : (
                  <ul className="w-full flex flex-col gap-2">
                    {pendingInvites.map((invite) => (
                      <li key={invite.lobbyID} className="flex flex-col gap-1 border border-gray-100 rounded-lg p-2">
                        <span className="text-sm text-gray-800">
                          <strong>{invite.inviterName}</strong> → <strong>{invite.lobbyName}</strong>
                        </span>
                        <div className="flex justify-end gap-2 mt-1">
                          <button
                            onClick={() => onJoinLobby(invite.lobbyID)}
                            className="bg-blue-500 hover:bg-blue-600 text-white text-xs px-2 py-1 rounded transition"
                          >
                            Join
                          </button>

                          <button
                            onClick={() => onDeclineInvite(invite.lobbyID)}
                            className="bg-gray-100 hover:bg-gray-200 text-gray-700 text-xs px-2 py-1 rounded transition"
                          >
                            Dismiss
                          </button>
                        </div>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          )}

          {view === "settings" && (
            <div className="p-2">
              <button
                onClick={() => setView("menu")}
                className="text-sm mb-2 border border-blue-500 text-blue-500 rounded px-3 py-1 hover:bg-blue-500 hover:text-white transition duration-200"
              >
                ← Back
              </button>
              <div>Settings component here</div>
            </div>
          )}

        </div>
      )}
    </div>
  );
}