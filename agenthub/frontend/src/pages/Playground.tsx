import React, {useEffect, useRef, useState} from 'react';
import {
    AlertCircle,
    Bot,
    CheckCircle,
    Clock,
    Copy,
    Loader,
    MessageSquare,
    Play,
    Send,
    Settings,
    Trash2,
    Zap
} from 'lucide-react';
import AgentHubAPI from '../services/api';
import {RegisteredAgent} from '../types';
import {copyToClipboard} from '../utils';

interface Message {
    id: string;
    type: 'user' | 'agent' | 'system';
    content: string;
    timestamp: Date;
    agentName?: string;
    agentId?: string;
    duration?: number;
    error?: string;
}

interface PlaygroundSession {
    id: string;
    name: string;
    messages: Message[];
    selectedAgent?: RegisteredAgent;
    createdAt: Date;
}

const Playground: React.FC = () => {
    const [agents, setAgents] = useState<RegisteredAgent[]>([]);
    const [selectedAgent, setSelectedAgent] = useState<RegisteredAgent | null>(null);
    const [inputMessage, setInputMessage] = useState('');
    const [messages, setMessages] = useState<Message[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [sessions, setSessions] = useState<PlaygroundSession[]>([]);
    const [currentSession, setCurrentSession] = useState<PlaygroundSession | null>(null);
    const [showSettings, setShowSettings] = useState(false);
    const messagesEndRef = useRef<HTMLDivElement>(null);

    // 载入Agents列表
    useEffect(() => {
        const loadAgents = async () => {
            try {
                const response = await AgentHubAPI.listAgents();
                const agentsData = response.data || [];
                setAgents(agentsData);

                // 如果有可用Agent，默认选择第一个
                if (agentsData.length > 0) {
                    setSelectedAgent(agentsData[0]);
                }
            } catch (error) {
                console.error('Failed to load agents:', error);
                addSystemMessage('加载Agent列表失败，请检查连接');
            }
        };

        loadAgents();
        createNewSession();
    }, []);

    // 自动滚动到最新消息
    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({behavior: 'smooth'});
    }, [messages]);

    const addSystemMessage = (content: string, error?: string) => {
        const message: Message = {
            id: Date.now().toString(),
            type: 'system',
            content,
            timestamp: new Date(),
            error
        };
        setMessages(prev => [...prev, message]);
    };

    const addUserMessage = (content: string) => {
        const message: Message = {
            id: Date.now().toString(),
            type: 'user',
            content,
            timestamp: new Date()
        };
        setMessages(prev => [...prev, message]);
    };

    const addAgentMessage = (content: string, agentName?: string, agentId?: string, duration?: number, error?: string) => {
        const message: Message = {
            id: Date.now().toString(),
            type: 'agent',
            content,
            timestamp: new Date(),
            agentName,
            agentId,
            duration,
            error
        };
        setMessages(prev => [...prev, message]);
    };

    const createNewSession = () => {
        const session: PlaygroundSession = {
            id: Date.now().toString(),
            name: `会话 ${sessions.length + 1}`,
            messages: [],
            selectedAgent: selectedAgent || undefined,
            createdAt: new Date()
        };
        setSessions(prev => [session, ...prev]);
        setCurrentSession(session);
        setMessages([]);

        // 添加欢迎消息
        setTimeout(() => {
            addSystemMessage('欢迎使用AgentHub Playground! 🚀\n选择一个Agent并开始对话，或使用智能分析功能自动匹配最适合的Agent。');
        }, 100);
    };

    const handleSendMessage = async () => {
        if (!inputMessage.trim()) return;

        const userMessage = inputMessage.trim();
        setInputMessage('');
        addUserMessage(userMessage);
        setIsLoading(true);

        const startTime = Date.now();

        try {
            if (selectedAgent) {
                // 直接调用选中的Agent
                await simulateAgentCall(selectedAgent, userMessage, startTime);
            } else {
                // 使用智能分析功能
                await handleIntelligentRouting(userMessage, startTime);
            }
        } catch (error) {
            const duration = Date.now() - startTime;
            addAgentMessage(
                '抱歉，处理您的请求时发生了错误。',
                undefined,
                undefined,
                duration,
                error instanceof Error ? error.message : '未知错误'
            );
        } finally {
            setIsLoading(false);
        }
    };

    const handleIntelligentRouting = async (userMessage: string, startTime: number) => {
        try {
            // 使用上下文分析功能
            const analysisResponse = await AgentHubAPI.analyzeContext({
                need_description: userMessage,
                user_context: `用户正在Playground中测试功能，时间: ${new Date().toISOString()}`
            });

            const duration = Date.now() - startTime;

            if (analysisResponse.success && analysisResponse.data?.route_result) {
                // 成功找到匹配的Agent
                const routeResult = analysisResponse.data.route_result;
                const matchedAgent = routeResult.agent_info;

                if (matchedAgent) {
                    addAgentMessage(
                        `🎯 智能分析结果:\n找到最匹配的Agent: **${matchedAgent.name}**\n\n` +
                        `匹配的技能: ${routeResult.skill_name}\n` +
                        `技能ID: ${routeResult.skill_id}\n\n` +
                        `Agent服务地址: ${routeResult.agent_url}\n` +
                        `描述: ${matchedAgent.description}`,
                        '智能分析系统',
                        'ai-router',
                        duration
                    );

                    // 模拟调用匹配的Agent - 创建临时RegisteredAgent对象
                    const tempAgent: RegisteredAgent = {
                        id: matchedAgent.name,
                        kind: 'Agent',
                        version: matchedAgent.version,
                        agent_card: matchedAgent,
                        host: 'localhost',
                        port: 8080,
                        status: 'online',
                        last_seen: new Date().toISOString(),
                        registered_at: new Date().toISOString()
                    };

                    setTimeout(() => simulateAgentCall(tempAgent, userMessage, Date.now()), 500);
                } else {
                    addAgentMessage(
                        `🔍 找到匹配技能但Agent信息不完整\n技能: ${routeResult.skill_name}\nURL: ${routeResult.agent_url}`,
                        '智能分析系统',
                        'ai-router',
                        duration
                    );
                }
            } else {
                // 没有找到匹配的Agent，但显示AI分析结果
                const analysisResult = analysisResponse.data?.analysis_result;
                let analysisInfo = '';

                if (analysisResult) {
                    if (analysisResult.required_skills && analysisResult.required_skills.length > 0) {
                        analysisInfo += `\n🎯 AI识别的需求技能:\n${analysisResult.required_skills.map(skill => `• ${skill}`).join('\n')}`;
                    }

                    if (analysisResult.context_tags && analysisResult.context_tags.length > 0) {
                        analysisInfo += `\n\n🏷️ 相关标签:\n${analysisResult.context_tags.map(tag => `#${tag}`).join(' ')}`;
                    }

                    if (analysisResult.suggested_workflow) {
                        analysisInfo += `\n\n💡 建议的处理方式:\n${analysisResult.suggested_workflow}`;
                    }
                }

                addAgentMessage(
                    `❌ 智能分析结果:\n暂时没有找到能处理此请求的Agent。${analysisInfo}\n\n` +
                    `📋 建议: \n` +
                    `1. 根据上述分析，注册具有相关技能的Agent\n` +
                    `2. 尝试重新描述您的需求\n` +
                    `3. 查看现有Agent列表，寻找相似功能`,
                    '智能分析系统',
                    'ai-router',
                    duration
                );
            }
        } catch (error) {
            const duration = Date.now() - startTime;
            addAgentMessage(
                '智能分析服务暂时不可用，请选择具体的Agent进行测试。',
                '智能分析系统',
                'ai-router',
                duration,
                error instanceof Error ? error.message : '分析失败'
            );
        }
    };

    const simulateAgentCall = async (agent: RegisteredAgent, message: string, startTime: number) => {
        // 模拟Agent调用延迟
        await new Promise(resolve => setTimeout(resolve, 800 + Math.random() * 1200));

        const duration = Date.now() - startTime;

        // 模拟不同类型的响应
        const agentCard = agent.agent_card;
        const responses = [
            `✅ 处理完成!\n我是 ${agentCard.name}，已成功处理您的请求: "${message}"\n\n基于我的技能 [${agentCard.skills?.map(s => s.name).join(', ')}]，我认为这个任务很适合我处理。`,
            `🔧 正在处理中...\n${agentCard.name} 收到您的请求，正在使用相关技能进行处理。\n\n预计处理时间: ${Math.floor(Math.random() * 5) + 1}秒`,
            `📋 任务分析:\n请求内容: "${message}"\n推荐技能: ${agentCard.skills?.[0]?.name || '通用处理'}\n状态: 已完成\n\n如需进一步处理，请提供更多详细信息。`,
            `💡 智能建议:\n基于您的请求，我建议采用以下策略:\n1. 使用${agentCard.skills?.[0]?.name || '核心技能'}进行初步处理\n2. 结合上下文信息进行优化\n3. 返回结构化结果\n\n当前状态: 就绪`
        ];

        const response = responses[Math.floor(Math.random() * responses.length)];
        addAgentMessage(response, agentCard.name, agent.id, duration);
    };

    const clearMessages = () => {
        setMessages([]);
        addSystemMessage('对话历史已清空');
    };

    const copyMessage = (content: string) => {
        copyToClipboard(content);
        // TODO: 添加Toast通知
        console.log('消息已复制到剪贴板');
    };

    const formatDuration = (ms?: number) => {
        if (!ms) return '';
        if (ms < 1000) return `${ms}ms`;
        return `${(ms / 1000).toFixed(1)}s`;
    };

    return (
        <div className="h-full flex flex-col lg:flex-row bg-gray-50">
            {/* 左侧面板 - Agent选择和会话管理 */}
            <div className="lg:w-80 bg-white border-b lg:border-r lg:border-b-0 border-gray-200">
                <div className="p-4 border-b border-gray-200">
                    <div className="flex items-center justify-between mb-4">
                        <h2 className="text-lg font-semibold text-gray-900 flex items-center">
                            <MessageSquare className="h-5 w-5 mr-2 text-primary-500"/>
                            Playground
                        </h2>
                        <button
                            onClick={() => setShowSettings(!showSettings)}
                            className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100"
                        >
                            <Settings className="h-4 w-4"/>
                        </button>
                    </div>

                    <button
                        onClick={createNewSession}
                        className="w-full bg-primary-500 text-white px-4 py-2 rounded-lg hover:bg-primary-600 transition-colors flex items-center justify-center"
                    >
                        <Play className="h-4 w-4 mr-2"/>
                        新建会话
                    </button>
                </div>


                {/* 会话历史 */}
                <div className="p-4">
                    <h3 className="text-sm font-medium text-gray-700 mb-3">会话历史</h3>
                    <div className="space-y-2 max-h-40 overflow-y-auto">
                        {sessions.map((session) => (
                            <button
                                key={session.id}
                                onClick={() => {
                                    setCurrentSession(session);
                                    setMessages(session.messages);
                                    setSelectedAgent(session.selectedAgent || null);
                                }}
                                className={`w-full text-left p-3 rounded-lg border transition-colors ${
                                    currentSession?.id === session.id
                                        ? 'border-primary-200 bg-primary-50'
                                        : 'border-gray-200 hover:bg-gray-50'
                                }`}
                            >
                                <div className="font-medium text-sm">{session.name}</div>
                                <div className="text-xs text-gray-500 flex items-center mt-1">
                                    <Clock className="h-3 w-3 mr-1"/>
                                    {session.createdAt.toLocaleTimeString()}
                                </div>
                            </button>
                        ))}
                    </div>
                </div>
            </div>

            {/* 右侧面板 - 对话区域 */}
            <div className="flex-1 flex flex-col min-h-0">
                {/* 对话头部 */}
                <div className="bg-white border-b border-gray-200 p-4">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center">
                            {selectedAgent ? (
                                <>
                                    <Bot className="h-5 w-5 text-blue-500 mr-2"/>
                                    <div>
                                        <h3 className="font-medium text-gray-900">{selectedAgent.agent_card.name}</h3>
                                        <p className="text-sm text-gray-500">{selectedAgent.agent_card.description}</p>
                                    </div>
                                </>
                            ) : (
                                <>
                                    <Zap className="h-5 w-5 text-yellow-500 mr-2"/>
                                    <div>
                                        <h3 className="font-medium text-gray-900">智能路由模式</h3>
                                        <p className="text-sm text-gray-500">AI将自动为您选择最合适的Agent</p>
                                    </div>
                                </>
                            )}
                        </div>
                        <button
                            onClick={clearMessages}
                            className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100"
                            title="清空对话"
                        >
                            <Trash2 className="h-4 w-4"/>
                        </button>
                    </div>
                </div>

                {/* 消息区域 */}
                <div className="flex-1 overflow-y-auto p-4 space-y-4">
                    {messages.map((message) => (
                        <div
                            key={message.id}
                            className={`flex ${message.type === 'user' ? 'justify-end' : 'justify-start'}`}
                        >
                            <div
                                className={`max-w-lg px-4 py-2 rounded-lg relative group ${
                                    message.type === 'user'
                                        ? 'bg-primary-500 text-white'
                                        : message.type === 'system'
                                            ? 'bg-gray-100 text-gray-700 border-l-4 border-yellow-400'
                                            : 'bg-white border border-gray-200 text-gray-900'
                                }`}
                            >
                                <div className="flex items-start">
                                    <div className="flex-1">
                                        {message.type !== 'user' && (
                                            <div className="flex items-center mb-1">
                                                {message.type === 'system' ? (
                                                    <div
                                                        className="flex items-center text-sm font-medium text-gray-600">
                                                        <Settings className="h-3 w-3 mr-1"/>
                                                        系统
                                                    </div>
                                                ) : (
                                                    <div
                                                        className="flex items-center text-sm font-medium text-gray-600">
                                                        <Bot className="h-3 w-3 mr-1"/>
                                                        {message.agentName || 'Agent'}
                                                        {message.duration && (
                                                            <span
                                                                className="ml-2 text-xs bg-gray-100 text-gray-500 px-1 rounded">
                                {formatDuration(message.duration)}
                              </span>
                                                        )}
                                                        {message.error ? (
                                                            <AlertCircle className="h-3 w-3 ml-1 text-red-500"/>
                                                        ) : (
                                                            <CheckCircle className="h-3 w-3 ml-1 text-green-500"/>
                                                        )}
                                                    </div>
                                                )}
                                            </div>
                                        )}
                                        <div className="whitespace-pre-wrap">{message.content}</div>
                                        {message.error && (
                                            <div className="mt-2 text-xs text-red-500 bg-red-50 px-2 py-1 rounded">
                                                错误: {message.error}
                                            </div>
                                        )}
                                        <div className="text-xs opacity-70 mt-1">
                                            {message.timestamp.toLocaleTimeString()}
                                        </div>
                                    </div>
                                    <button
                                        onClick={() => copyMessage(message.content)}
                                        className="ml-2 p-1 opacity-0 group-hover:opacity-100 transition-opacity hover:bg-gray-200 rounded"
                                        title="复制消息"
                                    >
                                        <Copy className="h-3 w-3"/>
                                    </button>
                                </div>
                            </div>
                        </div>
                    ))}

                    {isLoading && (
                        <div className="flex justify-start">
                            <div className="bg-white border border-gray-200 rounded-lg px-4 py-2 flex items-center">
                                <Loader className="h-4 w-4 animate-spin mr-2"/>
                                <span className="text-gray-500">Agent正在处理中...</span>
                            </div>
                        </div>
                    )}
                    <div ref={messagesEndRef}/>
                </div>

                {/* 输入区域 */}
                <div className="bg-white border-t border-gray-200 p-4">
                    <div className="flex space-x-2">
                        <input
                            type="text"
                            value={inputMessage}
                            onChange={(e) => setInputMessage(e.target.value)}
                            onKeyPress={(e) => e.key === 'Enter' && !e.shiftKey && handleSendMessage()}
                            placeholder={
                                selectedAgent
                                    ? `与 ${selectedAgent.agent_card.name} 对话...`
                                    : "描述您的需求，AI将为您选择最合适的Agent..."
                            }
                            className="flex-1 border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                            disabled={isLoading}
                        />
                        <button
                            onClick={handleSendMessage}
                            disabled={!inputMessage.trim() || isLoading}
                            className="bg-primary-500 text-white px-4 py-2 rounded-lg hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            <Send className="h-4 w-4"/>
                        </button>
                    </div>

                    <div className="text-xs text-gray-500 mt-2">
                        {selectedAgent ? (
                            `当前选择: ${selectedAgent.agent_card.name} | 技能: ${selectedAgent.agent_card.skills?.map(s => s.name).join(', ') || '无'}`
                        ) : (
                            '智能路由模式 - AI将根据您的需求自动选择最合适的Agent'
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
};

export default Playground;