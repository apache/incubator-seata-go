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

import { useState, useEffect, useCallback } from 'react';
import { Header } from './components/Layout/Header';
import { Sidebar } from './components/Layout/Sidebar';
import { SessionForm } from './components/Session/SessionForm';
import { SessionList } from './components/Session/SessionList';
import { SessionDetail } from './components/Session/SessionDetail';
import { ToastContainer } from './components/Common/Toast';
import { SessionHistorySkeleton } from './components/Common/Skeleton';
import { EmptyState } from './components/Common/EmptyState';
import { useToast } from './hooks/useToast';
import {
  createSession,
  listSessions,
  streamSessionProgress,
  getSessionHistory,
} from './services/api';
import type { Session, ProgressEvent, SessionStatus, ReactFlowGraph, SeataStateMachine } from './types/api';
import { FileText } from 'lucide-react';

type ViewType = 'new' | 'history' | 'settings';

// Active session data including real-time events
interface ActiveSessionData {
  id: string;
  description: string;
  events: ProgressEvent[];
  status: SessionStatus;
  workflow?: ReactFlowGraph;
  sagaWorkflow?: SeataStateMachine;
  eventSource?: EventSource;
  isLoadingHistory: boolean;
}

function App() {
  const [activeView, setActiveView] = useState<ViewType>('new');
  const [sessions, setSessions] = useState<Session[]>([]);
  const [activeSessions, setActiveSessions] = useState<Map<string, ActiveSessionData>>(
    new Map()
  );
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Toast system
  const toast = useToast();

  // Load sessions on mount
  useEffect(() => {
    loadSessions();
  }, []);

  const loadSessions = async () => {
    try {
      const response = await listSessions();
      setSessions(response.sessions);
    } catch (err) {
      console.error('Failed to load sessions:', err);
      setError('加载会话列表失败');
    }
  };

  // Load session history when clicking on a session
  const loadSessionHistory = useCallback(async (sessionId: string) => {
    // If already loaded and has events, skip
    const existing = activeSessions.get(sessionId);
    if (existing && existing.events.length > 0 && !existing.isLoadingHistory) {
      return;
    }

    // Mark as loading
    setActiveSessions((prev) => {
      const newMap = new Map(prev);
      const current = newMap.get(sessionId) || {
        id: sessionId,
        description: '',
        events: [],
        status: 'pending' as SessionStatus,
        isLoadingHistory: true,
      };
      newMap.set(sessionId, { ...current, isLoadingHistory: true });
      return newMap;
    });

    try {
      // Fetch complete history from backend
      const history = await getSessionHistory(sessionId);

      // Extract workflow data from the latest event or result
      let workflow: ReactFlowGraph | undefined;
      let sagaWorkflow: SeataStateMachine | undefined;

      // Try to get from events first
      for (let i = history.events.length - 1; i >= 0; i--) {
        const event = history.events[i];
        if (event.workflow?.react_flow) {
          workflow = event.workflow.react_flow;
        }
        if (event.workflow?.seata_workflow) {
          sagaWorkflow = event.workflow.seata_workflow;
        }
        if (workflow && sagaWorkflow) break;
      }

      // Update state with complete history
      setActiveSessions((prev) => {
        const newMap = new Map(prev);
        newMap.set(sessionId, {
          id: sessionId,
          description: history.description,
          events: history.events || [],
          status: history.status,
          workflow,
          sagaWorkflow,
          isLoadingHistory: false,
        });
        return newMap;
      });
    } catch (err) {
      console.error('Failed to load session history:', err);
      setError(`加载会话历史失败: ${err}`);

      // Mark as not loading even if failed
      setActiveSessions((prev) => {
        const newMap = new Map(prev);
        const current = newMap.get(sessionId);
        if (current) {
          newMap.set(sessionId, { ...current, isLoadingHistory: false });
        }
        return newMap;
      });
    }
  }, [activeSessions]);

  const handleCreateSession = async (description: string, context?: Record<string, any>) => {
    setIsCreating(true);
    setError(null);

    try {
      // Create session
      const response = await createSession({ description, context });
      const sessionId = response.session_id;

      // Initialize active session data
      const sessionData: ActiveSessionData = {
        id: sessionId,
        description,
        events: [],
        status: 'pending',
        isLoadingHistory: false,
      };

      // Start streaming progress
      const eventSource = streamSessionProgress(
        sessionId,
        (event: ProgressEvent) => {
          setActiveSessions((prev) => {
            const newMap = new Map(prev);
            const session = newMap.get(sessionId);
            if (session) {
              // Add event to history
              session.events = [...session.events, event];
              session.status = event.status;

              // Update workflow data if present
              if (event.workflow?.react_flow) {
                session.workflow = event.workflow.react_flow;
              }
              if (event.workflow?.seata_workflow) {
                session.sagaWorkflow = event.workflow.seata_workflow;
              }
            }
            return newMap;
          });
        },
        (error) => {
          console.error('Stream error:', error);
          setError('连接中断，请刷新页面');
        },
        () => {
          // Stream completed
          loadSessions(); // Refresh session list
        }
      );

      sessionData.eventSource = eventSource;
      setActiveSessions((prev) => new Map(prev).set(sessionId, sessionData));
      setCurrentSessionId(sessionId);
      setActiveView('history');

      // Show success toast
      toast.success('会话创建成功，开始编排工作流');
    } catch (err: any) {
      console.error('Failed to create session:', err);
      const errorMsg = err.message || '创建会话失败';
      setError(errorMsg);

      // Show error toast
      toast.error(errorMsg);
    } finally {
      setIsCreating(false);
    }
  };

  const handleNewSession = () => {
    setActiveView('new');
    setCurrentSessionId(null);
  };

  const handleSessionClick = async (sessionId: string) => {
    setCurrentSessionId(sessionId);

    // Load history if not already loaded
    if (!activeSessions.has(sessionId)) {
      const session = sessions.find((s) => s.id === sessionId);
      if (session) {
        // Initialize with basic info first
        setActiveSessions((prev) => {
          const newMap = new Map(prev);
          newMap.set(sessionId, {
            id: sessionId,
            description: session.description,
            events: [],
            status: session.status,
            isLoadingHistory: true,
          });
          return newMap;
        });

        // Load complete history
        await loadSessionHistory(sessionId);
      }
    } else {
      // Reload history to get latest data
      await loadSessionHistory(sessionId);
    }
  };

  // Cleanup event sources on unmount
  useEffect(() => {
    return () => {
      activeSessions.forEach((session) => {
        session.eventSource?.close();
      });
    };
  }, [activeSessions]);

  const currentSession = currentSessionId
    ? activeSessions.get(currentSessionId)
    : null;

  return (
    <div className="flex flex-col h-screen bg-base-200">
      <Header />

      {error && (
        <div className="alert alert-error shadow-lg border-0 rounded-none animate-slide-in">
          <svg xmlns="http://www.w3.org/2000/svg" className="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span className="font-medium">{error}</span>
          <button
            onClick={() => setError(null)}
            className="btn btn-ghost btn-sm"
          >
            关闭
          </button>
        </div>
      )}

      <div className="flex-1 flex overflow-hidden">
        <Sidebar
          activeView={activeView}
          onViewChange={setActiveView}
          onNewSession={handleNewSession}
        />

        <main className="flex-1 overflow-hidden bg-base-200">
          {activeView === 'new' && (
            <SessionForm onSubmit={handleCreateSession} isLoading={isCreating} toast={toast} />
          )}

          {activeView === 'history' && (
            <div className="flex h-full gap-4 p-4">
              <div className="w-96 card bg-base-100 shadow-xl overflow-hidden">
                <SessionList
                  sessions={sessions}
                  activeSessionId={currentSessionId || undefined}
                  onSessionClick={handleSessionClick}
                />
              </div>

              <div className="flex-1 card bg-base-100 shadow-xl overflow-hidden">
                {currentSession ? (
                  currentSession.isLoadingHistory ? (
                    <SessionHistorySkeleton />
                  ) : (
                    <SessionDetail
                      events={currentSession.events}
                      currentStatus={currentSession.status}
                      workflow={currentSession.workflow}
                      sagaWorkflow={currentSession.sagaWorkflow}
                      toast={toast}
                    />
                  )
                ) : (
                  <EmptyState
                    icon={FileText}
                    title="选择一个会话"
                    description="从左侧列表中选择一个会话，查看详细的编排进度和生成的工作流"
                    badge="等待选择"
                    gradient="from-violet-500 via-fuchsia-500 to-cyan-500"
                  />
                )}
              </div>
            </div>
          )}

          {activeView === 'settings' && (
            <div className="flex items-center justify-center h-full">
              <div className="text-center animate-fade-in">
                <div className="text-6xl mb-4">⚙️</div>
                <p className="text-xl font-bold mb-2">设置</p>
                <p className="text-sm opacity-60">功能开发中...</p>
                <button className="btn btn-primary mt-4">敬请期待</button>
              </div>
            </div>
          )}
        </main>
      </div>

      {/* Toast Container */}
      <ToastContainer toasts={toast.toasts} onRemove={toast.removeToast} />
    </div>
  );
}

export default App;
