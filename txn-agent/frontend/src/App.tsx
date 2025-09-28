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
