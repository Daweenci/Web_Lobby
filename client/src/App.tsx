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

  const lobby = useStore((state) => (state.lobby));
  const currentPage = useStore((state) => (state.currentPage));
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
                logout={logout}
                addFriend={addFriend}
                acceptFriendRequest={acceptFriendRequest}
                declineInvite={handleDeclineInvite}
              />
            );
          case Page.InLobby:
            return (
              <LobbyScreen
                startGame={handleStartGame}
                cancelGame={handleCancelGame}
                leaveLobby={handleLeaveLobby}
                inviteToLobby={handleInviteToLobby}
                addBotToLobby={handleAddBotToLobby}
                removeBotFromLobby={handleRemoveBotFromLobby}
                sendLobbyChatMessage={handleSendLobbyChatMessage}
                startedTyping={handleStartedTyping}
                stoppedTyping={handleStoppedTyping}
                logout={logout}
                addFriend={addFriend}
                acceptFriendRequest={acceptFriendRequest}
                declineInvite={handleDeclineInvite}
                joinLobby={handleJoinLobby}
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