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

import { useState } from 'react';
import { ReactFlowProvider } from 'reactflow';
import { Palette, Workflow } from 'lucide-react';
import TaskPlanning from './components/TaskPlanning';
import WorkflowVisualization from './components/WorkflowVisualization';
import ChatInterface from './components/ChatInterface';
import { WebSocketProvider } from './contexts/WebSocketContext';
import { ThemeProvider, useTheme } from './contexts/ThemeContext';
import 'reactflow/dist/style.css';
import './App.css';

function AppContent() {
  const [isConnected, setIsConnected] = useState(false);
  const { theme, toggleTheme } = useTheme();

  const getThemeLabel = () => {
    switch(theme) {
      case 'dark': return '深色';
      case 'light': return '浅色';
      case 'neon': return '霓虹';
      default: return '深色';
    }
  };

  return (
    <div className="app">
      <header className="app-header">
        <h1><Workflow size={24} style={{display: 'inline', marginRight: '8px'}} />Seata Saga Transaction Agent</h1>
        <div className="header-controls">
          <button 
            onClick={toggleTheme}
            className="theme-toggle"
            title={`当前主题: ${getThemeLabel()}`}
          >
            <Palette size={16} />
            <span>{getThemeLabel()}</span>
          </button>
          <div className="connection-status">
            <span className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`} />
            {isConnected ? '已连接' : '未连接'}
          </div>
        </div>
      </header>
      
      <main className="app-main">
        <div className="left-panel">
          <TaskPlanning />
        </div>
        
        <div className="center-panel">
          <WorkflowVisualization />
        </div>
        
        <div className="right-panel">
          <ChatInterface onConnectionChange={setIsConnected} />
        </div>
      </main>
    </div>
  );
}

function App() {
  return (
    <ThemeProvider>
      <WebSocketProvider>
        <ReactFlowProvider>
          <AppContent />
        </ReactFlowProvider>
      </WebSocketProvider>
    </ThemeProvider>
  );
}

export default App;
