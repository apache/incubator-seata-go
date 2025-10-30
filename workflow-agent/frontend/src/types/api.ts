// API types based on the workflow-agent backend

export type SessionStatus =
  | 'pending'
  | 'analyzing'
  | 'discovering'
  | 'generating'
  | 'completed'
  | 'failed';

export interface Position {
  x: number;
  y: number;
}

export interface ReactFlowNode {
  id: string;
  type: 'input' | 'default' | 'output';
  position: Position;
  data: {
    label: string;
    [key: string]: any;
  };
  style?: Record<string, any>;
}

export interface ReactFlowEdge {
  id: string;
  source: string;
  target: string;
  type?: string;
  label?: string;
  style?: Record<string, any>;
}

export interface ReactFlowGraph {
  nodes: ReactFlowNode[];
  edges: ReactFlowEdge[];
}

export interface StateNode {
  Type: string;
  ServiceName?: string;
  ServiceMethod?: string;
  CompensateState?: string;
  IsForUpdate?: boolean;
  Input?: string[];
  Output?: Record<string, string>;
  Status?: Record<string, string>;
  Catch?: Array<{
    Exceptions: string[];
    Next: string;
  }>;
  Next?: string;
  Choices?: Array<{
    Expression: string;
    Next: string;
  }>;
  Default?: string;
  ErrorCode?: string;
  Message?: string;
}

export interface SeataStateMachine {
  Name: string;
  Comment: string;
  StartState: string;
  Version: string;
  States: Record<string, StateNode>;
}

export interface CapabilityRequirement {
  name: string;
  description: string;
  required: boolean;
}

export interface AgentCard {
  name: string;
  description: string;
  url: string;
  version: string;
  skills?: any[];
}

export interface DiscoveredCapability {
  requirement: CapabilityRequirement;
  agent?: AgentCard;
  found: boolean;
}

export interface WorkflowSnapshot {
  seata_workflow?: SeataStateMachine;
  react_flow?: ReactFlowGraph;
}

export interface ProgressEvent {
  timestamp: string;
  step: number;
  action: string;
  message: string;
  status: SessionStatus;
  capabilities?: DiscoveredCapability[];
  workflow?: WorkflowSnapshot;
}

export interface Session {
  id: string;
  description: string;
  status: SessionStatus;
  created_at: string;
  updated_at: string;
  event_count: number;
}

export interface SessionDetail extends Session {
  context?: Record<string, any>;
  events?: ProgressEvent[];
  result?: OrchestrationResult;
}

export interface OrchestrationResult {
  success: boolean;
  message: string;
  retry_count: number;
  capabilities: DiscoveredCapability[];
  workflow?: {
    seata_workflow: SeataStateMachine;
    react_flow: ReactFlowGraph;
  };
  error?: string;
}

// API Request/Response types
export interface CreateSessionRequest {
  description: string;
  context?: Record<string, any>;
}

export interface CreateSessionResponse {
  session_id: string;
  message: string;
}

export interface SessionListResponse {
  sessions: Session[];
  count: number;
}

export interface HealthCheckResponse {
  status: string;
  component: string;
}
