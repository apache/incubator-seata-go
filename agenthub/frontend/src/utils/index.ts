import {AgentSkill, RegisteredAgent} from '../types';

// 格式化时间
export const formatDateTime = (dateString: string): string => {
    try {
        const date = new Date(dateString);
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    } catch {
        return dateString;
    }
};

// 格式化相对时间
export const formatRelativeTime = (dateString: string): string => {
    try {
        const date = new Date(dateString);
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffMinutes = Math.floor(diffMs / (1000 * 60));
        const diffHours = Math.floor(diffMinutes / 60);
        const diffDays = Math.floor(diffHours / 24);

        if (diffMinutes < 1) {
            return '刚刚';
        } else if (diffMinutes < 60) {
            return `${diffMinutes}分钟前`;
        } else if (diffHours < 24) {
            return `${diffHours}小时前`;
        } else if (diffDays < 7) {
            return `${diffDays}天前`;
        } else {
            return formatDateTime(dateString);
        }
    } catch {
        return dateString;
    }
};

// Agent状态颜色
export const getStatusColor = (status: string): string => {
    switch (status.toLowerCase()) {
        case 'active':
            return 'text-success-600 bg-success-50';
        case 'inactive':
            return 'text-gray-600 bg-gray-100';
        case 'error':
            return 'text-error-600 bg-error-50';
        default:
            return 'text-gray-600 bg-gray-100';
    }
};

// Agent状态文本
export const getStatusText = (status: string): string => {
    switch (status.toLowerCase()) {
        case 'active':
            return '活跃';
        case 'inactive':
            return '离线';
        case 'error':
            return '错误';
        default:
            return status;
    }
};

// 检查Agent是否在线
export const isAgentOnline = (agent: RegisteredAgent): boolean => {
    if (agent.status !== 'active') return false;

    const lastSeen = new Date(agent.last_seen);
    const now = new Date();
    const diffMs = now.getTime() - lastSeen.getTime();
    const diffMinutes = diffMs / (1000 * 60);

    // 超过5分钟没有心跳认为离线
    return diffMinutes <= 5;
};

// 搜索技能
export const searchSkills = (skills: AgentSkill[], query: string): AgentSkill[] => {
    if (!query.trim()) return skills;

    const lowerQuery = query.toLowerCase();
    return skills.filter(skill =>
        skill.name.toLowerCase().includes(lowerQuery) ||
        skill.description.toLowerCase().includes(lowerQuery) ||
        skill.id.toLowerCase().includes(lowerQuery) ||
        skill.tags.some(tag => tag.toLowerCase().includes(lowerQuery))
    );
};

// 复制到剪贴板
export const copyToClipboard = async (text: string): Promise<boolean> => {
    try {
        if (navigator.clipboard) {
            await navigator.clipboard.writeText(text);
            return true;
        } else {
            // 兼容旧浏览器
            const textArea = document.createElement('textarea');
            textArea.value = text;
            document.body.appendChild(textArea);
            textArea.focus();
            textArea.select();
            const result = document.execCommand('copy');
            document.body.removeChild(textArea);
            return result;
        }
    } catch {
        return false;
    }
};

// 生成随机ID
export const generateId = (): string => {
    return Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15);
};

// 验证URL格式
export const isValidUrl = (url: string): boolean => {
    try {
        new URL(url);
        return true;
    } catch {
        return false;
    }
};

// 验证端口号
export const isValidPort = (port: number): boolean => {
    return Number.isInteger(port) && port > 0 && port <= 65535;
};