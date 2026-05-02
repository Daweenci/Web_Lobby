// MainMenu.tsx
import { Button } from '@/components/ui/button';
import React, { useState } from 'react';
import type { broadcastedLobby, Friend, friendRequest, LobbyInvite } from '@/structs';
import CreateLobbyModal from './CreateLobbyModal';
import JoinPasswordModal from './JoinPasswordModal';
import { toast } from 'sonner';
import UserProfile from '@/components/UserProfile';

type MainMenuProps = {
  createLobby: (name: string, maxPlayers: number, isPrivate: boolean, password: string) => void;
  joinLobby: (id: string, joinPassword: string) => void;
  lobbies: broadcastedLobby[];
  currentPlayerID: string;
  playerName: string;
  pendingFriendRequests: friendRequest[];
  friendsList: Friend[];
  logout: () => void;
  addFriend: (friendName: string) => void;
  acceptFriendRequest: (friendID: string, accept: boolean) => void;
  pendingInvites: LobbyInvite[];
  declineInvite: (lobbyID: string) => void;
};

export default function MainMenu({
  createLobby,
  joinLobby,
  lobbies,
  playerName,
  pendingFriendRequests,
  friendsList,
  logout,
  addFriend,
  acceptFriendRequest,
  pendingInvites,
  declineInvite,
}: MainMenuProps) {
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showJoinModal, setShowJoinModal] = useState(false);
  const [selectedLobbyId, setSelectedLobbyId] = useState('');

  const handleJoinLobby = (lobbyID: string, password: string) => {
    joinLobby(lobbyID, password);
  };

  const handleCreateLobby = (name: string, maxPlayers: number, isPrivate: boolean, password: string) => {
    createLobby(name, maxPlayers, isPrivate, password);
  };

  const joinLobbyAccess = (id: string) => {
    for (const lobby of lobbies) {
      if (lobby.id === id) {
        const totalOccupied = (lobby.players?.length ?? 0) + (lobby.bots?.length ?? 0);
        if (totalOccupied >= lobby.maxPlayers) {
          toast('This lobby is full!');
          return;
        }
        if (lobby.isPrivate) {
          setSelectedLobbyId(lobby.id);
          setShowJoinModal(true);
        } else {
          joinLobby(id, '');
        }
        break;
      }
    }
  };

  return (
    <div>
      <div className="relative px-6 py-4">
        <h1 className="text-4xl font-bold text-center">Main Menu</h1>
        <div className="absolute top-4 right-6">
          <UserProfile
            playerName={playerName}
            onLogout={logout}
            onAddFriend={addFriend}
            pendingFriendRequests={pendingFriendRequests}
            onAcceptFriendRequest={acceptFriendRequest}
            friendsList={friendsList}
            pendingInvites={pendingInvites}
            onDeclineInvite={declineInvite}
          />
        </div>
      </div>

      <div id="existingLobbies" className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4 px-6">
        {lobbies?.map((lobby) => {
          const totalOccupied = (lobby.players?.length ?? 0) + (lobby.bots?.length ?? 0);
          return (
            <div
              key={lobby.id}
              onClick={() => joinLobbyAccess(lobby.id)}
              className="p-4 border rounded shadow bg-white hover:shadow-md transition cursor-pointer"
            >
              <h3 className="text-lg font-semibold mb-2">{lobby.name}</h3>
              <p>Players <strong>({totalOccupied}/{lobby.maxPlayers})</strong>:</p>
              <ul className="list-disc pl-5 mb-2">
                {lobby.players?.map((player, index) => (
                  <li key={index}>{player.name}</li>
                ))}
                {lobby.bots?.map((bot, index) => (
                  <li key={'bot-' + index}>{bot.name}</li>
                ))}
              </ul>
              <p>{lobby.isPrivate ? '🔒 Private' : '🌐 Public'}</p>
            </div>
          );
        })}
      </div>

      <CreateLobbyModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onCreateLobby={handleCreateLobby}
      />

      <JoinPasswordModal
        isOpen={showJoinModal}
        onClose={() => {
          setShowJoinModal(false);
          setSelectedLobbyId('');
        }}
        onJoinLobby={(password) => handleJoinLobby(selectedLobbyId, password)}
      />

      <Button
        className="mb-4 text-xl p-6 fixed bottom-10 right-16 z-1"
        onClick={() => setShowCreateModal(true)}
      >
        Create Lobby
      </Button>
    </div>
  );
}