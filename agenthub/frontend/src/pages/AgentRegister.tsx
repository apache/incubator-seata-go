import React, {useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {AlertCircle, CheckCircle, FileText, Plus, Save, Upload} from 'lucide-react';
import AgentHubAPI from '../services/api';
import {AgentSkill, RegisterRequest} from '../types';
import {isValidPort, isValidUrl} from '../utils';

const AgentRegister: React.FC = () => {
    const navigate = useNavigate();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    const [formData, setFormData] = useState<RegisterRequest>({
        agent_card: {
            name: '',
            description: '',
            url: '',
            iconUrl: '',
            version: '1.0.0',
            documentationUrl: '',
            capabilities: {
                streaming: false,
                pushNotifications: false,
                stateTransitionHistory: false
            },
            defaultInputModes: ['text'],
            defaultOutputModes: ['json'],
            skills: []
        },
        host: 'localhost',
        port: 8081
    });

    const [currentSkill, setCurrentSkill] = useState<AgentSkill>({
        id: '',
        name: '',
        description: '',
        tags: [],
        examples: [],
        inputModes: [],
        outputModes: []
    });

    const [showSkillForm, setShowSkillForm] = useState(false);
    const [editingSkillIndex, setEditingSkillIndex] = useState<number | null>(null);

    const handleInputChange = (path: string, value: any) => {
        setFormData(prev => {
            const keys = path.split('.');
            const newData = {...prev};
            let current: any = newData;

            for (let i = 0; i < keys.length - 1; i++) {
                current = current[keys[i]];
            }
            current[keys[keys.length - 1]] = value;

            return newData;
        });
    };

    const addSkill = () => {
        if (!currentSkill.id || !currentSkill.name || !currentSkill.description) {
            setError('请填写技能的ID、名称和描述');
            return;
        }

        const skills = [...formData.agent_card.skills];

        if (editingSkillIndex !== null) {
            skills[editingSkillIndex] = {...currentSkill};
            setEditingSkillIndex(null);
        } else {
            // 检查ID是否重复
            if (skills.some(skill => skill.id === currentSkill.id)) {
                setError('技能ID已存在，请使用不同的ID');
                return;
            }
            skills.push({...currentSkill});
        }

        handleInputChange('agent_card.skills', skills);
        setCurrentSkill({
            id: '',
            name: '',
            description: '',
            tags: [],
            examples: [],
            inputModes: [],
            outputModes: []
        });
        setShowSkillForm(false);
        setError(null);
    };

    const editSkill = (index: number) => {
        setCurrentSkill({...formData.agent_card.skills[index]});
        setEditingSkillIndex(index);
        setShowSkillForm(true);
    };

    const removeSkill = (index: number) => {
        const skills = formData.agent_card.skills.filter((_, i) => i !== index);
        handleInputChange('agent_card.skills', skills);
    };

    const handleArrayInput = (value: string, setter: (arr: string[]) => void) => {
        const items = value.split(',').map(item => item.trim()).filter(item => item);
        setter(items);
    };

    const validateForm = (): string | null => {
        const {agent_card, host, port} = formData;

        if (!agent_card.name.trim()) return '请输入Agent名称';
        if (!agent_card.description.trim()) return '请输入Agent描述';
        if (!agent_card.url.trim()) return '请输入Agent URL';
        if (!isValidUrl(agent_card.url)) return 'Agent URL格式不正确';
        if (!agent_card.version.trim()) return '请输入版本号';
        if (!host.trim()) return '请输入主机地址';
        if (!isValidPort(port)) return '端口号必须在1-65535之间';

        return null;
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        const validationError = validateForm();
        if (validationError) {
            setError(validationError);
            return;
        }

        try {
            setLoading(true);
            setError(null);
            setSuccess(null);

            const response = await AgentHubAPI.registerAgent(formData);

            if (response.success) {
                setSuccess('Agent注册成功！');
                setTimeout(() => {
                    navigate('/agents');
                }, 2000);
            } else {
                setError(response.error?.error || 'Registration failed');
            }
        } catch (err) {
            setError('Network error during registration');
        } finally {
            setLoading(false);
        }
    };

    const loadTemplate = () => {
        setFormData({
            agent_card: {
                name: 'text-analyzer',
                description: '文本分析代理，提供情感分析和关键词提取服务',
                url: 'http://localhost:8081',
                iconUrl: '',
                version: '1.0.0',
                documentationUrl: '',
                capabilities: {
                    streaming: true,
                    pushNotifications: false,
                    stateTransitionHistory: false
                },
                defaultInputModes: ['text'],
                defaultOutputModes: ['json'],
                skills: [
                    {
                        id: 'sentiment_analysis',
                        name: '情感分析',
                        description: '分析文本的情感倾向',
                        tags: ['nlp', 'sentiment', 'emotion'],
                        examples: ['分析评论情感', '判断文本正负面'],
                        inputModes: ['text'],
                        outputModes: ['json']
                    },
                    {
                        id: 'keyword_extraction',
                        name: '关键词提取',
                        description: '从文本中提取关键词',
                        tags: ['nlp', 'keywords', 'extraction'],
                        examples: ['提取文档关键词', '分析主题'],
                        inputModes: ['text'],
                        outputModes: ['json']
                    }
                ]
            },
            host: 'localhost',
            port: 8081
        });
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold text-gray-900">注册Agent</h1>
                <p className="mt-1 text-gray-600">向AgentHub注册一个新的智能Agent</p>
            </div>

            {/* Messages */}
            {error && (
                <div className="bg-error-50 border border-error-200 rounded-md p-4">
                    <div className="flex">
                        <AlertCircle className="h-5 w-5 text-error-400"/>
                        <div className="ml-3">
                            <p className="text-sm text-error-800">{error}</p>
                        </div>
                    </div>
                </div>
            )}

            {success && (
                <div className="bg-success-50 border border-success-200 rounded-md p-4">
                    <div className="flex">
                        <CheckCircle className="h-5 w-5 text-success-400"/>
                        <div className="ml-3">
                            <p className="text-sm text-success-800">{success}</p>
                        </div>
                    </div>
                </div>
            )}

            {/* Template button */}
            <div className="flex justify-end">
                <button
                    type="button"
                    onClick={loadTemplate}
                    className="btn-secondary"
                >
                    <FileText className="h-4 w-4 mr-2"/>
                    加载模板
                </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-6">
                {/* Basic info */}
                <div className="card">
                    <h2 className="text-lg font-medium text-gray-900 mb-4">基本信息</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                Agent名称 *
                            </label>
                            <input
                                type="text"
                                className="input-field"
                                value={formData.agent_card.name}
                                onChange={(e) => handleInputChange('agent_card.name', e.target.value)}
                                placeholder="my-agent"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                版本号 *
                            </label>
                            <input
                                type="text"
                                className="input-field"
                                value={formData.agent_card.version}
                                onChange={(e) => handleInputChange('agent_card.version', e.target.value)}
                                placeholder="1.0.0"
                            />
                        </div>
                        <div className="md:col-span-2">
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                描述 *
                            </label>
                            <textarea
                                className="textarea-field"
                                rows={3}
                                value={formData.agent_card.description}
                                onChange={(e) => handleInputChange('agent_card.description', e.target.value)}
                                placeholder="描述这个Agent的功能..."
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                Agent URL *
                            </label>
                            <input
                                type="url"
                                className="input-field"
                                value={formData.agent_card.url}
                                onChange={(e) => handleInputChange('agent_card.url', e.target.value)}
                                placeholder="http://localhost:8081"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                图标URL
                            </label>
                            <input
                                type="url"
                                className="input-field"
                                value={formData.agent_card.iconUrl || ''}
                                onChange={(e) => handleInputChange('agent_card.iconUrl', e.target.value)}
                                placeholder="https://example.com/icon.png"
                            />
                        </div>
                    </div>
                </div>

                {/* Network info */}
                <div className="card">
                    <h2 className="text-lg font-medium text-gray-900 mb-4">网络配置</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                主机地址 *
                            </label>
                            <input
                                type="text"
                                className="input-field"
                                value={formData.host}
                                onChange={(e) => handleInputChange('host', e.target.value)}
                                placeholder="localhost"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                端口号 *
                            </label>
                            <input
                                type="number"
                                className="input-field"
                                min="1"
                                max="65535"
                                value={formData.port}
                                onChange={(e) => handleInputChange('port', parseInt(e.target.value))}
                                placeholder="8081"
                            />
                        </div>
                    </div>
                </div>

                {/* Capabilities */}
                <div className="card">
                    <h2 className="text-lg font-medium text-gray-900 mb-4">能力配置</h2>
                    <div className="space-y-3">
                        <label className="flex items-center">
                            <input
                                type="checkbox"
                                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                                checked={formData.agent_card.capabilities.streaming}
                                onChange={(e) => handleInputChange('agent_card.capabilities.streaming', e.target.checked)}
                            />
                            <span className="ml-2 text-sm text-gray-700">支持流式处理</span>
                        </label>
                        <label className="flex items-center">
                            <input
                                type="checkbox"
                                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                                checked={formData.agent_card.capabilities.pushNotifications}
                                onChange={(e) => handleInputChange('agent_card.capabilities.pushNotifications', e.target.checked)}
                            />
                            <span className="ml-2 text-sm text-gray-700">支持推送通知</span>
                        </label>
                        <label className="flex items-center">
                            <input
                                type="checkbox"
                                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                                checked={formData.agent_card.capabilities.stateTransitionHistory}
                                onChange={(e) => handleInputChange('agent_card.capabilities.stateTransitionHistory', e.target.checked)}
                            />
                            <span className="ml-2 text-sm text-gray-700">支持状态转换历史</span>
                        </label>
                    </div>
                </div>

                {/* Skills */}
                <div className="card">
                    <div className="flex items-center justify-between mb-4">
                        <h2 className="text-lg font-medium text-gray-900">技能配置</h2>
                        <button
                            type="button"
                            onClick={() => setShowSkillForm(true)}
                            className="btn-primary"
                        >
                            <Plus className="h-4 w-4 mr-2"/>
                            添加技能
                        </button>
                    </div>

                    {/* Skills list */}
                    {formData.agent_card.skills.length > 0 && (
                        <div className="space-y-3 mb-4">
                            {formData.agent_card.skills.map((skill, index) => (
                                <div key={skill.id} className="border border-gray-200 rounded-lg p-4">
                                    <div className="flex items-start justify-between">
                                        <div className="flex-1">
                                            <div className="flex items-center">
                                                <h3 className="text-sm font-medium text-gray-900">{skill.name}</h3>
                                                <span className="ml-2 text-xs text-gray-500">({skill.id})</span>
                                            </div>
                                            <p className="text-sm text-gray-600 mt-1">{skill.description}</p>
                                            {skill.tags.length > 0 && (
                                                <div className="flex flex-wrap gap-1 mt-2">
                                                    {skill.tags.map((tag) => (
                                                        <span
                                                            key={tag}
                                                            className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800"
                                                        >
                              {tag}
                            </span>
                                                    ))}
                                                </div>
                                            )}
                                        </div>
                                        <div className="flex items-center space-x-2">
                                            <button
                                                type="button"
                                                onClick={() => editSkill(index)}
                                                className="text-primary-600 hover:text-primary-700 text-sm"
                                            >
                                                编辑
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => removeSkill(index)}
                                                className="text-error-600 hover:text-error-700 text-sm"
                                            >
                                                删除
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}

                    {/* Skill form */}
                    {showSkillForm && (
                        <div className="border-2 border-primary-200 rounded-lg p-4 bg-primary-50">
                            <h3 className="text-md font-medium text-gray-900 mb-3">
                                {editingSkillIndex !== null ? '编辑技能' : '添加新技能'}
                            </h3>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">
                                        技能ID *
                                    </label>
                                    <input
                                        type="text"
                                        className="input-field"
                                        value={currentSkill.id}
                                        onChange={(e) => setCurrentSkill(prev => ({...prev, id: e.target.value}))}
                                        placeholder="sentiment_analysis"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">
                                        技能名称 *
                                    </label>
                                    <input
                                        type="text"
                                        className="input-field"
                                        value={currentSkill.name}
                                        onChange={(e) => setCurrentSkill(prev => ({...prev, name: e.target.value}))}
                                        placeholder="情感分析"
                                    />
                                </div>
                                <div className="md:col-span-2">
                                    <label className="block text-sm font-medium text-gray-700 mb-1">
                                        描述 *
                                    </label>
                                    <textarea
                                        className="textarea-field"
                                        rows={2}
                                        value={currentSkill.description}
                                        onChange={(e) => setCurrentSkill(prev => ({
                                            ...prev,
                                            description: e.target.value
                                        }))}
                                        placeholder="分析文本的情感倾向"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">
                                        标签 (逗号分隔)
                                    </label>
                                    <input
                                        type="text"
                                        className="input-field"
                                        value={currentSkill.tags.join(', ')}
                                        onChange={(e) => handleArrayInput(e.target.value, (tags) =>
                                            setCurrentSkill(prev => ({...prev, tags}))
                                        )}
                                        placeholder="nlp, sentiment, emotion"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">
                                        示例 (逗号分隔)
                                    </label>
                                    <input
                                        type="text"
                                        className="input-field"
                                        value={currentSkill.examples?.join(', ') || ''}
                                        onChange={(e) => handleArrayInput(e.target.value, (examples) =>
                                            setCurrentSkill(prev => ({...prev, examples}))
                                        )}
                                        placeholder="分析评论情感, 判断文本正负面"
                                    />
                                </div>
                            </div>
                            <div className="flex justify-end space-x-2 mt-4">
                                <button
                                    type="button"
                                    onClick={() => {
                                        setShowSkillForm(false);
                                        setEditingSkillIndex(null);
                                        setCurrentSkill({
                                            id: '',
                                            name: '',
                                            description: '',
                                            tags: [],
                                            examples: [],
                                            inputModes: [],
                                            outputModes: []
                                        });
                                    }}
                                    className="btn-secondary"
                                >
                                    取消
                                </button>
                                <button
                                    type="button"
                                    onClick={addSkill}
                                    className="btn-primary"
                                >
                                    {editingSkillIndex !== null ? '更新技能' : '添加技能'}
                                </button>
                            </div>
                        </div>
                    )}
                </div>

                {/* Submit */}
                <div className="flex justify-end space-x-4">
                    <button
                        type="button"
                        onClick={() => navigate('/agents')}
                        className="btn-secondary"
                    >
                        取消
                    </button>
                    <button
                        type="submit"
                        disabled={loading}
                        className="btn-primary"
                    >
                        {loading ? (
                            <>
                                <Upload className="h-4 w-4 mr-2 animate-spin"/>
                                注册中...
                            </>
                        ) : (
                            <>
                                <Save className="h-4 w-4 mr-2"/>
                                注册Agent
                            </>
                        )}
                    </button>
                </div>
            </form>
        </div>
    );
};

export default AgentRegister;