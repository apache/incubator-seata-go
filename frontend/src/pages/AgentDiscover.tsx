import React, { useState } from 'react';
import { 
  Search, 
  Zap, 
  ExternalLink,
  Copy,
  AlertCircle,
  CheckCircle
} from 'lucide-react';
import AgentHubAPI from '../services/api';
import { DiscoverRequest, DiscoverResponse } from '../types';
import { copyToClipboard } from '../utils';

const AgentDiscover: React.FC = () => {
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<DiscoverResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const examples = [
    { skill: 'sentiment_analysis', description: '情感分析技能' },
    { skill: 'keyword_extraction', description: '关键词提取技能' },
    { skill: 'data_visualization', description: '数据可视化技能' }
  ];

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!query.trim()) {
      setError('请输入技能ID');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      setResult(null);

      const response = await AgentHubAPI.discoverAgents({ query: query.trim() });
      
      if (response.success && response.data) {
        setResult(response.data);
      } else {
        setError(response.error?.error || 'Discovery failed');
      }
    } catch (err) {
      setError('Network error during discovery');
    } finally {
      setLoading(false);
    }
  };

  const handleCopyUrl = async (url: string) => {
    const success = await copyToClipboard(url);
    if (success) {
      alert('URL已复制到剪贴板');
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <div className="flex items-center">
          <Search className="h-8 w-8 text-primary-500 mr-3" />
          <div>
            <h1 className="text-3xl font-bold text-gray-900">能力发现</h1>
            <p className="mt-1 text-gray-600">通过技能ID查找对应的Agent服务地址</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Search form */}
        <div className="lg:col-span-2">
          <div className="card">
            <form onSubmit={handleSearch} className="space-y-4">
              <div>
                <label htmlFor="skill-id" className="block text-sm font-medium text-gray-700 mb-2">
                  技能ID
                </label>
                <div className="relative">
                  <input
                    id="skill-id"
                    type="text"
                    className="input-field pr-10"
                    placeholder="例如: sentiment_analysis"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                  <div className="absolute inset-y-0 right-0 pr-3 flex items-center">
                    <Search className="h-4 w-4 text-gray-400" />
                  </div>
                </div>
                <p className="mt-1 text-xs text-gray-500">
                  输入您要查找的技能ID，系统将返回提供该技能的Agent服务地址
                </p>
              </div>

              {error && (
                <div className="bg-error-50 border border-error-200 rounded-md p-3">
                  <div className="flex">
                    <AlertCircle className="h-4 w-4 text-error-400 mt-0.5 mr-2" />
                    <span className="text-sm text-error-700">{error}</span>
                  </div>
                </div>
              )}

              <button
                type="submit"
                disabled={loading || !query.trim()}
                className="btn-primary w-full"
              >
                {loading ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    搜索中...
                  </>
                ) : (
                  <>
                    <Zap className="h-4 w-4 mr-2" />
                    查找Agent
                  </>
                )}
              </button>
            </form>
          </div>

          {/* Results */}
          {result && (
            <div className="card">
              <h2 className="text-lg font-medium text-gray-900 mb-4">发现结果</h2>
              
              {result.agents.length === 0 ? (
                <div className="text-center py-8">
                  <Search className="mx-auto h-12 w-12 text-gray-400" />
                  <h3 className="mt-2 text-sm font-medium text-gray-900">未找到匹配的Agent</h3>
                  <p className="mt-1 text-sm text-gray-500">
                    技能 "{query}" 未在任何Agent中注册
                  </p>
                </div>
              ) : (
                <div className="space-y-4">
                  {result.agents.map((agent, index) => (
                    <div key={index} className="border border-success-200 bg-success-50 rounded-lg p-4">
                      <div className="flex items-start justify-between mb-3">
                        <div>
                          <h3 className="text-sm font-medium text-success-900">
                            {agent.name}
                          </h3>
                          <p className="text-xs text-success-700 mt-1">
                            {agent.description}
                          </p>
                        </div>
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-success-200 text-success-800">
                          找到匹配
                        </span>
                      </div>
                      
                      <div className="space-y-2">
                        <div className="flex items-center justify-between">
                          <span className="text-xs font-medium text-success-900">Agent URL:</span>
                          <div className="flex items-center space-x-2">
                            <code className="text-xs bg-white px-2 py-1 rounded border text-gray-800">
                              {agent.url}
                            </code>
                            <button
                              onClick={() => handleCopyUrl(agent.url)}
                              className="text-success-600 hover:text-success-700"
                              title="复制URL"
                            >
                              <Copy className="h-3 w-3" />
                            </button>
                            <a
                              href={agent.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-success-600 hover:text-success-700"
                              title="打开链接"
                            >
                              <ExternalLink className="h-3 w-3" />
                            </a>
                          </div>
                        </div>
                        
                        <div className="flex items-center justify-between">
                          <span className="text-xs font-medium text-success-900">版本:</span>
                          <span className="text-xs text-success-700">v{agent.version}</span>
                        </div>

                        <div className="flex items-center justify-between">
                          <span className="text-xs font-medium text-success-900">总技能数:</span>
                          <span className="text-xs text-success-700">{agent.skills.length} 个</span>
                        </div>

                        {agent.capabilities && (
                          <div className="mt-3 pt-3 border-t border-success-200">
                            <div className="text-xs text-success-700">
                              <strong>支持能力:</strong>
                              {agent.capabilities.streaming && ' 流式处理'}
                              {agent.capabilities.pushNotifications && ' 推送通知'}
                              {agent.capabilities.stateTransitionHistory && ' 状态历史'}
                            </div>
                          </div>
                        )}

                        {/* Show matching skills */}
                        <div className="mt-3 pt-3 border-t border-success-200">
                          <div className="text-xs text-success-700 mb-2">
                            <strong>匹配的技能:</strong>
                          </div>
                          <div className="grid grid-cols-1 gap-2">
                            {agent.skills
                              .filter(skill => skill.id.toLowerCase().includes(query.toLowerCase()) || 
                                             skill.name.toLowerCase().includes(query.toLowerCase()))
                              .map(skill => (
                              <div key={skill.id} className="bg-white rounded p-2 border">
                                <div className="flex items-center justify-between">
                                  <span className="text-xs font-medium text-gray-900">{skill.name}</span>
                                  <span className="text-xs text-gray-500">({skill.id})</span>
                                </div>
                                <p className="text-xs text-gray-600 mt-1">{skill.description}</p>
                                {skill.tags.length > 0 && (
                                  <div className="flex flex-wrap gap-1 mt-1">
                                    {skill.tags.map(tag => (
                                      <span key={tag} className="inline-flex items-center px-1 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
                                        {tag}
                                      </span>
                                    ))}
                                  </div>
                                )}
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Examples */}
        <div>
          <div className="card">
            <h3 className="text-lg font-medium text-gray-900 mb-4">示例技能ID</h3>
            <div className="space-y-3">
              {examples.map((example, index) => (
                <button
                  key={index}
                  onClick={() => setQuery(example.skill)}
                  className="w-full text-left p-3 border border-gray-200 rounded-lg hover:border-primary-300 hover:bg-primary-50 transition-colors duration-200"
                >
                  <h4 className="text-sm font-medium text-gray-900 mb-1">
                    {example.skill}
                  </h4>
                  <p className="text-xs text-gray-600">
                    {example.description}
                  </p>
                </button>
              ))}
            </div>

            <div className="mt-6 p-3 bg-gray-50 rounded-lg">
              <h4 className="text-sm font-medium text-gray-900 mb-2">使用说明</h4>
              <ul className="text-xs text-gray-600 space-y-1">
                <li>• 输入精确的技能ID进行查找</li>
                <li>• 系统返回提供该技能的Agent信息</li>
                <li>• 可直接使用返回的URL调用Agent服务</li>
                <li>• 支持复制URL到剪贴板</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AgentDiscover;