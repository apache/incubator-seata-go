import React, {useEffect, useState} from 'react';
import {Link} from 'react-router-dom';
import {Activity, ExternalLink, Heart, Pause, Play, Plus, RefreshCw, Search, Trash2, Users} from 'lucide-react';
import AgentHubAPI from '../services/api';
import {RegisteredAgent} from '../types';
import {formatDateTime, formatRelativeTime, getStatusColor, getStatusText, isAgentOnline} from '../utils';

const AgentList: React.FC = () => {
    const [agents, setAgents] = useState<RegisteredAgent[]>([]);
    const [filteredAgents, setFilteredAgents] = useState<RegisteredAgent[]>([]);
    const [loading, setLoading] = useState(true);
    const [searchQuery, setSearchQuery] = useState('');
    const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all');
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        loadAgents();
    }, []);

    useEffect(() => {
        filterAgents();
    }, [agents, searchQuery, statusFilter]);

    const loadAgents = async () => {
        try {
            setLoading(true);
            const response = await AgentHubAPI.listAgents();
            if (response.success && response.data) {
                setAgents(response.data);
            } else {
                setError(response.error?.error || 'Failed to load agents');
            }
        } catch (err) {
            setError('Network error while loading agents');
        } finally {
            setLoading(false);
        }
    };

    const filterAgents = () => {
        let filtered = agents;

        // 搜索过滤
        if (searchQuery) {
            const query = searchQuery.toLowerCase();
            filtered = filtered.filter(agent =>
                agent.agent_card.name.toLowerCase().includes(query) ||
                agent.agent_card.description.toLowerCase().includes(query) ||
                agent.id.toLowerCase().includes(query) ||
                agent.agent_card.skills.some(skill =>
                    skill.name.toLowerCase().includes(query) ||
                    skill.description.toLowerCase().includes(query) ||
                    skill.tags.some(tag => tag.toLowerCase().includes(query))
                )
            );
        }

        // 状态过滤
        if (statusFilter !== 'all') {
            if (statusFilter === 'active') {
                filtered = filtered.filter(isAgentOnline);
            } else {
                filtered = filtered.filter(agent => !isAgentOnline(agent));
            }
        }

        setFilteredAgents(filtered);
    };

    const handleStatusChange = async (agentId: string, newStatus: string) => {
        try {
            const response = await AgentHubAPI.updateAgentStatus(agentId, newStatus);
            if (response.success) {
                setAgents(prev => prev.map(agent =>
                    agent.id === agentId
                        ? {...agent, status: newStatus}
                        : agent
                ));
            } else {
                alert('更新状态失败: ' + (response.error?.error || 'Unknown error'));
            }
        } catch (error) {
            alert('更新状态失败: 网络错误');
        }
    };

    const handleDelete = async (agentId: string) => {
        if (!window.confirm('确定要删除这个Agent吗？此操作无法撤销。')) {
            return;
        }

        try {
            const response = await AgentHubAPI.removeAgent(agentId);
            if (response.success) {
                setAgents(prev => prev.filter(agent => agent.id !== agentId));
            } else {
                alert('删除失败: ' + (response.error?.error || 'Unknown error'));
            }
        } catch (error) {
            alert('删除失败: 网络错误');
        }
    };

    const handleHeartbeat = async (agentId: string) => {
        try {
            const response = await AgentHubAPI.sendHeartbeat(agentId);
            if (response.success) {
                // 更新最后心跳时间
                setAgents(prev => prev.map(agent =>
                    agent.id === agentId
                        ? {...agent, last_seen: new Date().toISOString()}
                        : agent
                ));
            } else {
                alert('发送心跳失败: ' + (response.error?.error || 'Unknown error'));
            }
        } catch (error) {
            alert('发送心跳失败: 网络错误');
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

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="md:flex md:items-center md:justify-between">
                <div className="flex-1 min-w-0">
                    <h1 className="text-2xl font-bold leading-7 text-gray-900 sm:text-3xl">
                        Agent 管理
                    </h1>
                    <p className="mt-1 text-sm text-gray-500">
                        管理所有注册的Agent，共 {agents.length} 个
                    </p>
                </div>
                <div className="mt-4 flex md:mt-0 md:ml-4 space-x-2">
                    <button
                        onClick={loadAgents}
                        className="btn-secondary"
                    >
                        <RefreshCw className="h-4 w-4 mr-2"/>
                        刷新
                    </button>
                    <Link to="/register" className="btn-primary">
                        <Plus className="h-4 w-4 mr-2"/>
                        注册Agent
                    </Link>
                </div>
            </div>

            {/* Filters */}
            <div className="flex flex-col sm:flex-row gap-4">
                <div className="relative flex-1">
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                        <Search className="h-5 w-5 text-gray-400"/>
                    </div>
                    <input
                        type="text"
                        placeholder="搜索Agent名称、描述或技能..."
                        className="input-field pl-10"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                    />
                </div>
                <div className="w-full sm:w-48">
                    <select
                        className="input-field"
                        value={statusFilter}
                        onChange={(e) => setStatusFilter(e.target.value as any)}
                    >
                        <option value="all">所有状态</option>
                        <option value="active">在线</option>
                        <option value="inactive">离线</option>
                    </select>
                </div>
            </div>

            {/* Error message */}
            {error && (
                <div className="bg-error-50 border border-error-200 rounded-md p-4">
                    <div className="text-sm text-error-700">{error}</div>
                </div>
            )}

            {/* Agent list */}
            {filteredAgents.length === 0 ? (
                <div className="card text-center py-12">
                    <Users className="mx-auto h-12 w-12 text-gray-400"/>
                    <h3 className="mt-2 text-sm font-medium text-gray-900">
                        {agents.length === 0 ? '暂无Agent' : '无匹配结果'}
                    </h3>
                    <p className="mt-1 text-sm text-gray-500">
                        {agents.length === 0 ? '开始注册您的第一个Agent' : '尝试调整搜索条件'}
                    </p>
                    {agents.length === 0 && (
                        <div className="mt-6">
                            <Link to="/register" className="btn-primary">
                                <Plus className="h-4 w-4 mr-2"/>
                                注册Agent
                            </Link>
                        </div>
                    )}
                </div>
            ) : (
                <div className="card overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-gray-200">
                            <thead className="bg-gray-50">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    Agent
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    状态
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    技能
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    最后心跳
                                </th>
                                <th className="relative px-6 py-3">
                                    <span className="sr-only">操作</span>
                                </th>
                            </tr>
                            </thead>
                            <tbody className="bg-white divide-y divide-gray-200">
                            {filteredAgents.map((agent) => (
                                <tr key={agent.id} className="hover:bg-gray-50">
                                    <td className="px-6 py-4 whitespace-nowrap">
                                        <div className="flex items-center">
                                            <div className="flex-shrink-0 h-10 w-10">
                                                <div
                                                    className="h-10 w-10 rounded-full bg-primary-100 flex items-center justify-center">
                            <span className="text-sm font-medium text-primary-600">
                              {agent.agent_card.name.charAt(0).toUpperCase()}
                            </span>
                                                </div>
                                            </div>
                                            <div className="ml-4">
                                                <div className="text-sm font-medium text-gray-900">
                                                    <Link
                                                        to={`/agents/${agent.id}`}
                                                        className="hover:text-primary-600"
                                                    >
                                                        {agent.agent_card.name}
                                                    </Link>
                                                </div>
                                                <div className="text-sm text-gray-500">
                                                    {agent.agent_card.description}
                                                </div>
                                                <div className="text-xs text-gray-400 mt-1">
                                                    {agent.host}:{agent.port} • v{agent.agent_card.version}
                                                </div>
                                            </div>
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap">
                                        <div className="flex items-center">
                                            <Activity className={`h-4 w-4 mr-2 ${
                                                isAgentOnline(agent) ? 'text-success-500' : 'text-gray-400'
                                            }`}/>
                                            <span
                                                className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusColor(agent.status)}`}>
                          {getStatusText(agent.status)}
                        </span>
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap">
                                        <div className="text-sm text-gray-900">
                                            {agent.agent_card.skills.length} 个技能
                                        </div>
                                        <div className="text-xs text-gray-500">
                                            {agent.agent_card.skills.slice(0, 2).map(skill => skill.name).join(', ')}
                                            {agent.agent_card.skills.length > 2 && '...'}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        <div>{formatRelativeTime(agent.last_seen)}</div>
                                        <div className="text-xs text-gray-400">
                                            {formatDateTime(agent.last_seen)}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                        <div className="flex items-center justify-end space-x-2">
                                            <button
                                                onClick={() => handleHeartbeat(agent.id)}
                                                className="text-gray-400 hover:text-primary-600 p-1"
                                                title="发送心跳"
                                            >
                                                <Heart className="h-4 w-4"/>
                                            </button>
                                            <a
                                                href={agent.agent_card.url}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className="text-gray-400 hover:text-primary-600 p-1"
                                                title="打开Agent URL"
                                            >
                                                <ExternalLink className="h-4 w-4"/>
                                            </a>
                                            <button
                                                onClick={() => handleStatusChange(
                                                    agent.id,
                                                    agent.status === 'active' ? 'inactive' : 'active'
                                                )}
                                                className={`p-1 ${
                                                    agent.status === 'active'
                                                        ? 'text-gray-400 hover:text-warning-600'
                                                        : 'text-gray-400 hover:text-success-600'
                                                }`}
                                                title={agent.status === 'active' ? '暂停' : '激活'}
                                            >
                                                {agent.status === 'active' ?
                                                    <Pause className="h-4 w-4"/> :
                                                    <Play className="h-4 w-4"/>
                                                }
                                            </button>
                                            <button
                                                onClick={() => handleDelete(agent.id)}
                                                className="text-gray-400 hover:text-error-600 p-1"
                                                title="删除"
                                            >
                                                <Trash2 className="h-4 w-4"/>
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}
        </div>
    );
};

export default AgentList;