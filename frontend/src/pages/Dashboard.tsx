import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { 
  Users, 
  Plus, 
  Search, 
  Brain,
  Activity,
  AlertCircle,
  TrendingUp
} from 'lucide-react';
import AgentHubAPI from '../services/api';
import { RegisteredAgent } from '../types';
import { isAgentOnline, getStatusText, formatRelativeTime } from '../utils';

const Dashboard: React.FC = () => {
  const [agents, setAgents] = useState<RegisteredAgent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAgents();
  }, []);

  const loadAgents = async () => {
    try {
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

  const stats = {
    total: agents.length,
    online: agents.filter(isAgentOnline).length,
    offline: agents.filter(agent => !isAgentOnline(agent)).length,
    totalSkills: agents.reduce((sum, agent) => sum + agent.agent_card.skills.length, 0)
  };

  const quickActions = [
    {
      name: '注册新Agent',
      description: '注册一个新的Agent到系统中',
      href: '/register',
      icon: Plus,
      color: 'bg-primary-500',
    },
    {
      name: '能力发现',
      description: '根据技能ID查找对应的Agent服务',
      href: '/discover',
      icon: Search,
      color: 'bg-success-500',
    },
    {
      name: '智能分析',
      description: '使用AI进行需求分析和Agent匹配',
      href: '/analyze',
      icon: Brain,
      color: 'bg-warning-500',
    },
  ];

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
      {/* Page header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">概览</h1>
        <p className="mt-2 text-gray-600">欢迎来到AgentHub智能Agent管理平台</p>
      </div>

      {/* Error message */}
      {error && (
        <div className="bg-error-50 border border-error-200 rounded-md p-4">
          <div className="flex">
            <AlertCircle className="h-5 w-5 text-error-400" />
            <div className="ml-3">
              <h3 className="text-sm font-medium text-error-800">错误</h3>
              <div className="mt-2 text-sm text-error-700">
                <p>{error}</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
        <div className="card">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Users className="h-8 w-8 text-primary-500" />
            </div>
            <div className="ml-5 w-0 flex-1">
              <dl>
                <dt className="text-sm font-medium text-gray-500 truncate">总Agent数</dt>
                <dd className="text-2xl font-bold text-gray-900">{stats.total}</dd>
              </dl>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Activity className="h-8 w-8 text-success-500" />
            </div>
            <div className="ml-5 w-0 flex-1">
              <dl>
                <dt className="text-sm font-medium text-gray-500 truncate">在线Agent</dt>
                <dd className="text-2xl font-bold text-success-600">{stats.online}</dd>
              </dl>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <AlertCircle className="h-8 w-8 text-gray-400" />
            </div>
            <div className="ml-5 w-0 flex-1">
              <dl>
                <dt className="text-sm font-medium text-gray-500 truncate">离线Agent</dt>
                <dd className="text-2xl font-bold text-gray-600">{stats.offline}</dd>
              </dl>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <TrendingUp className="h-8 w-8 text-warning-500" />
            </div>
            <div className="ml-5 w-0 flex-1">
              <dl>
                <dt className="text-sm font-medium text-gray-500 truncate">总技能数</dt>
                <dd className="text-2xl font-bold text-warning-600">{stats.totalSkills}</dd>
              </dl>
            </div>
          </div>
        </div>
      </div>

      {/* Quick actions */}
      <div>
        <h2 className="text-lg font-medium text-gray-900 mb-4">快速操作</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {quickActions.map((action) => {
            const Icon = action.icon;
            return (
              <Link
                key={action.name}
                to={action.href}
                className="relative group bg-white p-6 focus-within:ring-2 focus-within:ring-inset focus-within:ring-primary-500 rounded-lg border border-gray-200 hover:shadow-md transition-all duration-200"
              >
                <div>
                  <span className={`${action.color} rounded-lg inline-flex p-3 text-white group-hover:scale-110 transition-transform duration-200`}>
                    <Icon className="h-6 w-6" />
                  </span>
                </div>
                <div className="mt-4">
                  <h3 className="text-lg font-medium text-gray-900">
                    {action.name}
                  </h3>
                  <p className="mt-2 text-sm text-gray-500">
                    {action.description}
                  </p>
                </div>
              </Link>
            );
          })}
        </div>
      </div>

      {/* Recent agents */}
      <div>
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-medium text-gray-900">最近的Agent</h2>
          <Link
            to="/agents"
            className="text-sm font-medium text-primary-600 hover:text-primary-500"
          >
            查看全部
          </Link>
        </div>
        <div className="mt-4">
          {agents.length === 0 ? (
            <div className="card text-center py-12">
              <Users className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">暂无Agent</h3>
              <p className="mt-1 text-sm text-gray-500">开始注册您的第一个Agent吧</p>
              <div className="mt-6">
                <Link
                  to="/register"
                  className="btn-primary"
                >
                  <Plus className="h-4 w-4 mr-2" />
                  注册Agent
                </Link>
              </div>
            </div>
          ) : (
            <div className="card overflow-hidden">
              <ul className="divide-y divide-gray-200">
                {agents.slice(0, 5).map((agent) => (
                  <li key={agent.id} className="px-6 py-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center">
                        <div className="flex-shrink-0">
                          <div className={`h-10 w-10 rounded-full bg-primary-100 flex items-center justify-center`}>
                            <span className="text-sm font-medium text-primary-600">
                              {agent.agent_card.name.charAt(0).toUpperCase()}
                            </span>
                          </div>
                        </div>
                        <div className="ml-4">
                          <div className="flex items-center">
                            <Link
                              to={`/agents/${agent.id}`}
                              className="text-sm font-medium text-gray-900 hover:text-primary-600"
                            >
                              {agent.agent_card.name}
                            </Link>
                            <span className={`ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                              isAgentOnline(agent) ? 'bg-success-100 text-success-800' : 'bg-gray-100 text-gray-800'
                            }`}>
                              {getStatusText(agent.status)}
                            </span>
                          </div>
                          <div className="mt-1 text-sm text-gray-500">
                            {agent.agent_card.description}
                          </div>
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-sm text-gray-900">
                          {agent.agent_card.skills.length} 个技能
                        </div>
                        <div className="text-sm text-gray-500">
                          {formatRelativeTime(agent.last_seen)}
                        </div>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default Dashboard;