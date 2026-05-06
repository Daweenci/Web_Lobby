// CreateLobbyModal.tsx
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import React, { useState, useEffect } from 'react';
import { toast } from "sonner"

type CreateLobbyModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onCreateLobby: (name: string, maxPlayers: number, isPrivate: boolean, password: string) => void;
};

export function CreateLobbyModal({ isOpen, onClose, onCreateLobby }: CreateLobbyModalProps) {
  const [lobbyName, setLobbyName] = useState('');
  const [maxPlayers, setMaxPlayers] = useState<"2" | "3" | "4">("2");
  const [isPrivate, setIsPrivate] = useState(false);
  const [password, setPassword] = useState('');

  // Reset form when modal closes
  useEffect(() => {
    if (!isOpen) {
      setLobbyName('');
      setMaxPlayers("2");
      setIsPrivate(false);
      setPassword('');
    }
  }, [isOpen]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!lobbyName.trim()) {
      toast('Please enter a lobby name');
      return;
    }
    
    if (isPrivate && password.trim().length < 4) {
      toast('Please enter a password with at least 4 characters');
      return;
    }
    
    onCreateLobby(lobbyName.trim(), Number(maxPlayers), isPrivate, password.trim());
  };

  const toggleIsPrivate = () => {
    setIsPrivate(prev => {
      const newState = !prev;
      if (!newState) {
        setPassword('');
      }
      return newState;
    });
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-white/80 z-10">
      <div className="flex flex-col p-6 border rounded shadow-lg bg-white">
        <h2 className="text-xl mb-2 py-2">Create Lobby</h2>
        <form onSubmit={handleSubmit} className="flex flex-col gap-2">
          <input
            type="text"
            placeholder="Lobby Name"
            className="border p-2 py-2"
            value={lobbyName}
            onChange={(e) => setLobbyName(e.target.value)}
          />

          <div className="flex justify-between gap-2 py-2">
            <Label className="text-sm">Max Player Count:</Label>
            <RadioGroup
              value={maxPlayers}
              onValueChange={(value: "2" | "3" | "4") => setMaxPlayers(value)}
              className="flex justify-between gap-3"
            >
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="2" id="option-one" />
                <Label htmlFor="option-one">2</Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="3" id="option-two" />
                <Label htmlFor="option-two">3</Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="4" id="option-four" />
                <Label htmlFor="option-four">4</Label>
              </div>
            </RadioGroup>
          </div>

          <div className="flex justify-between gap-2 py-2">
            <Label htmlFor="privateLobby">Private Lobby:</Label>
            <Checkbox 
              id="privateLobby" 
              checked={isPrivate} 
              onCheckedChange={toggleIsPrivate} 
              className="h-4 w-4" 
            />
          </div>

          <div className="flex justify-between gap-2 py-2">
            <Label htmlFor="password">Password:</Label>
            <input
              type="text"
              id="password"
              placeholder="Enter password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={!isPrivate}
              className="border p-2 disabled:bg-gray-100"
            />
          </div>

          <div className="flex justify-between gap-2 py-2">
            <Button type="button" onClick={onClose}>Cancel</Button>
            <Button type="submit">Create Lobby</Button>
          </div>
        </form>
      </div>
    </div>
  );
}