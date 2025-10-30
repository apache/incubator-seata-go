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

import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import {
  PlayCircle,
  CheckCircle2,
  XCircle,
  Workflow,
  Zap,
  Circle
} from 'lucide-react';

interface CustomNodeData {
  label: string;
  type?: 'input' | 'output' | 'default';
  status?: 'pending' | 'running' | 'completed' | 'failed';
  description?: string;
}

const nodeIcons = {
  input: PlayCircle,
  output: CheckCircle2,
  default: Workflow,
};

const statusConfig = {
  pending: {
    gradient: 'from-slate-400 to-slate-500',
    bg: 'bg-slate-500/10',
    border: 'border-slate-500/30',
    icon: Circle,
    pulse: false,
  },
  running: {
    gradient: 'from-violet-500 to-fuchsia-500',
    bg: 'bg-violet-500/10',
    border: 'border-violet-500/50',
    icon: Zap,
    pulse: true,
  },
  completed: {
    gradient: 'from-emerald-500 to-cyan-500',
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/30',
    icon: CheckCircle2,
    pulse: false,
  },
  failed: {
    gradient: 'from-rose-500 to-red-500',
    bg: 'bg-rose-500/10',
    border: 'border-rose-500/30',
    icon: XCircle,
    pulse: false,
  },
};

function CustomNodeComponent({ data, selected }: NodeProps<CustomNodeData>) {
  const nodeType = data.type || 'default';
  const status = data.status || 'pending';
  const config = statusConfig[status];
  const Icon = nodeIcons[nodeType] || nodeIcons.default;
  const StatusIcon = config.icon;

  const isInput = nodeType === 'input';
  const isOutput = nodeType === 'output';
  const isFailed = data.label?.includes('Fail');

  return (
    <div className="relative group">
      {/* Selection glow */}
      {selected && (
        <div className={`absolute -inset-2 rounded-3xl bg-gradient-to-r ${config.gradient} opacity-20 blur-xl animate-pulse`}></div>
      )}

      {/* Main card */}
      <div
        className={`
          relative min-w-[240px] max-w-[320px] rounded-2xl overflow-hidden
          glass-strong border-2 transition-all duration-300
          ${selected ? `${config.border} shadow-2xl scale-105` : 'border-white/20 dark:border-white/10 shadow-lg hover:shadow-xl'}
          ${config.pulse ? 'animate-pulse-glow' : ''}
        `}
      >
        {/* Top gradient bar */}
        <div className={`h-1.5 bg-gradient-to-r ${config.gradient}`}></div>

        {/* Content */}
        <div className="p-4 space-y-3">
          {/* Header */}
          <div className="flex items-start gap-3">
            {/* Icon */}
            <div className={`relative flex-shrink-0 w-10 h-10 rounded-xl bg-gradient-to-br ${config.gradient} shadow-lg flex items-center justify-center`}>
              <Icon className="h-5 w-5 text-white" />
              {/* Icon glow */}
              <div className={`absolute inset-0 rounded-xl blur-md opacity-50 bg-gradient-to-br ${config.gradient}`}></div>
            </div>

            {/* Title and status */}
            <div className="flex-1 min-w-0">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white truncate">
                {data.label}
              </h3>
              {data.description && (
                <p className="text-xs text-slate-600 dark:text-slate-400 mt-1 line-clamp-2">
                  {data.description}
                </p>
              )}
            </div>

            {/* Status indicator */}
            <div className="relative flex-shrink-0">
              <div className={`w-8 h-8 rounded-lg ${config.bg} border ${config.border} flex items-center justify-center`}>
                <StatusIcon className="h-4 w-4 text-slate-700 dark:text-slate-300" />
              </div>
              {config.pulse && (
                <div className="absolute inset-0 flex items-center justify-center">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-lg bg-violet-500 opacity-75"></span>
                </div>
              )}
            </div>
          </div>

          {/* Type badge */}
          <div className="flex items-center gap-2">
            <div className={`
              px-2.5 py-1 rounded-full text-xs font-medium border
              ${isInput
                ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20'
                : isOutput
                  ? isFailed
                    ? 'bg-rose-500/10 text-rose-600 dark:text-rose-400 border-rose-500/20'
                    : 'bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 border-cyan-500/20'
                  : 'bg-violet-500/10 text-violet-600 dark:text-violet-400 border-violet-500/20'
              }
            `}>
              {isInput ? '开始' : isOutput ? (isFailed ? '失败' : '完成') : '处理'}
            </div>
          </div>
        </div>

        {/* Bottom gradient line */}
        <div className={`h-px bg-gradient-to-r ${config.gradient} opacity-30`}></div>
      </div>

      {/* Handles */}
      {!isOutput && (
        <Handle
          type="source"
          position={Position.Right}
          className="!w-3 !h-3 !bg-gradient-to-r !from-violet-500 !to-fuchsia-500 !border-2 !border-white dark:!border-slate-900 !shadow-lg"
        />
      )}
      {!isInput && (
        <Handle
          type="target"
          position={Position.Left}
          className="!w-3 !h-3 !bg-gradient-to-r !from-cyan-500 !to-violet-500 !border-2 !border-white dark:!border-slate-900 !shadow-lg"
        />
      )}
    </div>
  );
}

export const CustomNode = memo(CustomNodeComponent);
