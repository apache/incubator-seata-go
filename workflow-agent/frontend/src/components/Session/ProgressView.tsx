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

import { useEffect, useRef, useMemo } from 'react';
import {
  CheckCircle2,
  XCircle,
  Loader2,
  Search,
  Brain,
  Workflow,
  Clock,
  Zap,
  Target,
  Sparkles,
  TrendingUp,
  Timer,
  Layers,
} from 'lucide-react';
import type { ProgressEvent, SessionStatus } from '../../types/api';
import { formatRelativeTime, getStatusText } from '../../utils/formatters';

interface ProgressViewProps {
  events: ProgressEvent[];
  currentStatus: SessionStatus;
}

export function ProgressView({ events, currentStatus }: ProgressViewProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  // Auto scroll to bottom when new events arrive
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [events]);

  // Calculate statistics
  const stats = useMemo(() => {
    const totalSteps = events.length;
    const totalCapabilities = events.reduce((acc, e) => {
      const caps = e.capabilities;
      return acc + (Array.isArray(caps) ? caps.length : 0);
    }, 0);
    const foundCapabilities = events.reduce((acc, e) => {
      const caps = e.capabilities;
      return acc + (Array.isArray(caps) ? caps.filter(c => c.found).length : 0);
    }, 0);
    const firstEvent = events[0];
    const lastEvent = events[events.length - 1];

    let duration = 0;
    if (firstEvent && lastEvent) {
      const start = new Date(firstEvent.timestamp).getTime();
      const end = new Date(lastEvent.timestamp).getTime();
      duration = Math.floor((end - start) / 1000); // seconds
    }

    return { totalSteps, totalCapabilities, foundCapabilities, duration };
  }, [events]);

  const getActionIconAndGradient = (action: string, status: SessionStatus) => {
    if (status === 'completed') {
      return {
        icon: CheckCircle2,
        gradient: 'from-emerald-500 to-cyan-500',
        color: 'text-emerald-500',
        bgGlow: 'bg-emerald-500',
      };
    }
    if (status === 'failed') {
      return {
        icon: XCircle,
        gradient: 'from-rose-500 to-fuchsia-500',
        color: 'text-rose-500',
        bgGlow: 'bg-rose-500',
      };
    }

    switch (action) {
      case 'analyzing':
      case 'react_thinking':
      case 'analyzing_capabilities':
        return {
          icon: Brain,
          gradient: 'from-violet-500 to-fuchsia-500',
          color: 'text-violet-500',
          bgGlow: 'bg-violet-500',
        };
      case 'discovering':
      case 'discovering_agents':
      case 'discovery_complete':
        return {
          icon: Search,
          gradient: 'from-cyan-500 to-violet-500',
          color: 'text-cyan-500',
          bgGlow: 'bg-cyan-500',
        };
      case 'generating':
      case 'generating_workflow':
      case 'workflow_complete':
      case 'workflow_generated':
        return {
          icon: Workflow,
          gradient: 'from-amber-500 to-rose-500',
          color: 'text-amber-500',
          bgGlow: 'bg-amber-500',
        };
      case 'react_action':
        return {
          icon: Zap,
          gradient: 'from-violet-500 to-fuchsia-500',
          color: 'text-violet-500',
          bgGlow: 'bg-violet-500',
        };
      case 'verifying':
      case 'verification_passed':
        return {
          icon: Target,
          gradient: 'from-emerald-500 to-cyan-500',
          color: 'text-emerald-500',
          bgGlow: 'bg-emerald-500',
        };
      default:
        return {
          icon: Loader2,
          gradient: 'from-slate-500 to-slate-600',
          color: 'text-slate-400',
          bgGlow: 'bg-slate-500',
        };
    }
  };

  const getStatusBadgeClass = (status: SessionStatus) => {
    const baseClasses = 'inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium border';

    switch (status) {
      case 'completed':
        return `${baseClasses} bg-emerald-500/10 dark:bg-emerald-400/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20`;
      case 'failed':
        return `${baseClasses} bg-rose-500/10 dark:bg-rose-400/10 text-rose-600 dark:text-rose-400 border-rose-500/20`;
      case 'analyzing':
      case 'discovering':
        return `${baseClasses} bg-violet-500/10 dark:bg-violet-400/10 text-violet-600 dark:text-violet-400 border-violet-500/20`;
      case 'generating':
        return `${baseClasses} bg-amber-500/10 dark:bg-amber-400/10 text-amber-600 dark:text-amber-400 border-amber-500/20`;
      default:
        return `${baseClasses} bg-slate-500/10 dark:bg-slate-400/10 text-slate-600 dark:text-slate-400 border-slate-500/20`;
    }
  };

  const isActive = currentStatus !== 'completed' && currentStatus !== 'failed';

  return (
    <div className="flex flex-col h-full">
      {/* Header with glass effect */}
      <div className="glass border-b border-white/10 dark:border-white/5 px-6 py-4 animate-fade-in-down">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="relative">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center shadow-lg">
                <Sparkles className="h-5 w-5 text-white" />
              </div>
              <div className="absolute inset-0 rounded-xl blur-md opacity-50 bg-gradient-to-br from-violet-500 to-fuchsia-500"></div>
            </div>
            <div>
              <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                编排进度
              </h3>
              <p className="text-sm text-slate-600 dark:text-slate-400">实时追踪每一个ReAct步骤</p>
            </div>
          </div>
          <div className={getStatusBadgeClass(currentStatus)}>
            {isActive && (
              <div className="flex gap-0.5">
                <div className="w-1.5 h-1.5 rounded-full bg-current animate-pulse"></div>
                <div className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" style={{ animationDelay: '150ms' }}></div>
                <div className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" style={{ animationDelay: '300ms' }}></div>
              </div>
            )}
            {getStatusText(currentStatus)}
          </div>
        </div>

        {/* Statistics Bar */}
        {events.length > 0 && (
          <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-3">
            {/* Total Steps */}
            <div className="relative group animate-fade-in" style={{ animationDelay: '100ms' }}>
              <div className="relative glass-strong p-3 rounded-xl border border-white/20 dark:border-white/10 hover-lift transition-all duration-300">
                <div className="absolute inset-0 rounded-xl bg-gradient-to-r from-violet-500/5 to-fuchsia-500/5 opacity-0 group-hover:opacity-100 transition-opacity"></div>
                <div className="relative flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center shadow-lg">
                    <Layers className="h-4 w-4 text-white" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-xs text-slate-600 dark:text-slate-400 font-medium">总步骤</p>
                    <p className="text-lg font-bold text-slate-900 dark:text-white">{stats.totalSteps}</p>
                  </div>
                </div>
              </div>
            </div>

            {/* Duration */}
            <div className="relative group animate-fade-in" style={{ animationDelay: '150ms' }}>
              <div className="relative glass-strong p-3 rounded-xl border border-white/20 dark:border-white/10 hover-lift transition-all duration-300">
                <div className="absolute inset-0 rounded-xl bg-gradient-to-r from-cyan-500/5 to-violet-500/5 opacity-0 group-hover:opacity-100 transition-opacity"></div>
                <div className="relative flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-cyan-500 to-violet-500 flex items-center justify-center shadow-lg">
                    <Timer className="h-4 w-4 text-white" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-xs text-slate-600 dark:text-slate-400 font-medium">耗时</p>
                    <p className="text-lg font-bold text-slate-900 dark:text-white">{stats.duration}s</p>
                  </div>
                </div>
              </div>
            </div>

            {/* Capabilities */}
            <div className="relative group animate-fade-in" style={{ animationDelay: '200ms' }}>
              <div className="relative glass-strong p-3 rounded-xl border border-white/20 dark:border-white/10 hover-lift transition-all duration-300">
                <div className="absolute inset-0 rounded-xl bg-gradient-to-r from-emerald-500/5 to-cyan-500/5 opacity-0 group-hover:opacity-100 transition-opacity"></div>
                <div className="relative flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-emerald-500 to-cyan-500 flex items-center justify-center shadow-lg">
                    <Target className="h-4 w-4 text-white" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-xs text-slate-600 dark:text-slate-400 font-medium">能力发现</p>
                    <p className="text-lg font-bold text-slate-900 dark:text-white">{stats.foundCapabilities}/{stats.totalCapabilities}</p>
                  </div>
                </div>
              </div>
            </div>

            {/* Success Rate */}
            <div className="relative group animate-fade-in" style={{ animationDelay: '250ms' }}>
              <div className="relative glass-strong p-3 rounded-xl border border-white/20 dark:border-white/10 hover-lift transition-all duration-300">
                <div className="absolute inset-0 rounded-xl bg-gradient-to-r from-amber-500/5 to-rose-500/5 opacity-0 group-hover:opacity-100 transition-opacity"></div>
                <div className="relative flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-amber-500 to-rose-500 flex items-center justify-center shadow-lg">
                    <TrendingUp className="h-4 w-4 text-white" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-xs text-slate-600 dark:text-slate-400 font-medium">成功率</p>
                    <p className="text-lg font-bold text-slate-900 dark:text-white">
                      {stats.totalCapabilities > 0 ? Math.round((stats.foundCapabilities / stats.totalCapabilities) * 100) : 0}%
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Timeline Content */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-4xl mx-auto">
          {events.length === 0 ? (
            <div className="text-center py-16 animate-fade-in space-y-6">
              {/* Empty state with enhanced animation */}
              <div className="relative inline-flex animate-scale-in">
                <div className="absolute inset-0 gradient-aurora rounded-3xl blur-3xl opacity-20 animate-float"></div>
                <div className="relative w-24 h-24 mx-auto rounded-3xl glass-strong flex items-center justify-center animate-float" style={{ animationDelay: '200ms' }}>
                  <Clock className="h-12 w-12 text-slate-400 dark:text-slate-500" />
                </div>
              </div>

              <div className="space-y-2 animate-fade-in-up" style={{ animationDelay: '200ms' }}>
                <p className="text-xl font-bold text-slate-900 dark:text-white">等待编排开始</p>
                <p className="text-sm text-slate-600 dark:text-slate-400">系统正在准备工作流编排...</p>
              </div>

              <div className="flex justify-center gap-2 animate-fade-in-up" style={{ animationDelay: '300ms' }}>
                {[0, 200, 400].map((delay, i) => (
                  <div
                    key={i}
                    className="w-2 h-2 rounded-full bg-gradient-to-r from-violet-500 to-fuchsia-500 animate-pulse"
                    style={{ animationDelay: `${delay}ms` }}
                  ></div>
                ))}
              </div>
            </div>
          ) : (
            <div className="relative space-y-6">
              {/* Animated timeline connector */}
              <div className="absolute left-7 top-0 bottom-0 w-0.5 overflow-hidden">
                <div className="absolute inset-0 bg-gradient-to-b from-violet-500/20 via-fuchsia-500/20 to-cyan-500/20"></div>
                {isActive && (
                  <div
                    className="absolute top-0 left-0 right-0 h-32 bg-gradient-to-b from-violet-500 via-fuchsia-500 to-transparent animate-timeline-flow"
                  ></div>
                )}
              </div>

              {events.map((event, index) => {
                const { icon: Icon, gradient, bgGlow } = getActionIconAndGradient(event.action, event.status);
                const isLatest = index === events.length - 1;
                return (
                  <div key={index} className="relative animate-fade-in-up" style={{ animationDelay: `${Math.min(index * 50, 500)}ms` }}>
                    {/* Timeline dot with enhanced animation */}
                    <div className="absolute left-0 top-0 z-10">
                      <div className="relative">
                        {/* Glow layer with pulse */}
                        {isActive && isLatest && (
                          <>
                            <div className={`absolute inset-0 rounded-2xl blur-xl opacity-50 ${bgGlow} animate-pulse`}></div>
                            <div className="absolute -inset-4">
                              <div className={`absolute inset-0 rounded-3xl ${bgGlow} opacity-20 blur-2xl animate-ping`}></div>
                            </div>
                          </>
                        )}
                        {/* Step number badge */}
                        <div className="absolute -top-2 -right-2 z-20">
                          <div className={`
                            w-6 h-6 rounded-full bg-gradient-to-br ${gradient}
                            flex items-center justify-center shadow-lg
                            ${isActive && isLatest ? 'animate-bounce' : ''}
                          `}>
                            <span className="text-xs font-bold text-white">{event.step}</span>
                          </div>
                        </div>
                        {/* Icon container */}
                        <div className={`
                          relative w-14 h-14 rounded-2xl bg-gradient-to-br ${gradient} shadow-lg
                          flex items-center justify-center transition-transform duration-300
                          ${isActive && isLatest ? 'animate-pulse-glow scale-110' : 'hover:scale-105'}
                        `}>
                          <Icon className={`h-7 w-7 text-white ${isActive && isLatest ? 'animate-spin' : ''}`} />
                        </div>
                      </div>
                    </div>

                    {/* Event Card with enhanced styling */}
                    <div className="ml-20">
                      <div className="relative group glass-strong p-5 rounded-2xl hover-lift transition-all duration-300 border border-white/20 dark:border-white/10">
                        {/* Gradient background on active */}
                        {isActive && isLatest && (
                          <>
                            <div className={`absolute inset-0 rounded-2xl bg-gradient-to-r ${gradient} opacity-5 dark:opacity-10`}></div>
                            <div className={`absolute inset-0 rounded-2xl bg-gradient-to-r ${gradient} opacity-0 group-hover:opacity-10 transition-opacity`}></div>
                          </>
                        )}

                        {/* Content */}
                        <div className="relative space-y-3">
                          {/* Header: badges and timestamp */}
                          <div className="flex items-center gap-2 flex-wrap">
                            {event.action && (
                              <div className={`
                                px-2.5 py-1 rounded-full border text-xs font-medium font-mono
                                ${isActive && isLatest
                                  ? `bg-gradient-to-r ${gradient} text-white border-transparent`
                                  : 'bg-slate-500/10 dark:bg-slate-400/10 border-slate-500/20 text-slate-600 dark:text-slate-400'
                                }
                              `}>
                                {event.action}
                              </div>
                            )}
                            <time className="text-xs text-slate-500 dark:text-slate-400 font-mono ml-auto flex items-center gap-1.5">
                              <Clock className="h-3 w-3" />
                              {formatRelativeTime(event.timestamp)}
                            </time>
                          </div>

                          {/* Message with icon */}
                          <div className="flex items-start gap-2">
                            <div className={`w-1 h-1 rounded-full ${bgGlow} mt-2 flex-shrink-0`}></div>
                            <p className="text-sm text-slate-900 dark:text-white leading-relaxed font-medium flex-1">
                              {event.message}
                            </p>
                          </div>

                          {/* Capabilities Display with enhanced styling */}
                          {event.capabilities && Array.isArray(event.capabilities) && event.capabilities.length > 0 && (
                            <div className="mt-4 space-y-3">
                              <div className="h-px bg-gradient-to-r from-transparent via-slate-300 dark:via-slate-700 to-transparent"></div>

                              <div className="flex items-center justify-between">
                                <p className="text-xs text-slate-600 dark:text-slate-400 font-semibold flex items-center gap-1.5">
                                  <Target className="h-3.5 w-3.5" />
                                  发现的能力
                                </p>
                                <div className="px-2 py-0.5 rounded-full bg-slate-500/10 dark:bg-slate-400/10 border border-slate-500/20 text-xs font-medium text-slate-600 dark:text-slate-400">
                                  {event.capabilities.filter(c => c.found).length}/{event.capabilities.length}
                                </div>
                              </div>

                              <div className="grid gap-2">
                                {event.capabilities.map((cap, capIndex) => (
                                  <div
                                    key={capIndex}
                                    className={`
                                      relative group/cap overflow-hidden rounded-xl transition-all duration-300
                                      ${cap.found
                                        ? 'glass border border-emerald-500/30 hover-lift'
                                        : 'glass border border-slate-500/20 opacity-60'
                                      }
                                    `}
                                  >
                                    {/* Animated gradient background for found capabilities */}
                                    {cap.found && (
                                      <>
                                        <div className="absolute inset-0 bg-gradient-to-r from-emerald-500/10 to-cyan-500/10"></div>
                                        <div className="absolute inset-0 bg-gradient-to-r from-emerald-500/5 to-cyan-500/5 opacity-0 group-hover/cap:opacity-100 transition-opacity"></div>
                                      </>
                                    )}

                                    {/* Content */}
                                    <div className="relative flex items-center gap-2.5 p-3">
                                      {cap.found ? (
                                        <div className="relative flex-shrink-0">
                                          <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-emerald-500 to-cyan-500 flex items-center justify-center shadow-lg">
                                            <CheckCircle2 className="h-4 w-4 text-white" />
                                          </div>
                                          {/* Animated glow */}
                                          <div className="absolute inset-0 rounded-lg blur-md opacity-0 group-hover/cap:opacity-50 transition-opacity bg-gradient-to-br from-emerald-500 to-cyan-500 animate-pulse"></div>
                                        </div>
                                      ) : (
                                        <div className="w-7 h-7 rounded-lg bg-slate-500/10 dark:bg-slate-400/10 flex items-center justify-center flex-shrink-0">
                                          <XCircle className="h-4 w-4 text-slate-400 dark:text-slate-500" />
                                        </div>
                                      )}

                                      <div className="text-xs flex-1 min-w-0">
                                        <div className="flex items-center gap-2">
                                          <span className={`font-semibold truncate ${cap.found ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-700 dark:text-slate-300'}`}>
                                            {cap.requirement.name}
                                          </span>
                                          {cap.found && (
                                            <div className="flex-shrink-0 px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-medium text-[10px]">
                                              ✓
                                            </div>
                                          )}
                                        </div>
                                        {cap.agent && (
                                          <p className="text-slate-600 dark:text-slate-400 mt-1 truncate">
                                            Agent: {cap.agent.name}
                                          </p>
                                        )}
                                      </div>
                                    </div>
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>

                        {/* Bottom gradient line for active with animation */}
                        {isActive && isLatest && (
                          <div className="absolute bottom-0 left-0 right-0 h-0.5 overflow-hidden rounded-b-2xl">
                            <div className={`h-full bg-gradient-to-r ${gradient} animate-shimmer`}></div>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}

              {/* Active Indicator with enhanced animation */}
              {isActive && (
                <div className="flex justify-center py-6 animate-pulse">
                  <div className="relative inline-flex">
                    {/* Multi-layer glow */}
                    <div className="absolute -inset-4 gradient-aurora rounded-full blur-2xl opacity-20 animate-ping"></div>
                    <div className="absolute inset-0 gradient-aurora rounded-full blur-xl opacity-30 animate-pulse"></div>
                    {/* Badge */}
                    <div className="relative px-5 py-2.5 rounded-full glass-strong border border-violet-500/30 shadow-lg">
                      <div className="flex items-center gap-2.5">
                        <div className="flex gap-1">
                          {[0, 150, 300].map((delay, i) => (
                            <div
                              key={i}
                              className="w-1.5 h-1.5 rounded-full bg-violet-500 animate-pulse"
                              style={{ animationDelay: `${delay}ms` }}
                            ></div>
                          ))}
                        </div>
                        <span className="text-sm font-semibold text-violet-600 dark:text-violet-400">编排进行中...</span>
                        <Loader2 className="h-4 w-4 text-violet-500 animate-spin" />
                      </div>
                    </div>
                  </div>
                </div>
              )}

              <div ref={bottomRef} />
            </div>
          )}
        </div>
      </div>

      {/* Add custom animations */}
      <style>{`
        @keyframes timeline-flow {
          0% {
            transform: translateY(-100%);
            opacity: 0;
          }
          50% {
            opacity: 0.5;
          }
          100% {
            transform: translateY(100vh);
            opacity: 0;
          }
        }

        @keyframes shimmer {
          0% {
            transform: translateX(-100%);
          }
          100% {
            transform: translateX(100%);
          }
        }

        .animate-timeline-flow {
          animation: timeline-flow 3s ease-in-out infinite;
        }

        .animate-shimmer {
          animation: shimmer 2s ease-in-out infinite;
        }
      `}</style>
    </div>
  );
}
