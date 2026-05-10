import { MainMenu } from './pages/MainMenu';
import { LobbyScreen } from './pages/LobbyScreen';
import { Page } from './structs';
import { useWebSocket } from './useWebSocket';
import { Toaster } from 'sonner';
import { Auth } from './pages/Auth';
import { useStore } from './store';
import type { CustomBotConfig } from './structs';
import { UserProfile } from '@/components/UserProfile';
import { PrivateChat } from '@/components/PrivateChat';

export function App() {

  const lobby = useStore((state) => (state.lobby));
  const currentPage = useStore((state) => (state.currentPage));
  const {
    connect,
    createLobby,
    startGame,
    cancelGame,
    leaveLobby,
    joinLobby,
    removeFriend,
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
    createCustomBot,
  } = useWebSocket();

  const handleConnectWebSocket = () => { connect(); };
  const handleCreateLobby = (name: string, max: number, priv: boolean, pass: string) => createLobby(name, max, priv, pass);
  const handleStartGame = () => startGame(lobby.id);
  const handleCancelGame = () => cancelGame(lobby.id);
  const handleLeaveLobby = () => leaveLobby(lobby.id);
  const handleJoinLobby = (lobbyID: string, lobbyPassword: string) => joinLobby(lobbyID, lobbyPassword);
  const handleRemoveFriend = (friendID: string) => removeFriend(friendID);
  const handleInviteToLobby = (friendID: string) => inviteToLobby(lobby.id, friendID);
  const handleDeclineInvite = (lobbyID: string) => declineInvite(lobbyID);
  const handleAddBotToLobby = () => addBotToLobby(lobby.id);
  const handleRemoveBotFromLobby = (botID: string) => removeBotFromLobby(lobby.id, botID);
  const handleSendLobbyChatMessage = (content: string) => sendLobbyChatMessage(lobby.id, content);
  const handleStartedTyping = () => startedTyping(lobby.id);
  const handleStoppedTyping = () => stoppedTyping(lobby.id);
  const handleCreateCustomBot = (cfg: CustomBotConfig) =>
  createCustomBot(lobby.id, cfg.archetypeId, cfg.energy, cfg.tilt, cfg.chaos, cfg.humor,
    cfg.usesEmojis, cfg.usesGamingSlang, cfg.usesMemes, cfg.asksQuestions);
  return (
    <>
      <Toaster />
      {currentPage !== Page.Auth && (
        <>
          <div className="absolute top-4 right-6 z-50">
            <UserProfile
              onLogout={logout}
              onAddFriend={addFriend}
              onAcceptFriendRequest={acceptFriendRequest}
              onDeclineInvite={handleDeclineInvite}
              onJoinLobby={(lobbyID) => joinLobby(lobbyID, '')}
              onRemoveFriend={(friendID) => handleRemoveFriend(friendID)}
            />
          </div>
          <PrivateChat />
      </>
      )}
      {(() => {
        switch (currentPage) {
          case Page.Auth:
            return <Auth connectWebSocket={handleConnectWebSocket} />;
          case Page.MainMenu:
            return (
              <MainMenu
                createLobby={handleCreateLobby}
                joinLobby={handleJoinLobby}
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
                createCustomBot={handleCreateCustomBot}
              />
            );
          default:
            return null;
        }
      })()}
    </>
  );
}