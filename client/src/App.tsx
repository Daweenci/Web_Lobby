import MainMenu from './pages/MainMenu';
import GameOfTwo from './pages/GameOfTwo';
import GameOfThree from './pages/GameOfThree';
import GameOfFour from './pages/GameOfFour';
import LobbyScreen from './pages/LobbyScreen';
import { Page } from './structs';
import useWebSocket from './useWebSocket';
import { Toaster } from 'sonner';
import Auth from './pages/Auth';
import { useStore } from './store';

export default function App() {

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
  } = useWebSocket();

  const handleConnectWebSocket = () => { connect(); };
  const handleCreateLobby = (name: string, max: number, priv: boolean, pass: string) => createLobby(name, max, priv, pass);
  const handleStartGame = () => startGame(useStore.getState().lobby.id);
  const handleCancelGame = () => cancelGame(useStore.getState().lobby.id);
  const handleLeaveLobby = () => leaveLobby(useStore.getState().lobby.id);
  const handleJoinLobby = (lobbyID: string, lobbyPassword: string) => joinLobby(lobbyID, lobbyPassword);
  const handleInviteToLobby = (friendID: string) => inviteToLobby(useStore.getState().lobby.id, friendID);
  const handleDeclineInvite = (lobbyID: string) => declineInvite(lobbyID);
  const handleAddBotToLobby = () => addBotToLobby(useStore.getState().lobby.id);
  const handleRemoveBotFromLobby = (botID: string) => removeBotFromLobby(useStore.getState().lobby.id, botID);
  const handleSendLobbyChatMessage = (content: string) => sendLobbyChatMessage(useStore.getState().lobby.id, content);
  const handleStartedTyping = () => startedTyping(useStore.getState().lobby.id);
  const handleStoppedTyping = () => stoppedTyping(useStore.getState().lobby.id);

  return (
    <>
      <Toaster />
      {(() => {
        switch (useStore.getState().currentPage) {
          case Page.Auth:
            return <Auth connectWebSocket={handleConnectWebSocket} />;
          case Page.MainMenu:
            return (
              <MainMenu
                createLobby={handleCreateLobby}
                joinLobby={handleJoinLobby}
                lobbies={useStore.getState().broadcastedLobbies}
                currentPlayerID={useStore.getState().player.id}
                playerName={useStore.getState().player.name}
                pendingFriendRequests={useStore.getState().pendingFriendRequests}
                logout={logout}
                addFriend={addFriend}
                acceptFriendRequest={acceptFriendRequest}
                friendsList={useStore.getState().friendsList}
                pendingInvites={useStore.getState().pendingInvites}
                declineInvite={handleDeclineInvite}
              />
            );
          case Page.InLobby:
            return (
              <LobbyScreen
                startGame={handleStartGame}
                cancelGame={handleCancelGame}
                leaveLobby={handleLeaveLobby}
                friendsList={useStore.getState().friendsList}
                lobby={useStore.getState().lobby}
                currentPlayerID={useStore.getState().player.id}
                chatMessages={useStore.getState().chatMessages}
                typingPlayers={useStore.getState().typingPlayers}
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