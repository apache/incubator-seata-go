/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import type { WebSocketMessage, WebSocketContextType, AgentResponse } from '../types';

const WebSocketContext = createContext<WebSocketContextType | null>(null);

interface WebSocketProviderProps {
  children: ReactNode;
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ children }) => {
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);
  const [currentResponse, setCurrentResponse] = useState<AgentResponse | null>(null);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const maxReconnectAttempts = 5;

  const generateSessionId = () => {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  };

  const connect = () => {
    try {
      const sessionId = generateSessionId();
      const wsUrl = `ws://localhost:8080/ws?session_id=${sessionId}`;
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        setReconnectAttempts(0);
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          console.log('Received message:', message);
          setLastMessage(message);
          
          // 处理Agent响应数据
          if (message.type === 'agent_response' && message.data) {
            try {
              // 尝试解析Agent的JSON响应
              const agentData = typeof message.data === 'string' 
                ? JSON.parse(message.data) 
                : message.data;
              
              if (agentData && typeof agentData === 'object') {
                setCurrentResponse(agentData as AgentResponse);
              }
            } catch (parseError) {
              console.error('Failed to parse agent response:', parseError);
            }
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      ws.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason);
        setIsConnected(false);
        setSocket(null);

        // 自动重连
        if (reconnectAttempts < maxReconnectAttempts) {
          const timeout = Math.pow(2, reconnectAttempts) * 1000; // 指数退避
          console.log(`Attempting to reconnect in ${timeout}ms...`);
          setTimeout(() => {
            setReconnectAttempts(prev => prev + 1);
            connect();
          }, timeout);
        } else {
          console.log('Max reconnection attempts reached');
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

      setSocket(ws);
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
    }
  };

  useEffect(() => {
    connect();

    return () => {
      if (socket) {
        socket.close();
      }
    };
  }, []);

  const sendMessage = async (content: string): Promise<void> => {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket is not connected');
    }

    const message = {
      type: 'user_input',
      data: {
        content: content,
        timestamp: new Date().toISOString()
      }
    };

    try {
      socket.send(JSON.stringify(message));
    } catch (error) {
      console.error('Failed to send message:', error);
      throw error;
    }
  };

  const clearChat = async (): Promise<void> => {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket is not connected');
    }

    const message = {
      type: 'clear_chat',
      data: {}
    };

    try {
      socket.send(JSON.stringify(message));
    } catch (error) {
      console.error('Failed to clear chat:', error);
      throw error;
    }
  };

  const reconnect = () => {
    if (socket) {
      socket.close();
    }
    setReconnectAttempts(0);
    connect();
  };

  const value: WebSocketContextType = {
    isConnected,
    lastMessage,
    currentResponse,
    sendMessage,
    clearChat,
    reconnect,
  };

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
};

export const useWebSocketContext = (): WebSocketContextType => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocketContext must be used within a WebSocketProvider');
  }
  return context;
};