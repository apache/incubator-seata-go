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

import { useState } from 'react';
import type { ProgressEvent, SessionStatus, ReactFlowGraph, SeataStateMachine } from '../../types/api';
import { ProgressView } from './ProgressView';
import { WorkflowCanvas } from '../Workflow/WorkflowCanvas';
import { JsonViewer } from '../Workflow/JsonViewer';
import { LayoutGrid, ListTree, FileJson } from 'lucide-react';

interface ToastMethods {
  success: (message: string, duration?: number) => string;
  error: (message: string, duration?: number) => string;
  warning: (message: string, duration?: number) => string;
  info: (message: string, duration?: number) => string;
}

interface SessionDetailProps {
  events: ProgressEvent[];
  currentStatus: SessionStatus;
  workflow?: ReactFlowGraph;
  sagaWorkflow?: SeataStateMachine;
  toast: ToastMethods;
}

type TabType = 'progress' | 'canvas' | 'json';

export function SessionDetail({
  events,
  currentStatus,
  workflow,
  sagaWorkflow,
  toast,
}: SessionDetailProps) {
  const [activeTab, setActiveTab] = useState<TabType>('progress');

  // Extract workflow from latest event if not provided directly
  const latestWorkflow = workflow || events
    .slice()
    .reverse()
    .find(e => e.workflow?.react_flow)?.workflow?.react_flow as ReactFlowGraph | undefined;

  const latestSagaWorkflow = sagaWorkflow || events
    .slice()
    .reverse()
    .find(e => e.workflow?.seata_workflow)?.workflow?.seata_workflow as SeataStateMachine | undefined;

  // Removed auto-switch behavior - let user manually navigate between tabs

  const hasWorkflow = latestWorkflow && latestWorkflow.nodes && latestWorkflow.nodes.length > 0;
  const hasSagaWorkflow = latestSagaWorkflow != null;

  const tabs = [
    {
      id: 'progress' as TabType,
      label: '编排进度',
      icon: ListTree,
      badge: events.length > 0 ? events.length : undefined,
      enabled: true,
      isActive: currentStatus !== 'completed' && currentStatus !== 'failed',
      gradient: 'from-violet-500 to-fuchsia-500',
    },
    {
      id: 'canvas' as TabType,
      label: '工作流画布',
      icon: LayoutGrid,
      badge: hasWorkflow ? `${latestWorkflow.nodes.length}节点` : undefined,
      enabled: hasWorkflow,
      isActive: false,
      gradient: 'from-cyan-500 to-violet-500',
    },
    {
      id: 'json' as TabType,
      label: 'Saga 定义',
      icon: FileJson,
      badge: hasSagaWorkflow ? 'JSON' : undefined,
      enabled: hasSagaWorkflow,
      isActive: false,
      gradient: 'from-emerald-500 to-cyan-500',
    },
  ];

  return (
    <div className="flex flex-col h-full">
      {/* Tab Navigation with Glass Effect */}
      <div className="glass border-b border-white/10 dark:border-white/5 animate-fade-in-down">
        <div className="flex items-center gap-2 p-3">
          {tabs.map((tab, index) => {
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => tab.enabled && setActiveTab(tab.id)}
                disabled={!tab.enabled}
                className={`
                  relative group inline-flex items-center gap-2 px-4 py-2.5 rounded-xl
                  font-medium text-sm transition-all duration-300
                  animate-fade-in
                  ${isActive
                    ? 'text-white'
                    : tab.enabled
                      ? 'text-slate-700 dark:text-slate-300 hover:bg-white/50 dark:hover:bg-white/5'
                      : 'text-slate-400 dark:text-slate-600 cursor-not-allowed'
                  }
                `}
                style={{ animationDelay: `${index * 50}ms` }}
              >
                {/* Active state gradient background */}
                {isActive && (
                  <>
                    <div className={`absolute inset-0 rounded-xl bg-gradient-to-r ${tab.gradient} opacity-100 group-hover:opacity-90 transition-opacity`}></div>
                    {/* Glow layer */}
                    <div className="absolute inset-0 rounded-xl opacity-50 glow-violet"></div>
                  </>
                )}

                {/* Content */}
                <div className="relative flex items-center gap-2">
                  <tab.icon className={`h-5 w-5 transition-transform ${isActive ? '' : 'group-hover:scale-110'}`} />
                  <span>{tab.label}</span>

                  {/* Badge */}
                  {tab.badge && (
                    <div className={`
                      px-2 py-0.5 rounded-full text-xs font-medium border transition-all
                      ${isActive
                        ? 'bg-white/20 text-white border-white/30'
                        : 'bg-slate-500/10 dark:bg-slate-400/10 text-slate-600 dark:text-slate-400 border-slate-500/20'
                      }
                    `}>
                      {tab.badge}
                    </div>
                  )}

                  {/* Active pulse indicator */}
                  {tab.isActive && (
                    <div className="relative flex h-2 w-2">
                      <span className={`animate-ping absolute inline-flex h-full w-full rounded-full ${isActive ? 'bg-white' : 'bg-violet-500'} opacity-75`}></span>
                      <span className={`relative inline-flex rounded-full h-2 w-2 ${isActive ? 'bg-white' : 'bg-violet-500'}`}></span>
                    </div>
                  )}
                </div>

                {/* Bottom indicator line for active tab */}
                {isActive && (
                  <div className={`absolute bottom-0 left-0 right-0 h-0.5 bg-gradient-to-r ${tab.gradient}`}></div>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'progress' && (
          <ProgressView events={events} currentStatus={currentStatus} />
        )}
        {activeTab === 'canvas' && (
          <WorkflowCanvas workflow={latestWorkflow} toast={toast} />
        )}
        {activeTab === 'json' && (
          <JsonViewer data={latestSagaWorkflow || null} toast={toast} />
        )}
      </div>
    </div>
  );
}
