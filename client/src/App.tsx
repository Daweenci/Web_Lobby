import React, { useState } from 'react';
import MainMenu from './pages/MainMenu';
import GameOfTwo from './pages/GameOfTwo';
import GameOfThree from './pages/GameOfThree';
import GameOfFour from './pages/GameOfFour';
import LobbyScreen from './pages/LobbyScreen';
import type { yourLobby, broadcastedLobby, PageType, Player, friendRequest, Friend, LobbyInvite, ChatMessage, TypingPlayer } from './structs';
import { Page } from './structs';
import useWebSocket from './useWebSocket';
import { Toaster } from 'sonner';
import Auth from './pages/Auth';

export default function App() {
  const [player, setPlayer] = useState<Player>({} as Player);
  const [broadcastedLobbies, setbroadcastedLobbies] = useState<broadcastedLobby[]>([]);
  const [lobby, setLobby] = useState<yourLobby>({} as yourLobby);
  const [pendingFriendRequests, setPendingFriendRequests] = useState<friendRequest[]>([]);
  const [friendsList, setFriendsList] = useState<Friend[]>([]);
  const [currentPage, setCurrentPage] = useState<PageType>(Page.Auth);
  const [pendingInvites, setPendingInvites] = useState<LobbyInvite[]>([]);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [typingPlayers, setTypingPlayers] = useState<TypingPlayer[]>([]);

  const {
    connect,
    createLobby,
    startGame,
    cancelGame,
    leaveLobby,
    joinLobby,
    logout,
    addFriend,
    acceptFriendRequest,
    inviteToLobby,
    declineInvite,
    addBotToLobby,
    removeBotFromLobby,
    sendLobbyChatMessage,
    startedTyping,
    stoppedTyping,
  } = useWebSocket({
    onSetPlayer: setPlayer,
    onSetLobby: setLobby,
    onSetLobbies: setbroadcastedLobbies,
    onSetPendingFriendRequests: setPendingFriendRequests,
    onSetFriendsList: setFriendsList,
    onSetPage: setCurrentPage,
    onSetPendingInvites: setPendingInvites,
    onSetChatMessages: setChatMessages,
    onSetTypingPlayers: setTypingPlayers,
  });

  const handleConnectWebSocket = () => { connect(); };
  const handleCreateLobby = (name: string, max: number, priv: boolean, pass: string) => createLobby(name, max, priv, pass);
  const handleStartGame = () => startGame(lobby.id);
  const handleCancelGame = () => cancelGame(lobby.id);
  const handleLeaveLobby = () => leaveLobby(lobby.id);
  const handleJoinLobby = (lobbyID: string, lobbyPassword: string) => joinLobby(lobbyID, lobbyPassword);
  const handleInviteToLobby = (friendID: string) => inviteToLobby(lobby.id, friendID);
  const handleDeclineInvite = (lobbyID: string) => declineInvite(lobbyID);
  const handleAddBotToLobby = () => addBotToLobby(lobby.id);
  const handleRemoveBotFromLobby = (botID: string) => removeBotFromLobby(lobby.id, botID);
  const handleSendLobbyChatMessage = (content: string) => sendLobbyChatMessage(lobby.id, content);
  const handleStartedTyping = () => startedTyping(lobby.id);
  const handleStoppedTyping = () => stoppedTyping(lobby.id);

  return (
    <>
      <Toaster />
      {(() => {
        switch (currentPage) {
          case Page.Auth:
            return <Auth connectWebSocket={handleConnectWebSocket} />;
          case Page.MainMenu:
            return (
              <MainMenu
                createLobby={handleCreateLobby}
                joinLobby={handleJoinLobby}
                lobbies={broadcastedLobbies}
                currentPlayerID={player.id}
                playerName={player.name}
                pendingFriendRequests={pendingFriendRequests}
                logout={logout}
                addFriend={addFriend}
                acceptFriendRequest={acceptFriendRequest}
                friendsList={friendsList}
                pendingInvites={pendingInvites}
                declineInvite={handleDeclineInvite}
              />
            );
          case Page.InLobby:
            return (
              <LobbyScreen
                startGame={handleStartGame}
                cancelGame={handleCancelGame}
                leaveLobby={handleLeaveLobby}
                friendsList={friendsList}
                lobby={lobby}
                currentPlayerID={player.id}
                chatMessages={chatMessages}
                typingPlayers={typingPlayers}
                inviteToLobby={handleInviteToLobby}
                addBotToLobby={handleAddBotToLobby}
                removeBotFromLobby={handleRemoveBotFromLobby}
                sendLobbyChatMessage={handleSendLobbyChatMessage}
                startedTyping={handleStartedTyping}
                stoppedTyping={handleStoppedTyping}
              />
            );
          case Page.GameOfTwo:
            return <GameOfTwo />;
          case Page.GameOfThree:
            return <GameOfThree />;
          case Page.GameOfFour:
            return <GameOfFour />;
          default:
            return null;
        }
      })()}
    </>
  );
}