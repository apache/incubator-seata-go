import type { Session } from '../../types/api';
import { formatDate, getStatusText } from '../../utils/formatters';
import { Clock, FileText, ChevronRight, Sparkles, Activity } from 'lucide-react';
import { EmptyState } from '../Common/EmptyState';

interface SessionListProps {
  sessions: Session[];
  activeSessionId?: string;
  onSessionClick: (sessionId: string) => void;
}

export function SessionList({ sessions, activeSessionId, onSessionClick }: SessionListProps) {
  if (sessions.length === 0) {
    return (
      <EmptyState
        icon={FileText}
        title="暂无历史记录"
        description="还没有创建任何工作流会话，点击「新建会话」开始您的第一次智能编排之旅"
        badge="准备开始"
        gradient="from-emerald-500 via-cyan-500 to-violet-500"
      />
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header with glass effect */}
      <div className="glass border-b border-white/10 dark:border-white/5 px-6 py-4 animate-fade-in-down">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-violet-500" />
              会话历史
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">点击查看详情</p>
          </div>
          <div className="relative">
            <div className="px-3 py-1.5 rounded-full bg-violet-500/10 dark:bg-violet-400/10 border border-violet-500/20 text-violet-600 dark:text-violet-400 font-semibold text-sm">
              {sessions.length}
            </div>
          </div>
        </div>
      </div>

      {/* Session List */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-4 space-y-3">
          {sessions.map((session, index) => {
            const isActive = activeSessionId === session.id;
            return (
              <button
                key={session.id}
                onClick={() => onSessionClick(session.id)}
                className={`
                  relative group w-full text-left p-5 rounded-2xl transition-all duration-300
                  ${isActive ? 'glass' : 'glass hover-lift'}
                  animate-fade-in-up
                `}
                style={{ animationDelay: `${index * 50}ms` }}
              >
                {/* Active state gradient background */}
                {isActive && (
                  <div className="absolute inset-0 rounded-2xl bg-gradient-to-r from-violet-500/10 to-fuchsia-500/10 dark:from-violet-500/20 dark:to-fuchsia-500/20"></div>
                )}

                {/* Active state glow */}
                {isActive && (
                  <div className="absolute inset-0 rounded-2xl opacity-50 glow-violet"></div>
                )}

                {/* Content */}
                <div className="relative space-y-3">
                  {/* Description and Indicator */}
                  <div className="flex items-start justify-between gap-3">
                    <p className={`
                      text-sm flex-1 line-clamp-2 leading-relaxed transition-colors
                      ${isActive
                        ? 'font-semibold text-slate-900 dark:text-white'
                        : 'text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-white'
                      }
                    `}>
                      {session.description}
                    </p>

                    {/* Chevron indicator */}
                    <div className="relative flex-shrink-0">
                      {isActive ? (
                        <>
                          <ChevronRight className="h-5 w-5 text-violet-600 dark:text-violet-400 rotate-90 transition-transform duration-300" />
                          {/* Pulse indicator */}
                          <div className="absolute -top-1 -right-1 flex h-2 w-2">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-violet-500 opacity-75"></span>
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-violet-500"></span>
                          </div>
                        </>
                      ) : (
                        <ChevronRight className="h-5 w-5 text-slate-400 dark:text-slate-500 group-hover:text-slate-600 dark:group-hover:text-slate-400 transition-colors" />
                      )}
                    </div>
                  </div>

                  {/* Status and Time */}
                  <div className="flex items-center justify-between gap-2">
                    {/* Status Badge with gradient */}
                    <div className={getStatusBadgeClass(session.status, isActive)}>
                      {getStatusText(session.status)}
                    </div>

                    {/* Timestamp */}
                    <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400 font-medium">
                      <Clock className="h-3.5 w-3.5" />
                      <span>{formatDate(session.created_at)}</span>
                    </div>
                  </div>

                  {/* Event Count */}
                  {session.event_count > 0 && (
                    <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-slate-500/10 dark:bg-slate-400/10 border border-slate-500/20 text-slate-600 dark:text-slate-400 text-xs font-medium w-fit">
                      <Activity className="h-3 w-3" />
                      <span>{session.event_count} 个事件</span>
                    </div>
                  )}
                </div>

                {/* Bottom gradient line for active state */}
                {isActive && (
                  <div className="absolute bottom-0 left-4 right-4 h-px bg-gradient-to-r from-violet-500 via-fuchsia-500 to-violet-500 opacity-30"></div>
                )}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// Helper function to get badge class based on status with gradient styling
function getStatusBadgeClass(status: string, isActive: boolean): string {
  const baseClasses = 'px-2.5 py-1 rounded-full text-xs font-medium border';

  switch (status) {
    case 'completed':
      return `${baseClasses} bg-emerald-500/10 dark:bg-emerald-400/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20`;
    case 'failed':
      return `${baseClasses} bg-rose-500/10 dark:bg-rose-400/10 text-rose-600 dark:text-rose-400 border-rose-500/20`;
    case 'analyzing':
    case 'discovering':
      return `${baseClasses} bg-violet-500/10 dark:bg-violet-400/10 text-violet-600 dark:text-violet-400 border-violet-500/20 ${isActive ? 'animate-pulse-glow' : ''}`;
    case 'generating':
      return `${baseClasses} bg-amber-500/10 dark:bg-amber-400/10 text-amber-600 dark:text-amber-400 border-amber-500/20`;
    default:
      return `${baseClasses} bg-slate-500/10 dark:bg-slate-400/10 text-slate-600 dark:text-slate-400 border-slate-500/20`;
  }
}
