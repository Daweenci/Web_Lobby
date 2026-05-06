// JoinPasswordModal.tsx
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import React, { useState, useEffect } from 'react';

type JoinPasswordModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onJoinLobby: (password: string) => void;
};

export function JoinPasswordModal({ isOpen, onClose, onJoinLobby }: JoinPasswordModalProps) {
  const [password, setPassword] = useState('');

  // Reset password when modal closes
  useEffect(() => {
    if (!isOpen) {
      setPassword('');
    }
  }, [isOpen]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onJoinLobby(password);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-white/80 z-10">
      <div className="flex flex-col p-6 border rounded shadow-lg bg-white">
        <form onSubmit={handleSubmit} className="flex flex-col gap-2">
          <div>
            <Label>Password:</Label>
            <input
              type="text"
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter password"
              value={password}
              className="border p-2 w-full"
              autoFocus
            />
          </div>
          <div className="flex justify-between gap-2 py-2">
            <Button type="button" onClick={onClose}>Cancel</Button>
            <Button type="submit">Join</Button>
          </div>
        </form>
      </div>
    </div>
  );
}