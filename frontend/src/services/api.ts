import axios from 'axios';
import {
  RegisterRequest,
  RegisterResponse,
  DiscoverRequest,
  DiscoverResponse,
  ContextAnalysisRequest,
  ContextAnalysisResponse,
  ApiResponse,
  RegisteredAgent,
  HealthResponse
} from '../types';

// 配置axios实例
const api = axios.create({
  baseURL: process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 响应拦截器
api.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API请求错误:', error);
    return Promise.reject(error);
  }
);

export class AgentHubAPI {
  // 健康检查
  static async healthCheck(): Promise<HealthResponse> {
    const response = await api.get<HealthResponse>('/health');
    return response.data;
  }

  // 注册Agent
  static async registerAgent(request: RegisterRequest): Promise<ApiResponse<RegisterResponse>> {
    const response = await api.post<ApiResponse<RegisterResponse>>('/agent/register', request);
    return response.data;
  }

  // 获取所有Agent
  static async listAgents(): Promise<ApiResponse<RegisteredAgent[]>> {
    const response = await api.get<ApiResponse<RegisteredAgent[]>>('/agents');
    return response.data;
  }

  // 获取特定Agent
  static async getAgent(id: string): Promise<ApiResponse<RegisteredAgent>> {
    const response = await api.get<ApiResponse<RegisteredAgent>>(`/agent/get?id=${id}`);
    return response.data;
  }

  // 发现Agent
  static async discoverAgents(request: DiscoverRequest): Promise<ApiResponse<DiscoverResponse>> {
    const response = await api.post<ApiResponse<DiscoverResponse>>('/agent/discover', request);
    return response.data;
  }

  // 更新Agent状态
  static async updateAgentStatus(id: string, status: string): Promise<ApiResponse> {
    const response = await api.put<ApiResponse>(`/agent/status?id=${id}`, { status });
    return response.data;
  }

  // 删除Agent
  static async removeAgent(id: string): Promise<ApiResponse> {
    const response = await api.delete<ApiResponse>(`/agent/remove?id=${id}`);
    return response.data;
  }

  // 发送心跳
  static async sendHeartbeat(id: string): Promise<ApiResponse> {
    const response = await api.post<ApiResponse>(`/agent/heartbeat?id=${id}`);
    return response.data;
  }

  // 动态上下文分析
  static async analyzeContext(request: ContextAnalysisRequest): Promise<ApiResponse<ContextAnalysisResponse>> {
    const response = await api.post<ApiResponse<ContextAnalysisResponse>>('/agent/analyze', request);
    return response.data;
  }
}

export default AgentHubAPI;