// Format date to readable string
export function formatDate(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMins < 1) return '刚刚';
  if (diffMins < 60) return `${diffMins}分钟前`;
  if (diffHours < 24) return `${diffHours}小时前`;
  if (diffDays < 7) return `${diffDays}天前`;

  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// Format relative time
export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// Get status display text in Chinese
export function getStatusText(status: string): string {
  const statusMap: Record<string, string> = {
    pending: '等待中',
    analyzing: '分析中',
    discovering: '发现Agent',
    generating: '生成工作流',
    completed: '已完成',
    failed: '失败',
  };
  return statusMap[status] || status;
}

// Get status color
export function getStatusColor(status: string): string {
  const colorMap: Record<string, string> = {
    pending: 'text-gray-600 bg-gray-100',
    analyzing: 'text-blue-600 bg-blue-100',
    discovering: 'text-purple-600 bg-purple-100',
    generating: 'text-yellow-600 bg-yellow-100',
    completed: 'text-green-600 bg-green-100',
    failed: 'text-red-600 bg-red-100',
  };
  return colorMap[status] || 'text-gray-600 bg-gray-100';
}
