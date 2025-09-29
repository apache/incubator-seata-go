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

import React, {useEffect, useState} from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {
    Activity,
    ArrowLeft,
    Clock,
    Copy,
    ExternalLink,
    Globe,
    Heart,
    Pause,
    Play,
    Server,
    Settings,
    Tag,
    Trash2,
    User
} from 'lucide-react';
import AgentHubAPI from '../services/api';
import {RegisteredAgent} from '../types';
import {copyToClipboard, formatDateTime, formatRelativeTime, getStatusText, isAgentOnline} from '../utils';

const AgentDetail: React.FC = () => {
    const {id} = useParams<{ id: string }>();
    const navigate = useNavigate();
    const [agent, setAgent] = useState<RegisteredAgent | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (id) {
            loadAgent(id);
        }
    }, [id]);

    const loadAgent = async (agentId: string) => {
        try {
            setLoading(true);
            const response = await AgentHubAPI.getAgent(agentId);
            if (response.success && response.data) {
                setAgent(response.data);
            } else {
                setError(response.error?.error || 'Agent not found');
            }
        } catch (err) {
            setError('Network error while loading agent');
        } finally {
            setLoading(false);
        }
    };

    const handleStatusChange = async (newStatus: string) => {
        if (!agent) return;

        try {
            const response = await AgentHubAPI.updateAgentStatus(agent.id, newStatus);
            if (response.success) {
                setAgent(prev => prev ? {...prev, status: newStatus} : null);
            } else {
                alert('更新状态失败: ' + (response.error?.error || 'Unknown error'));
            }
        } catch (error) {
            alert('更新状态失败: 网络错误');
        }
    };

    const handleDelete = async () => {
        if (!agent) return;

        if (!window.confirm('确定要删除这个Agent吗？此操作无法撤销。')) {
            return;
        }

        try {
            const response = await AgentHubAPI.removeAgent(agent.id);
            if (response.success) {
                navigate('/agents');
            } else {
                alert('删除失败: ' + (response.error?.error || 'Unknown error'));
            }
        } catch (error) {
            alert('删除失败: 网络错误');
        }
    };

    const handleHeartbeat = async () => {
        if (!agent) return;

        try {
            const response = await AgentHubAPI.sendHeartbeat(agent.id);
            if (response.success) {
                setAgent(prev => prev ? {
                    ...prev,
                    last_seen: new Date().toISOString()
                } : null);
                alert('心跳发送成功');
            } else {
                alert('发送心跳失败: ' + (response.error?.error || 'Unknown error'));
            }
        } catch (error) {
            alert('发送心跳失败: 网络错误');
        }
    };

    const handleCopy = async (text: string) => {
        const success = await copyToClipboard(text);
        if (success) {
            alert('已复制到剪贴板');
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
                <span className="ml-2 text-gray-600">加载中...</span>
            </div>
        );
    }

    if (error || !agent) {
        return (
            <div className="text-center py-12">
                <div className="text-error-500 text-6xl mb-4">⚠️</div>
                <h1 className="text-2xl font-bold text-gray-900 mb-2">Agent未找到</h1>
                <p className="text-gray-600 mb-4">{error || 'The requested agent does not exist'}</p>
                <button
                    onClick={() => navigate('/agents')}
                    className="btn-primary"
                >
                    <ArrowLeft className="h-4 w-4 mr-2"/>
                    返回Agent列表
                </button>
            </div>
        );
    }

    const isOnline = isAgentOnline(agent);

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center">
                    <button
                        onClick={() => navigate('/agents')}
                        className="mr-4 p-2 text-gray-400 hover:text-gray-600"
                    >
                        <ArrowLeft className="h-5 w-5"/>
                    </button>
                    <div>
                        <h1 className="text-3xl font-bold text-gray-900">{agent.agent_card.name}</h1>
                        <p className="mt-1 text-gray-600">{agent.agent_card.description}</p>
                    </div>
                </div>

                <div className="flex items-center space-x-2">
                    <button
                        onClick={handleHeartbeat}
                        className="btn-secondary"
                        title="发送心跳"
                    >
                        <Heart className="h-4 w-4 mr-2"/>
                        心跳
                    </button>
                    <button
                        onClick={() => handleStatusChange(agent.status === 'active' ? 'inactive' : 'active')}
                        className={agent.status === 'active' ? 'btn-warning' : 'btn-success'}
                    >
                        {agent.status === 'active' ? (
                            <>
                                <Pause className="h-4 w-4 mr-2"/>
                                暂停
                            </>
                        ) : (
                            <>
                                <Play className="h-4 w-4 mr-2"/>
                                激活
                            </>
                        )}
                    </button>
                    <button
                        onClick={handleDelete}
                        className="btn-error"
                    >
                        <Trash2 className="h-4 w-4 mr-2"/>
                        删除
                    </button>
                </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Main info */}
                <div className="lg:col-span-2 space-y-6">
                    {/* Status */}
                    <div className="card">
                        <h2 className="text-lg font-medium text-gray-900 mb-4">状态信息</h2>
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                            <div className="text-center">
                                <div className="flex items-center justify-center mb-2">
                                    <Activity className={`h-8 w-8 ${isOnline ? 'text-success-500' : 'text-gray-400'}`}/>
                                </div>
                                <div className="text-sm text-gray-500">运行状态</div>
                                <div
                                    className={`text-sm font-medium ${isOnline ? 'text-success-600' : 'text-gray-600'}`}>
                                    {getStatusText(agent.status)}
                                </div>
                            </div>
                            <div className="text-center">
                                <div className="flex items-center justify-center mb-2">
                                    <Clock className="h-8 w-8 text-gray-400"/>
                                </div>
                                <div className="text-sm text-gray-500">最后心跳</div>
                                <div className="text-sm font-medium text-gray-900">
                                    {formatRelativeTime(agent.last_seen)}
                                </div>
                            </div>
                            <div className="text-center">
                                <div className="flex items-center justify-center mb-2">
                                    <User className="h-8 w-8 text-gray-400"/>
                                </div>
                                <div className="text-sm text-gray-500">注册时间</div>
                                <div className="text-sm font-medium text-gray-900">
                                    {formatRelativeTime(agent.registered_at)}
                                </div>
                            </div>
                            <div className="text-center">
                                <div className="flex items-center justify-center mb-2">
                                    <Settings className="h-8 w-8 text-gray-400"/>
                                </div>
                                <div className="text-sm text-gray-500">版本</div>
                                <div className="text-sm font-medium text-gray-900">
                                    v{agent.agent_card.version}
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Skills */}
                    <div className="card">
                        <h2 className="text-lg font-medium text-gray-900 mb-4">技能列表</h2>
                        {agent.agent_card.skills.length === 0 ? (
                            <p className="text-gray-500 text-center py-8">该Agent暂无注册技能</p>
                        ) : (
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                {agent.agent_card.skills.map((skill) => (
                                    <div key={skill.id} className="border border-gray-200 rounded-lg p-4">
                                        <div className="flex items-start justify-between mb-2">
                                            <h3 className="text-sm font-medium text-gray-900">{skill.name}</h3>
                                            <button
                                                onClick={() => handleCopy(skill.id)}
                                                className="text-gray-400 hover:text-gray-600"
                                                title="复制技能ID"
                                            >
                                                <Copy className="h-3 w-3"/>
                                            </button>
                                        </div>
                                        <p className="text-xs text-gray-600 mb-2">{skill.description}</p>
                                        <div className="text-xs text-gray-500 mb-2">
                                            ID: <code className="bg-gray-100 px-1 py-0.5 rounded">{skill.id}</code>
                                        </div>
                                        {skill.tags.length > 0 && (
                                            <div className="flex flex-wrap gap-1 mb-2">
                                                {skill.tags.map((tag) => (
                                                    <span
                                                        key={tag}
                                                        className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800"
                                                    >
                            <Tag className="h-2 w-2 mr-1"/>
                                                        {tag}
                          </span>
                                                ))}
                                            </div>
                                        )}
                                        {skill.examples && skill.examples.length > 0 && (
                                            <div className="text-xs text-gray-500">
                                                <strong>示例:</strong> {skill.examples.join(', ')}
                                            </div>
                                        )}
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Capabilities */}
                    <div className="card">
                        <h2 className="text-lg font-medium text-gray-900 mb-4">能力配置</h2>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <div className="flex items-center">
                                <div className={`w-3 h-3 rounded-full mr-2 ${
                                    agent.agent_card.capabilities.streaming ? 'bg-success-500' : 'bg-gray-300'
                                }`}></div>
                                <span className="text-sm text-gray-700">流式处理</span>
                            </div>
                            <div className="flex items-center">
                                <div className={`w-3 h-3 rounded-full mr-2 ${
                                    agent.agent_card.capabilities.pushNotifications ? 'bg-success-500' : 'bg-gray-300'
                                }`}></div>
                                <span className="text-sm text-gray-700">推送通知</span>
                            </div>
                            <div className="flex items-center">
                                <div className={`w-3 h-3 rounded-full mr-2 ${
                                    agent.agent_card.capabilities.stateTransitionHistory ? 'bg-success-500' : 'bg-gray-300'
                                }`}></div>
                                <span className="text-sm text-gray-700">状态历史</span>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Sidebar */}
                <div className="space-y-6">
                    {/* Connection info */}
                    <div className="card">
                        <h3 className="text-lg font-medium text-gray-900 mb-4">连接信息</h3>
                        <div className="space-y-3">
                            <div>
                                <div className="flex items-center justify-between">
                                    <span className="text-sm font-medium text-gray-500">Agent URL</span>
                                    <div className="flex items-center space-x-1">
                                        <button
                                            onClick={() => handleCopy(agent.agent_card.url)}
                                            className="text-gray-400 hover:text-gray-600"
                                            title="复制URL"
                                        >
                                            <Copy className="h-3 w-3"/>
                                        </button>
                                        <a
                                            href={agent.agent_card.url}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-gray-400 hover:text-gray-600"
                                            title="打开链接"
                                        >
                                            <ExternalLink className="h-3 w-3"/>
                                        </a>
                                    </div>
                                </div>
                                <code className="text-xs text-gray-800 bg-gray-100 p-1 rounded block mt-1">
                                    {agent.agent_card.url}
                                </code>
                            </div>
                            <div>
                                <span className="text-sm font-medium text-gray-500">主机地址</span>
                                <div className="text-sm text-gray-900 mt-1 flex items-center">
                                    <Server className="h-3 w-3 mr-1"/>
                                    {agent.host}:{agent.port}
                                </div>
                            </div>
                            <div>
                                <span className="text-sm font-medium text-gray-500">Agent ID</span>
                                <div className="text-sm text-gray-900 mt-1 flex items-center justify-between">
                                    <code className="bg-gray-100 px-1 py-0.5 rounded text-xs">{agent.id}</code>
                                    <button
                                        onClick={() => handleCopy(agent.id)}
                                        className="text-gray-400 hover:text-gray-600"
                                        title="复制ID"
                                    >
                                        <Copy className="h-3 w-3"/>
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Additional info */}
                    <div className="card">
                        <h3 className="text-lg font-medium text-gray-900 mb-4">其他信息</h3>
                        <div className="space-y-3">
                            <div>
                                <span className="text-sm font-medium text-gray-500">输入模式</span>
                                <div className="flex flex-wrap gap-1 mt-1">
                                    {agent.agent_card.defaultInputModes.map((mode) => (
                                        <span key={mode}
                                              className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                      {mode}
                    </span>
                                    ))}
                                </div>
                            </div>
                            <div>
                                <span className="text-sm font-medium text-gray-500">输出模式</span>
                                <div className="flex flex-wrap gap-1 mt-1">
                                    {agent.agent_card.defaultOutputModes.map((mode) => (
                                        <span key={mode}
                                              className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                      {mode}
                    </span>
                                    ))}
                                </div>
                            </div>
                            {agent.agent_card.documentationUrl && (
                                <div>
                                    <span className="text-sm font-medium text-gray-500">文档链接</span>
                                    <div className="mt-1">
                                        <a
                                            href={agent.agent_card.documentationUrl}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-sm text-primary-600 hover:text-primary-700 flex items-center"
                                        >
                                            <Globe className="h-3 w-3 mr-1"/>
                                            查看文档
                                        </a>
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Timestamps */}
                    <div className="card">
                        <h3 className="text-lg font-medium text-gray-900 mb-4">时间信息</h3>
                        <div className="space-y-3">
                            <div>
                                <span className="text-sm font-medium text-gray-500">注册时间</span>
                                <div className="text-sm text-gray-900 mt-1">
                                    {formatDateTime(agent.registered_at)}
                                </div>
                            </div>
                            <div>
                                <span className="text-sm font-medium text-gray-500">最后更新</span>
                                <div className="text-sm text-gray-900 mt-1">
                                    {formatDateTime(agent.last_seen)}
                                </div>
                            </div>
                            <div>
                                <span className="text-sm font-medium text-gray-500">最后心跳</span>
                                <div className="text-sm text-gray-900 mt-1">
                                    {formatDateTime(agent.last_seen)}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default AgentDetail;