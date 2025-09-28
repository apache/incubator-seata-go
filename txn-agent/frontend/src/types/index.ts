// WebSocket 消息类型
export interface WebSocketMessage {
  type: 'user_input' | 'agent_response' | 'typing' | 'error' | 'clear_chat';
  data: any;
  timestamp?: string;
}

// Agent 响应数据结构
export interface AgentResponse {
  text: string;
  graph?: ReactFlowGraph;
  seata_json?: SeataWorkflow;
  phase: number;
}

// React Flow 图形结构
export interface ReactFlowGraph {
  nodes: ReactFlowNode[];
  edges: ReactFlowEdge[];
}

export interface ReactFlowNode {
  id: string;
  type: 'default' | 'input' | 'output';
  position: { x: number; y: number };
  data: { label: string };
  style?: {
    background?: string;
    color?: string;
    border?: string;
  };
}

export interface ReactFlowEdge {
  id: string;
  source: string;
  target: string;
  label?: string;
  type?: 'default' | 'step' | 'straight' | 'smoothstep';
  style?: {
    stroke?: string;
    strokeWidth?: number;
  };
}

// Seata 工作流定义
export interface SeataWorkflow {
  Name: string;
  Comment: string;
  StartState: string;
  Version: string;
  States: Record<string, any>;
  IsRetryPersistModeUpdate?: boolean;
  IsCompensatePersistModeUpdate?: boolean;
}

// 任务阶段定义
export interface TaskPhase {
  id: number;
  name: string;
  description: string;
  status: 'pending' | 'in-progress' | 'completed';
}

// 聊天消息
export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  agentData?: AgentResponse;
}

// 主题类型
export type Theme = 'dark' | 'light' | 'neon';

// WebSocket 上下文类型
export interface WebSocketContextType {
  isConnected: boolean;
  lastMessage: WebSocketMessage | null;
  currentResponse: AgentResponse | null;
  sendMessage: (message: string) => Promise<void>;
  clearChat: () => Promise<void>;
  reconnect: () => void;
}

// 主题上下文类型
export interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}