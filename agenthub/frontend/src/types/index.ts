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

// Agent相关类型定义
export interface AgentSkill {
    id: string;
    name: string;
    description: string;
    tags: string[];
    examples?: string[];
    inputModes?: string[];
    outputModes?: string[];
}

export interface AgentCapabilities {
    streaming?: boolean;
    pushNotifications?: boolean;
    stateTransitionHistory?: boolean;
}

export interface AgentCard {
    name: string;
    description: string;
    url: string;
    iconUrl?: string;
    version: string;
    documentationUrl?: string;
    capabilities: AgentCapabilities;
    defaultInputModes: string[];
    defaultOutputModes: string[];
    skills: AgentSkill[];
}

export interface RegisteredAgent {
    id: string;
    kind: string;
    version: string;
    agent_card: AgentCard;
    host: string;
    port: number;
    status: string;
    last_seen: string;
    registered_at: string;
}

// API请求/响应类型
export interface RegisterRequest {
    agent_card: AgentCard;
    host: string;
    port: number;
}

export interface RegisterResponse {
    success: boolean;
    message: string;
    agent_id?: string;
}

export interface DiscoverRequest {
    query: string;
}

export interface DiscoverResponse {
    agents: AgentCard[];
}

export interface ContextAnalysisRequest {
    need_description: string;
    user_context?: string;
}

export interface RouteResult {
    agent_url: string;
    skill_id: string;
    skill_name: string;
    agent_info?: AgentCard;
    agent_response?: any;
}

export interface AnalysisResult {
    required_skills?: string[];
    context_tags?: string[];
    suggested_workflow?: string;
}

export interface ContextAnalysisResponse {
    success: boolean;
    message: string;
    matched_skills?: AgentSkill[];
    route_result?: RouteResult;
    analysis_result?: AnalysisResult;
}

// API响应包装类型
export interface ApiResponse<T = any> {
    success: boolean;
    message?: string;
    data?: T;
    error?: {
        error: string;
        code: number;
    };
}

// 健康检查响应
export interface HealthResponse {
    component: string;
    status: string;
}