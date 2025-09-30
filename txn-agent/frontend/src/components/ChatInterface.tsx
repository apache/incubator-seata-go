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

import React, { useState, useRef, useEffect } from 'react';
import { Send, Trash2, Download, User, Bot, Loader2, Wifi, WifiOff, MessageCircle, ArrowRight, Lightbulb, AlertTriangle, Link, RefreshCw, FileText } from 'lucide-react';
import { useWebSocket } from '../hooks/useWebSocket';
import type { ChatMessage } from '../types';
import MarkdownRenderer from './MarkdownRenderer';
import './ChatInterface.css';

interface ChatInterfaceProps {
  onConnectionChange: (connected: boolean) => void;
}

const ChatInterface: React.FC<ChatInterfaceProps> = ({ onConnectionChange }) => {
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      id: '1',
      role: 'assistant',
      content: '欢迎使用 Seata Go Transaction Agent！我将帮助您设计分布式事务工作流。\n\n您可以描述您的业务场景，我会为您生成相应的Saga工作流设计。',
      timestamp: new Date(),
    }
  ]);
  const [input, setInput] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  
  const { 
    isConnected, 
    sendMessage, 
    clearChat,
    lastMessage,
    currentResponse
  } = useWebSocket();

  useEffect(() => {
    onConnectionChange(isConnected);
  }, [isConnected, onConnectionChange]);

  useEffect(() => {
    if (lastMessage) {
      if (lastMessage.type === 'agent_response') {
        setIsTyping(false);
        const newMessage: ChatMessage = {
          id: Date.now().toString(),
          role: 'assistant',
          content: currentResponse?.text || lastMessage.data.response || lastMessage.data.content || '收到响应',
          timestamp: new Date(),
          agentData: currentResponse || undefined
        };
        setMessages(prev => [...prev, newMessage]);
      } else if (lastMessage.type === 'typing') {
        setIsTyping(lastMessage.data?.isTyping || false);
      } else if (lastMessage.type === 'error') {
        setIsTyping(false);
        const errorMessage: ChatMessage = {
          id: Date.now().toString(),
          role: 'assistant',
          content: `错误：${lastMessage.data?.message || '未知错误'}`,
          timestamp: new Date(),
        };
        setMessages(prev => [...prev, errorMessage]);
      }
    }
  }, [lastMessage, currentResponse]);

  useEffect(() => {
    scrollToBottom();
  }, [messages, isTyping]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const handleSend = async () => {
    if (!input.trim() || !isConnected) return;

    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      role: 'user',
      content: input.trim(),
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);
    const messageContent = input.trim();
    setInput('');
    setIsTyping(true);

    try {
      await sendMessage(messageContent);
    } catch (error) {
      setIsTyping(false);
      const errorMessage: ChatMessage = {
        id: Date.now().toString(),
        role: 'assistant',
        content: '发送消息失败，请检查连接状态。',
        timestamp: new Date(),
      };
      setMessages(prev => [...prev, errorMessage]);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleClearChat = async () => {
    try {
      await clearChat();
      setMessages([{
        id: '1',
        role: 'assistant',
        content: '对话历史已清空，让我们重新开始吧！\n\n请描述您的业务场景，我将为您设计Saga工作流。',
        timestamp: new Date(),
      }]);
    } catch (error) {
      console.error('Clear chat failed:', error);
    }
  };

  const handleExportChat = () => {
    const chatData = {
      timestamp: new Date().toISOString(),
      messages: messages.map(msg => ({
        role: msg.role,
        content: msg.content,
        timestamp: msg.timestamp.toISOString(),
        agentData: msg.agentData
      }))
    };

    const blob = new Blob([JSON.stringify(chatData, null, 2)], {
      type: 'application/json'
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `chat-export-${new Date().toISOString().split('T')[0]}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const formatTimestamp = (timestamp: Date) => {
    return timestamp.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  const renderMessage = (message: ChatMessage) => (
    <div key={message.id} className={`message ${message.role}`}>
      <div className="message-avatar">
        {message.role === 'user' ? (
          <User size={18} />
        ) : (
          <Bot size={18} />
        )}
      </div>
      <div className="message-content">
        <div className="message-header">
          <span className="message-role">
            {message.role === 'user' ? '用户' : 'Transaction Agent'}
          </span>
          <span className="message-timestamp">
            {formatTimestamp(message.timestamp)}
          </span>
        </div>
        <div className="message-text">
          <MarkdownRenderer content={message.content} />
        </div>
        {message.agentData && (
          <div className="workflow-preview">
            <div className="preview-tabs">
              <span className="preview-tab active" style={{display: 'flex', alignItems: 'center', gap: '6px'}}>
                <RefreshCw size={16} />
                阶段信息
              </span>
            </div>
            <div className="preview-content">
              <div className="phase-info">
                <span className="phase-badge">第{message.agentData.phase}步</span>
                <span className="phase-description">
                  {message.agentData.phase === 1 && '需求分析'}
                  {message.agentData.phase === 2 && '事务分解'}
                  {message.agentData.phase === 3 && '依赖分析'}
                  {message.agentData.phase === 4 && '补偿设计'}
                  {message.agentData.phase === 5 && '异常处理'}
                  {message.agentData.phase === 6 && '方案输出'}
                </span>
              </div>
              {message.agentData.graph && (
                <div className="preview-stats">
                  <span style={{display: 'flex', alignItems: 'center', gap: '6px'}}>
                    <Link size={16} /> {message.agentData.graph.nodes?.length || 0} 个节点
                  </span>
                  <span style={{display: 'flex', alignItems: 'center', gap: '6px'}}>
                    <ArrowRight size={16} /> {message.agentData.graph.edges?.length || 0} 条边
                  </span>
                </div>
              )}
              {message.agentData.seata_json && (
                <div className="preview-stats">
                  <span style={{display: 'flex', alignItems: 'center', gap: '6px'}}>
                    <FileText size={16} /> Seata 配置已生成
                  </span>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );

  return (
    <div className="chat-interface">
      <div className="chat-header">
        <div className="chat-title">
          <h2><MessageCircle size={24} /> 智能对话</h2>
          <div className="connection-indicator">
            {isConnected ? (
              <Wifi size={18} className="connected" />
            ) : (
              <WifiOff size={18} className="disconnected" />
            )}
            <span>{isConnected ? '已连接' : '未连接'}</span>
          </div>
        </div>
        <div className="chat-actions">
          <button 
            onClick={handleExportChat}
            className="action-button"
            title="导出对话"
          >
            <Download size={18} />
          </button>
          <button 
            onClick={handleClearChat}
            className="action-button"
            title="清空对话"
            disabled={!isConnected}
          >
            <Trash2 size={18} />
          </button>
        </div>
      </div>

      <div className="chat-messages">
        {messages.map(renderMessage)}
        {isTyping && (
          <div className="message assistant typing">
            <div className="message-avatar">
              <Bot size={18} />
            </div>
            <div className="message-content">
              <div className="message-header">
                <span className="message-role">Transaction Agent</span>
                <span className="message-timestamp">正在输入...</span>
              </div>
              <div className="typing-indicator">
                <Loader2 size={18} className="spinning" />
                <span>正在思考工作流设计...</span>
              </div>
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="chat-input-area">
        <div className="chat-input-container">
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder={isConnected ? "描述您的业务场景，我将为您设计Saga工作流..." : "请先连接到服务器"}
            className="chat-input"
            rows={3}
            disabled={!isConnected}
          />
          <button 
            onClick={handleSend}
            className="send-button"
            disabled={!input.trim() || !isConnected || isTyping}
          >
            <Send size={20} />
          </button>
        </div>
        <div className="input-hint">
          <span style={{display: 'flex', alignItems: 'center', gap: '6px'}}>
            <Lightbulb size={16} /> 按 Enter 发送消息，Shift + Enter 换行
          </span>
          {!isConnected && (
            <span className="connection-warning" style={{display: 'flex', alignItems: 'center', gap: '6px'}}>
              <AlertTriangle size={16} /> 未连接到服务器
            </span>
          )}
        </div>
      </div>
    </div>
  );
};

export default ChatInterface;