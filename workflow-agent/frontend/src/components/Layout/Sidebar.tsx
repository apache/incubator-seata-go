import { Plus, History, Settings, Sparkles, Zap, Layers } from 'lucide-react';

interface SidebarProps {
  activeView: 'new' | 'history' | 'settings';
  onViewChange: (view: 'new' | 'history' | 'settings') => void;
  onNewSession: () => void;
}

export function Sidebar({ activeView, onViewChange, onNewSession }: SidebarProps) {
  const menuItems = [
    {
      id: 'new' as const,
      label: '新建会话',
      icon: Plus,
      description: '创建新工作流',
      gradient: 'from-violet-500 to-fuchsia-500'
    },
    {
      id: 'history' as const,
      label: '历史记录',
      icon: History,
      description: '查看所有会话',
      gradient: 'from-cyan-500 to-violet-500'
    },
    {
      id: 'settings' as const,
      label: '设置',
      icon: Settings,
      description: '系统配置',
      gradient: 'from-emerald-500 to-cyan-500'
    },
  ];

  return (
    <aside className="w-72 border-r border-white/10 dark:border-white/5 flex flex-col animate-fade-in-up">
      {/* 玻璃拟态背景 */}
      <div className="glass-strong h-full flex flex-col">
        {/* 创建按钮 - 带渐变和光晕 */}
        <div className="p-6 animate-scale-in">
          <button
            onClick={onNewSession}
            className="group relative w-full overflow-hidden rounded-2xl transition-all duration-300 hover-lift focus-ring"
          >
            {/* 渐变背景层 */}
            <div className="absolute inset-0 gradient-aurora opacity-100 group-hover:opacity-90 transition-opacity"></div>

            {/* 光晕层 */}
            <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 glow-violet"></div>

            {/* 内容 */}
            <div className="relative flex items-center justify-center gap-2 px-6 py-3.5 text-white">
              <div className="relative">
                <Plus className="h-5 w-5 transition-transform group-hover:rotate-90 duration-300" />
                {/* 按钮图标光晕 */}
                <div className="absolute inset-0 blur-sm opacity-50">
                  <Plus className="h-5 w-5" />
                </div>
              </div>
              <span className="font-semibold tracking-wide">创建新工作流</span>
              <Zap className="h-4 w-4 opacity-0 group-hover:opacity-100 transition-opacity" />
            </div>

            {/* 底部高光 */}
            <div className="absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-white/30 to-transparent"></div>
          </button>
        </div>

        {/* 导航菜单 */}
        <nav className="flex-1 px-4 py-2 space-y-1">
          {menuItems.map((item, index) => {
            const isActive = activeView === item.id;
            return (
              <button
                key={item.id}
                onClick={() => onViewChange(item.id)}
                className={`
                  group relative w-full rounded-xl px-4 py-3.5 transition-all duration-300
                  ${isActive ? 'glass' : 'hover:bg-white/5 dark:hover:bg-white/5'}
                  animate-slide-in-right
                `}
                style={{ animationDelay: `${index * 50}ms` }}
              >
                {/* 活动状态渐变背景 */}
                {isActive && (
                  <div className={`absolute inset-0 rounded-xl bg-gradient-to-r ${item.gradient} opacity-10 dark:opacity-20`}></div>
                )}

                {/* 悬停光晕 */}
                <div className={`absolute inset-0 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-300 ${
                  isActive ? 'glow-violet' : ''
                }`}></div>

                {/* 内容 */}
                <div className="relative flex items-center gap-3">
                  {/* 图标容器 */}
                  <div className={`
                    relative flex items-center justify-center w-10 h-10 rounded-xl transition-all duration-300
                    ${isActive
                      ? `bg-gradient-to-br ${item.gradient} shadow-lg`
                      : 'bg-slate-100/50 dark:bg-slate-800/50 group-hover:bg-slate-200/50 dark:group-hover:bg-slate-700/50'
                    }
                  `}>
                    <item.icon className={`h-5 w-5 transition-all duration-300 ${
                      isActive
                        ? 'text-white'
                        : 'text-slate-600 dark:text-slate-400 group-hover:text-slate-900 dark:group-hover:text-slate-200'
                    } ${isActive ? 'animate-pulse-glow' : ''}`} />

                    {/* 图标光晕 */}
                    {isActive && (
                      <div className="absolute inset-0 rounded-xl blur-md opacity-50">
                        <div className={`w-full h-full rounded-xl bg-gradient-to-br ${item.gradient}`}></div>
                      </div>
                    )}
                  </div>

                  {/* 文字 */}
                  <div className="flex-1 text-left">
                    <div className={`font-semibold text-sm transition-colors ${
                      isActive
                        ? 'text-slate-900 dark:text-white'
                        : 'text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-white'
                    }`}>
                      {item.label}
                    </div>
                    <div className={`text-xs mt-0.5 transition-colors ${
                      isActive
                        ? 'text-slate-600 dark:text-slate-400'
                        : 'text-slate-500 dark:text-slate-500 group-hover:text-slate-600 dark:group-hover:text-slate-400'
                    }`}>
                      {item.description}
                    </div>
                  </div>

                  {/* 活动指示器 */}
                  {isActive && (
                    <div className="flex items-center gap-1">
                      <div className={`w-2 h-2 rounded-full bg-gradient-to-r ${item.gradient} animate-pulse`}></div>
                    </div>
                  )}
                </div>

                {/* 底部边框 */}
                {isActive && (
                  <div className={`absolute bottom-0 left-4 right-4 h-px bg-gradient-to-r ${item.gradient} opacity-30`}></div>
                )}
              </button>
            );
          })}
        </nav>

        {/* 底部提示卡片 */}
        <div className="border-t border-white/10 dark:border-white/5 p-6 animate-fade-in-up" style={{ animationDelay: '300ms' }}>
          <div className="relative group rounded-2xl overflow-hidden hover-lift transition-all duration-300 cursor-pointer">
            {/* 渐变背景 */}
            <div className="absolute inset-0 gradient-ocean opacity-10 dark:opacity-20"></div>

            {/* 玻璃层 */}
            <div className="relative glass-strong p-5">
              <div className="flex items-start gap-3">
                {/* 图标 */}
                <div className="relative flex-shrink-0">
                  <div className="w-10 h-10 rounded-xl bg-gradient-ocean flex items-center justify-center shadow-lg">
                    <Sparkles className="h-5 w-5 text-white" />
                  </div>
                  {/* 图标光晕 */}
                  <div className="absolute inset-0 rounded-xl blur-md opacity-50 glow-cyan"></div>
                </div>

                {/* 内容 */}
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                      💡 智能提示
                    </h3>
                    <Layers className="h-3.5 w-3.5 text-cyan-500" />
                  </div>
                  <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    输入详细的工作流描述，AI将为您智能编排最优方案，自动发现并组合合适的Agent。
                  </p>
                </div>
              </div>

              {/* 底部装饰线 */}
              <div className="mt-4 h-px bg-gradient-to-r from-transparent via-cyan-500/30 to-transparent"></div>
            </div>

            {/* 悬停光晕 */}
            <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 glow-cyan pointer-events-none"></div>
          </div>
        </div>
      </div>
    </aside>
  );
}
