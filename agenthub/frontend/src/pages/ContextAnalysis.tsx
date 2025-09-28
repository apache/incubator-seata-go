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

import React, {useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {AlertCircle, Brain, CheckCircle, Copy, ExternalLink, Loader2, Target, Zap} from 'lucide-react';
import AgentHubAPI from '../services/api';
import {ContextAnalysisRequest, ContextAnalysisResponse} from '../types';
import {copyToClipboard} from '../utils';

const ContextAnalysis: React.FC = () => {
    const navigate = useNavigate();
    const [request, setRequest] = useState<ContextAnalysisRequest>({
        need_description: '',
        user_context: ''
    });
    const [response, setResponse] = useState<ContextAnalysisResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const examples = [
        {
            title: '情感分析需求',
            need_description: '我需要对用户输入的文本进行情感分析和关键词提取',
            user_context: '电商平台用户评论分析'
        },
        {
            title: '数据可视化需求',
            need_description: '需要分析销售数据，生成可视化图表',
            user_context: '月度销售报告生成'
        },
        {
            title: '文档处理需求',
            need_description: '我需要先处理用户上传的文档，然后根据内容生成摘要',
            user_context: '智能文档助手场景'
        }
    ];

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!request.need_description.trim()) {
            setError('请输入需求描述');
            return;
        }

        try {
            setLoading(true);
            setError(null);
            setResponse(null);

            const result = await AgentHubAPI.analyzeContext(request);

            if (result.success && result.data) {
                setResponse(result.data);
            } else {
                setError(result.error?.error || 'Analysis failed');
            }
        } catch (err) {
            setError('Network error during analysis');
        } finally {
            setLoading(false);
        }
    };

    const handleExampleClick = (example: typeof examples[0]) => {
        setRequest({
            need_description: example.need_description,
            user_context: example.user_context
        });
        setResponse(null);
        setError(null);
    };

    const handleCopyUrl = async (url: string) => {
        const success = await copyToClipboard(url);
        if (success) {
            // 这里可以添加一个toast通知
            alert('URL已复制到剪贴板');
        }
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <div className="flex items-center">
                    <Brain className="h-8 w-8 text-primary-500 mr-3"/>
                    <div>
                        <h1 className="text-3xl font-bold text-gray-900">智能分析</h1>
                        <p className="mt-1 text-gray-600">使用AI进行需求分析和Agent智能匹配</p>
                    </div>
                </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Input form */}
                <div className="lg:col-span-2">
                    <div className="card">
                        <div className="flex items-center mb-4">
                            <Target className="h-5 w-5 text-primary-500 mr-2"/>
                            <h2 className="text-lg font-medium text-gray-900">需求分析</h2>
                        </div>

                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div>
                                <label htmlFor="need_description" className="block text-sm font-medium text-gray-700">
                                    需求描述 *
                                </label>
                                <textarea
                                    id="need_description"
                                    rows={4}
                                    className="textarea-field mt-1"
                                    placeholder="详细描述您的需求，例如：我需要对用户评论进行情感分析..."
                                    value={request.need_description}
                                    onChange={(e) => setRequest(prev => ({...prev, need_description: e.target.value}))}
                                />
                            </div>

                            <div>
                                <label htmlFor="user_context" className="block text-sm font-medium text-gray-700">
                                    使用场景 (可选)
                                </label>
                                <input
                                    type="text"
                                    id="user_context"
                                    className="input-field mt-1"
                                    placeholder="例如：电商平台、客户服务、内容审核等"
                                    value={request.user_context}
                                    onChange={(e) => setRequest(prev => ({...prev, user_context: e.target.value}))}
                                />
                            </div>

                            {error && (
                                <div className="bg-error-50 border border-error-200 rounded-md p-3">
                                    <div className="flex">
                                        <AlertCircle className="h-4 w-4 text-error-400 mt-0.5 mr-2"/>
                                        <span className="text-sm text-error-700">{error}</span>
                                    </div>
                                </div>
                            )}

                            <button
                                type="submit"
                                disabled={loading || !request.need_description.trim()}
                                className="btn-primary w-full"
                            >
                                {loading ? (
                                    <>
                                        <Loader2 className="h-4 w-4 mr-2 animate-spin"/>
                                        AI分析中...
                                    </>
                                ) : (
                                    <>
                                        <Zap className="h-4 w-4 mr-2"/>
                                        开始分析
                                    </>
                                )}
                            </button>
                        </form>
                    </div>
                </div>

                {/* Examples */}
                <div>
                    <div className="card">
                        <h3 className="text-lg font-medium text-gray-900 mb-4">示例需求</h3>
                        <div className="space-y-3">
                            {examples.map((example, index) => (
                                <button
                                    key={index}
                                    onClick={() => handleExampleClick(example)}
                                    className="w-full text-left p-3 border border-gray-200 rounded-lg hover:border-primary-300 hover:bg-primary-50 transition-colors duration-200"
                                >
                                    <h4 className="text-sm font-medium text-gray-900 mb-1">
                                        {example.title}
                                    </h4>
                                    <p className="text-xs text-gray-600 mb-2">
                                        {example.need_description}
                                    </p>
                                    <p className="text-xs text-gray-500">
                                        场景: {example.user_context}
                                    </p>
                                </button>
                            ))}
                        </div>
                    </div>
                </div>
            </div>

            {/* Results */}
            {response && (
                <div className="card">
                    <div className="flex items-center mb-4">
                        {response.success ? (
                            <CheckCircle className="h-5 w-5 text-success-500 mr-2"/>
                        ) : (
                            <AlertCircle className="h-5 w-5 text-error-500 mr-2"/>
                        )}
                        <h2 className="text-lg font-medium text-gray-900">分析结果</h2>
                    </div>

                    {response.success || (!response.success && (response.message?.includes('没有匹配') || response.message?.includes('No matching') || response.message?.includes('no matching') || response.message?.includes('无相关技能') || response.message?.includes('skills found'))) ? (
                        <div className="space-y-6">
                            {/* Matched skills */}
                            {response.matched_skills && response.matched_skills.length > 0 && (
                                <div>
                                    <h3 className="text-md font-medium text-gray-900 mb-3">匹配的技能</h3>
                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                        {response.matched_skills.map((skill, index) => (
                                            <div key={skill.id} className="border border-gray-200 rounded-lg p-4">
                                                <div className="flex items-start justify-between">
                                                    <div>
                                                        <h4 className="text-sm font-medium text-gray-900">{skill.name}</h4>
                                                        <p className="text-xs text-gray-600 mt-1">{skill.description}</p>
                                                    </div>
                                                    {index === 0 && (
                                                        <span
                                                            className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-primary-100 text-primary-800">
                              推荐
                            </span>
                                                    )}
                                                </div>
                                                <div className="mt-3">
                                                    <div className="flex flex-wrap gap-1">
                                                        {skill.tags.map((tag) => (
                                                            <span
                                                                key={tag}
                                                                className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800"
                                                            >
                                {tag}
                              </span>
                                                        ))}
                                                    </div>
                                                </div>
                                                {skill.examples && skill.examples.length > 0 && (
                                                    <div className="mt-2">
                                                        <p className="text-xs text-gray-500">
                                                            示例: {skill.examples.join(', ')}
                                                        </p>
                                                    </div>
                                                )}
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* Route result */}
                            {response.route_result && (
                                <div>
                                    <h3 className="text-md font-medium text-gray-900 mb-3">推荐的Agent</h3>
                                    <div className="border border-success-200 bg-success-50 rounded-lg p-4">
                                        <div className="flex items-start justify-between mb-3">
                                            <div>
                                                <h4 className="text-sm font-medium text-success-900">
                                                    {response.route_result.agent_info?.name || 'Unknown Agent'}
                                                </h4>
                                                <p className="text-xs text-success-700 mt-1">
                                                    {response.route_result.agent_info?.description}
                                                </p>
                                            </div>
                                            <span
                                                className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-success-200 text-success-800">
                        最佳匹配
                      </span>
                                        </div>

                                        <div className="space-y-2">
                                            <div className="flex items-center justify-between">
                                                <span className="text-xs font-medium text-success-900">Agent URL:</span>
                                                <div className="flex items-center space-x-2">
                                                    <code
                                                        className="text-xs bg-white px-2 py-1 rounded border text-gray-800">
                                                        {response.route_result.agent_url}
                                                    </code>
                                                    <button
                                                        onClick={() => handleCopyUrl(response.route_result!.agent_url)}
                                                        className="text-success-600 hover:text-success-700"
                                                    >
                                                        <Copy className="h-3 w-3"/>
                                                    </button>
                                                    <a
                                                        href={response.route_result.agent_url}
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        className="text-success-600 hover:text-success-700"
                                                    >
                                                        <ExternalLink className="h-3 w-3"/>
                                                    </a>
                                                </div>
                                            </div>

                                            <div className="flex items-center justify-between">
                                                <span className="text-xs font-medium text-success-900">匹配技能:</span>
                                                <span className="text-xs text-success-700">
                          {response.route_result.skill_name} ({response.route_result.skill_id})
                        </span>
                                            </div>

                                            {response.route_result.agent_info && (
                                                <div className="mt-3 pt-3 border-t border-success-200">
                                                    <div className="text-xs text-success-700">
                                                        <strong>Agent详情:</strong> v{response.route_result.agent_info.version} •
                                                        支持 {response.route_result.agent_info.skills.length} 个技能 •
                                                        {response.route_result.agent_info.capabilities.streaming && ' 流式处理'}
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            )}

                            {/* No matching agents found */}
                            {(!response.matched_skills || response.matched_skills.length === 0) && !response.route_result && (
                                <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                                    <div className="flex">
                                        <Target className="h-5 w-5 text-orange-400 mr-3 flex-shrink-0 mt-0.5"/>
                                        <div>
                                            <h3 className="text-sm font-medium text-orange-800">暂无匹配的Agent</h3>
                                            <p className="mt-1 text-sm text-orange-700">
                                                AI已成功分析您的需求，但系统中暂时没有具备相关技能的Agent。
                                            </p>

                                            {/* AI Analysis Result */}
                                            {response.analysis_result && (
                                                <div
                                                    className="mt-3 p-3 bg-orange-100 rounded-lg border border-orange-300">
                                                    <h4 className="text-xs font-medium text-orange-800 mb-2 flex items-center">
                                                        <Brain className="h-3 w-3 mr-1"/>
                                                        AI需求分析结果
                                                    </h4>
                                                    <div className="text-xs text-orange-700 space-y-1">
                                                        {response.analysis_result.required_skills && response.analysis_result.required_skills.length > 0 && (
                                                            <div>
                                                                <span className="font-medium">识别的关键技能: </span>
                                                                <div className="flex flex-wrap gap-1 mt-1">
                                                                    {response.analysis_result.required_skills.map((skill, index) => (
                                                                        <span key={index}
                                                                              className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-orange-200 text-orange-800">
                                      {skill}
                                    </span>
                                                                    ))}
                                                                </div>
                                                            </div>
                                                        )}
                                                        {response.analysis_result.context_tags && response.analysis_result.context_tags.length > 0 && (
                                                            <div className="mt-2">
                                                                <span className="font-medium">相关标签: </span>
                                                                <span
                                                                    className="text-orange-600">{response.analysis_result.context_tags.join(', ')}</span>
                                                            </div>
                                                        )}
                                                        {response.analysis_result.suggested_workflow && (
                                                            <div className="mt-2">
                                                                <span className="font-medium">建议工作流: </span>
                                                                <span
                                                                    className="text-orange-600">{response.analysis_result.suggested_workflow}</span>
                                                            </div>
                                                        )}
                                                    </div>
                                                </div>
                                            )}

                                            <div className="mt-3">
                                                <p className="text-sm text-orange-700 mb-2">建议您可以：</p>
                                                <ul className="text-sm text-orange-700 space-y-1 ml-4">
                                                    <li>• 注册具备相关技能的新Agent</li>
                                                    <li>• 尝试调整需求描述，使用更通用的关键词</li>
                                                    <li>• 查看现有Agent列表，寻找相似功能</li>
                                                </ul>
                                            </div>
                                            <div className="mt-3 pt-3 border-t border-orange-200 flex space-x-3">
                                                <button
                                                    onClick={() => navigate('/register')}
                                                    className="text-xs bg-orange-100 hover:bg-orange-200 text-orange-800 px-3 py-1.5 rounded-md transition-colors duration-200"
                                                >
                                                    注册新Agent
                                                </button>
                                                <button
                                                    onClick={() => navigate('/agents')}
                                                    className="text-xs bg-orange-100 hover:bg-orange-200 text-orange-800 px-3 py-1.5 rounded-md transition-colors duration-200"
                                                >
                                                    查看现有Agent
                                                </button>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            )}
                        </div>
                    ) : (
                        <div className="bg-error-50 border border-error-200 rounded-lg p-4">
                            <div className="flex">
                                <AlertCircle className="h-5 w-5 text-error-400 mr-3 flex-shrink-0 mt-0.5"/>
                                <div>
                                    <h3 className="text-sm font-medium text-error-800">分析失败</h3>
                                    <p className="mt-1 text-sm text-error-700">{response.message}</p>
                                </div>
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
};

export default ContextAnalysis;