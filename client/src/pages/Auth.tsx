import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { toast } from "sonner";

type AuthProps = {
  connectWebSocket: () => void; 
};

export function Auth({ connectWebSocket }: AuthProps) {
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [activeTab, setActiveTab] = useState<'login' | 'register'>('login');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    if (activeTab === 'login') {
      if (!name.trim() || !password.trim()) {
        toast('Please fill in all fields');
        return;
      }
      onLogin(name.trim(), password.trim());
    } else {
      // Register logic
      if (!name.trim() || !password.trim() || !confirmPassword.trim()) {
        toast('Please fill in all fields');
        return;
      }
      if (password !== confirmPassword) {
        toast('Passwords do not match');
        return;
      }
      onRegister(name.trim(), password.trim());
    }
  };

  async function onRegister(name: string, password: string) {
    try {
      const API_URL = import.meta.env.REACT_APP_API_URL || 'http://localhost:4000';
      const response = await fetch(`${API_URL}/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name, password }),
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.message);
      }

      if (data.token) {
        localStorage.setItem('gameToken', data.token);
      }

      connectWebSocket();
      toast('Registration successful');
    } catch (error: any) {
      console.error('Registration error:', error);
      toast(error.message);
    }
  }

  async function onLogin(name: string, password: string) {
    try {
      const API_URL = import.meta.env.REACT_APP_API_URL || 'http://localhost:4000';
      const response = await fetch(`${API_URL}/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name, password }),
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.message);
      }

      if (data.token) {
        localStorage.setItem('gameToken', data.token);
      }

      connectWebSocket();
      toast('Login successful');
    } catch (error: any) {
      console.error('Login error:', error);
      toast(error.message);
    }
  }

  return (
    <div className="h-screen flex items-center justify-center bg-teal-50">
      <div className="bg-white p-8 rounded-lg shadow-lg w-96">
        {/* Tab Headers */}
        <div className="flex mb-6">
          <button
            type="button"
            onClick={() => setActiveTab('login')}
            className={`flex-1 py-3 px-4 text-center font-medium transition-colors ${
              activeTab === 'login'
                ? 'bg-teal-500 text-white'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
            } rounded-tl-lg rounded-bl-lg`}
          >
            Sign In
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('register')}
            className={`flex-1 py-3 px-4 text-center font-medium transition-colors ${
              activeTab === 'register'
                ? 'bg-teal-500 text-white'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
            } rounded-tr-lg rounded-br-lg`}
          >
            Sign Up
          </button>
        </div>

        {/* Form Content */}
        <div className="text-center mb-6">
          <h1 className="text-2xl font-bold text-gray-800">
            {activeTab === 'login' ? 'Welcome Back!' : 'Get Started!'}
          </h1>
        </div>

        <div className="space-y-4">
          {/* Name Field */}
          <div>
            <Input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Your name"
              className="w-full"
            />
          </div>

          {/* Password Field - Always show */}
          <div>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Password"
              className="w-full"
            />
          </div>

          {/* Confirm Password Field - Only for Register */}
          {activeTab === 'register' && (
            <div>
              <Input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Confirm Password"
                className="w-full"
              />
            </div>
          )}

          {/* Submit Button */}
          <Button 
            onClick={handleSubmit}
            className="w-full bg-red-500 hover:bg-red-600 text-white py-3 rounded-lg font-medium text-lg"
          >
            {activeTab === 'login' ? 'LOG IN' : 'REGISTER'}
          </Button>
        </div>
      </div>
    </div>
  );
}
