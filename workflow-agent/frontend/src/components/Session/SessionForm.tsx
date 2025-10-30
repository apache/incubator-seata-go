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
import { Send, Sparkles, ChevronDown, ChevronUp, Zap, Layers, Box, AlertCircle } from 'lucide-react';

interface ToastMethods {
  success: (message: string, duration?: number) => string;
  error: (message: string, duration?: number) => string;
  warning: (message: string, duration?: number) => string;
}

interface SessionFormProps {
  onSubmit: (description: string, context?: Record<string, any>) => void;
  isLoading?: boolean;
  toast: ToastMethods;
}

export function SessionForm({ onSubmit, isLoading, toast }: SessionFormProps) {
  const [description, setDescription] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [contextJson, setContextJson] = useState('');
  const [jsonError, setJsonError] = useState<string | null>(null);

  // Validate JSON in real-time
  const validateJson = (value: string) => {
    if (!value.trim()) {
      setJsonError(null);
      return;
    }
    try {
      JSON.parse(value);
      setJsonError(null);
    } catch (error) {
      setJsonError('JSON格式错误');
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const trimmedDescription = description.trim();

    // Validation checks
    if (!trimmedDescription) {
      toast.error('请输入工作流描述');
      return;
    }

    if (trimmedDescription.length < 20) {
      toast.warning('描述至少需要20个字符，请提供更详细的信息');
      return;
    }

    if (trimmedDescription.length > maxChars) {
      toast.error(`描述不能超过${maxChars}个字符`);
      return;
    }

    let context: Record<string, any> | undefined;
    if (contextJson.trim()) {
      try {
        context = JSON.parse(contextJson);
      } catch (error) {
        toast.error('上下文JSON格式错误，请检查后重试');
        return;
      }
    }

    onSubmit(trimmedDescription, context);
  };

  const examplePrompts = [
    {
      title: '综合研究报告',
      description: '创建一个多Agent工作流来生成综合研究报告。工作流应该：1) 使用网络搜索Agent从多个来源收集信息，2) 使用数据分析Agent处理和总结收集的数据，3) 使用文档生成Agent创建格式良好的报告。',
      icon: Layers,
      gradient: 'from-violet-500 to-fuchsia-500',
      badge: '多Agent',
    },
    {
      title: '数据处理管道',
      description: '构建一个数据处理管道，从外部来源收集数据，分析和验证数据，然后生成带有可视化的摘要报告。',
      icon: Zap,
      gradient: 'from-cyan-500 to-violet-500',
      badge: '数据分析',
    },
    {
      title: '内容审核工作流',
      description: '创建一个自动化内容审核工作流：1) 分析用户生成的内容是否存在不当材料，2) 根据严重程度对内容进行分类，3) 生成审核报告并采取适当措施。',
      icon: Box,
      gradient: 'from-emerald-500 to-cyan-500',
      badge: '自动化',
    },
  ];

  const charCount = description.length;
  const minChars = 20;
  const maxChars = 2000;
  const warningThreshold = maxChars * 0.9; // 90% = 1800 characters

  // Determine character count status
  const getCharCountStatus = () => {
    if (charCount === 0) return 'empty';
    if (charCount < minChars) return 'too-short';
    if (charCount > maxChars) return 'over-limit';
    if (charCount >= warningThreshold) return 'warning';
    return 'valid';
  };

  const charStatus = getCharCountStatus();
  const charPercentage = Math.min((charCount / maxChars) * 100, 100);

  // Get color classes based on status
  const getCharCountColor = () => {
    switch (charStatus) {
      case 'empty':
      case 'too-short':
        return 'text-slate-500 dark:text-slate-400';
      case 'over-limit':
        return 'text-rose-600 dark:text-rose-400';
      case 'warning':
        return 'text-amber-600 dark:text-amber-400';
      default:
        return 'text-emerald-600 dark:text-emerald-400';
    }
  };

  const getProgressBarColor = () => {
    switch (charStatus) {
      case 'over-limit':
        return 'from-rose-500 to-red-500';
      case 'warning':
        return 'from-amber-500 to-orange-500';
      case 'valid':
        return 'from-emerald-500 to-cyan-500';
      default:
        return 'from-slate-400 to-slate-500';
    }
  };

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      <div className="flex-1 p-8">
        <div className="max-w-4xl mx-auto space-y-8">
          {/* Hero Section */}
          <div className="text-center space-y-4 animate-fade-in">
            <div className="inline-flex animate-scale-in">
              <div className="relative group">
                {/* 背景光晕 - 流动效果 */}
                <div className="absolute inset-0 gradient-aurora rounded-3xl blur-3xl opacity-30 group-hover:opacity-50 transition-all duration-500"></div>

                {/* Icon容器 - 渐变背景 + 浮动动画 */}
                <div className="relative w-20 h-20 rounded-3xl gradient-aurora shadow-[0_8px_32px_-4px_rgba(139,92,246,0.4)] flex items-center justify-center hover-lift animate-float">
                  <Sparkles className="h-10 w-10 text-white" />
                  {/* 图标光晕 */}
                  <div className="absolute inset-0 rounded-3xl blur-md opacity-50 glow-violet"></div>
                </div>
              </div>
            </div>

            {/* 渐变标题 */}
            <h2 className="text-4xl font-bold text-gradient-aurora tracking-tight animate-fade-in-down" style={{ animationDelay: '100ms' }}>
              创建智能工作流
            </h2>

            <p className="text-lg text-slate-600 dark:text-slate-400 max-w-2xl mx-auto leading-relaxed animate-fade-in-down" style={{ animationDelay: '200ms' }}>
              用自然语言描述您的需求，AI将为您自动编排工作流，智能发现并组合合适的Agent
            </p>
          </div>

          {/* Main Form Card - Glass Effect */}
          <div className="relative group animate-fade-in-up" style={{ animationDelay: '300ms' }}>
            {/* 卡片背景渐变 */}
            <div className="absolute inset-0 gradient-aurora rounded-3xl opacity-5 dark:opacity-10"></div>

            {/* 玻璃卡片 */}
            <div className="relative glass-strong p-8 rounded-3xl hover-lift transition-all duration-300">
              <form onSubmit={handleSubmit} className="space-y-6">
                {/* Description Input */}
                <div className="space-y-3">
                  <label className="flex items-center justify-between">
                    <span className="text-base font-semibold text-slate-900 dark:text-white flex items-center gap-2">
                      工作流描述
                      <span className="px-2 py-0.5 text-xs font-medium bg-rose-500/10 dark:bg-rose-400/10 text-rose-600 dark:text-rose-400 rounded-full border border-rose-500/20">
                        必填
                      </span>
                    </span>
                    <span className={`text-xs font-mono font-medium transition-colors ${getCharCountColor()}`}>
                      {charCount} / {maxChars}
                    </span>
                  </label>

                  {/* Character count progress bar */}
                  <div className="h-1.5 bg-slate-200/50 dark:bg-slate-700/50 rounded-full overflow-hidden">
                    <div
                      className={`h-full bg-gradient-to-r ${getProgressBarColor()} transition-all duration-300`}
                      style={{ width: `${charPercentage}%` }}
                    ></div>
                  </div>

                  {/* Validation hints */}
                  {charStatus === 'too-short' && charCount > 0 && (
                    <div className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400 animate-fade-in">
                      <AlertCircle className="h-3.5 w-3.5" />
                      <span>还需要至少 {minChars - charCount} 个字符（最少{minChars}字符）</span>
                    </div>
                  )}
                  {charStatus === 'warning' && (
                    <div className="flex items-center gap-2 text-xs text-amber-600 dark:text-amber-400 animate-fade-in">
                      <AlertCircle className="h-3.5 w-3.5" />
                      <span>即将达到字符限制（剩余 {maxChars - charCount} 字符）</span>
                    </div>
                  )}
                  {charStatus === 'over-limit' && (
                    <div className="flex items-center gap-2 text-xs text-rose-600 dark:text-rose-400 animate-fade-in">
                      <AlertCircle className="h-3.5 w-3.5" />
                      <span>超出字符限制 {charCount - maxChars} 个字符</span>
                    </div>
                  )}

                  {/* 玻璃输入框 */}
                  <div className="relative">
                    <textarea
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      placeholder="详细描述您需要的工作流程，包括涉及的步骤、Agent需求、数据流向等...

例如：我需要一个工作流来处理客户反馈，包括：
1. 从多个渠道收集反馈数据
2. 使用情感分析Agent分析客户情绪
3. 按优先级分类反馈
4. 生成反馈摘要报告"
                      className="w-full min-h-[180px] px-4 py-3 rounded-2xl glass border border-white/20 dark:border-white/10
                                focus:outline-none focus:ring-2 focus:ring-violet-500/50 focus:border-violet-500/50
                                text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500
                                leading-relaxed text-sm resize-none transition-all duration-300
                                hover:border-violet-500/30 disabled:opacity-50 disabled:cursor-not-allowed"
                      disabled={isLoading}
                      maxLength={maxChars}
                    />
                  </div>

                  <p className="text-xs text-slate-500 dark:text-slate-400 flex items-center gap-1.5">
                    <Sparkles className="h-3.5 w-3.5 text-violet-500" />
                    <span>提供越详细的描述，生成的工作流越精确</span>
                  </p>
                </div>

                {/* Divider */}
                <div className="h-px bg-gradient-to-r from-transparent via-slate-300 dark:via-slate-700 to-transparent"></div>

                {/* Advanced Options Toggle */}
                <button
                  type="button"
                  onClick={() => setShowAdvanced(!showAdvanced)}
                  className="group flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-slate-300
                            hover:text-violet-600 dark:hover:text-violet-400 transition-colors"
                >
                  {showAdvanced ? (
                    <ChevronUp className="h-4 w-4 transition-transform group-hover:-translate-y-0.5" />
                  ) : (
                    <ChevronDown className="h-4 w-4 transition-transform group-hover:translate-y-0.5" />
                  )}
                  <span>高级选项</span>
                </button>

                {/* Advanced Options Content */}
                {showAdvanced && (
                  <div className="space-y-3 animate-fade-in-down">
                    <label className="text-sm font-semibold text-slate-900 dark:text-white flex items-center gap-2">
                      上下文配置
                      <span className="px-2 py-0.5 text-xs font-medium bg-slate-500/10 dark:bg-slate-400/10 text-slate-600 dark:text-slate-400 rounded-full border border-slate-500/20">
                        可选
                      </span>
                    </label>
                    <div className="relative">
                      <textarea
                        value={contextJson}
                        onChange={(e) => {
                          setContextJson(e.target.value);
                          validateJson(e.target.value);
                        }}
                        placeholder='{"task_type": "research", "collaboration": "multi-agent", "timeout": 300}'
                        className={`w-full min-h-[100px] px-4 py-3 rounded-2xl glass border transition-all duration-300
                                  focus:outline-none focus:ring-2 focus:border-violet-500/50
                                  text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500
                                  font-mono text-sm resize-none
                                  hover:border-violet-500/30 disabled:opacity-50 disabled:cursor-not-allowed
                                  ${jsonError
                                    ? 'border-rose-500/50 focus:ring-rose-500/50'
                                    : contextJson.trim() && !jsonError
                                      ? 'border-emerald-500/50 focus:ring-emerald-500/50'
                                      : 'border-white/20 dark:border-white/10 focus:ring-violet-500/50'
                                  }`}
                        disabled={isLoading}
                      />
                      {/* Validation indicator */}
                      {contextJson.trim() && (
                        <div className="absolute top-3 right-3">
                          {jsonError ? (
                            <div className="w-2 h-2 rounded-full bg-rose-500 animate-pulse"></div>
                          ) : (
                            <div className="w-2 h-2 rounded-full bg-emerald-500"></div>
                          )}
                        </div>
                      )}
                    </div>
                    {jsonError ? (
                      <p className="text-xs text-rose-600 dark:text-rose-400 flex items-center gap-1.5 animate-fade-in">
                        <AlertCircle className="h-3.5 w-3.5" />
                        <span>{jsonError}</span>
                      </p>
                    ) : contextJson.trim() ? (
                      <p className="text-xs text-emerald-600 dark:text-emerald-400 flex items-center gap-1.5 animate-fade-in">
                        <span>✓ JSON格式正确</span>
                      </p>
                    ) : (
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        请输入有效的JSON格式
                      </p>
                    )}
                  </div>
                )}

                {/* Submit Button - Gradient with Glow */}
                <button
                  type="submit"
                  disabled={!description.trim() || isLoading || charCount < minChars || charCount > maxChars || Boolean(contextJson.trim() && jsonError !== null)}
                  className="group relative w-full overflow-hidden rounded-2xl transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed focus-ring"
                >
                  {/* 渐变背景层 */}
                  <div className={`absolute inset-0 gradient-aurora ${!description.trim() || isLoading || charCount < minChars || charCount > maxChars || Boolean(contextJson.trim() && jsonError !== null) ? 'opacity-50' : 'opacity-100 group-hover:opacity-90'} transition-opacity`}></div>

                  {/* 光晕层 */}
                  <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 glow-violet"></div>

                  {/* 内容 */}
                  <div className="relative flex items-center justify-center gap-3 px-6 py-3.5 text-white font-semibold">
                    {isLoading ? (
                      <>
                        {/* 加载动画 */}
                        <div className="flex gap-1">
                          <div className="w-2 h-2 rounded-full bg-white animate-pulse" style={{ animationDelay: '0ms' }}></div>
                          <div className="w-2 h-2 rounded-full bg-white animate-pulse" style={{ animationDelay: '150ms' }}></div>
                          <div className="w-2 h-2 rounded-full bg-white animate-pulse" style={{ animationDelay: '300ms' }}></div>
                        </div>
                        <span>正在创建工作流...</span>
                      </>
                    ) : (
                      <>
                        <Send className="h-5 w-5 transition-transform group-hover:translate-x-0.5" />
                        <span>开始智能编排</span>
                        <Zap className="h-4 w-4 opacity-0 group-hover:opacity-100 transition-opacity" />
                      </>
                    )}
                  </div>

                  {/* 底部高光 */}
                  <div className="absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-white/30 to-transparent"></div>
                </button>
              </form>
            </div>
          </div>

          {/* Example Prompts */}
          <div className="space-y-4 animate-fade-in-up" style={{ animationDelay: '400ms' }}>
            <div className="flex items-center gap-3">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white flex items-center gap-2">
                <Sparkles className="h-5 w-5 text-violet-500" />
                示例工作流
              </h3>
              <span className="px-2 py-1 text-xs font-medium bg-violet-500/10 dark:bg-violet-400/10 text-violet-600 dark:text-violet-400 rounded-full border border-violet-500/20">
                点击使用
              </span>
            </div>

            <div className="grid gap-4 md:grid-cols-3">
              {examplePrompts.map((prompt, index) => (
                <button
                  key={index}
                  onClick={() => !isLoading && setDescription(prompt.description)}
                  disabled={isLoading}
                  className="relative group text-left p-6 rounded-2xl glass hover-lift transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed animate-fade-in-up"
                  style={{ animationDelay: `${500 + index * 100}ms` }}
                >
                  {/* 渐变背景 - 悬停时显示 */}
                  <div className={`absolute inset-0 rounded-2xl bg-gradient-to-r ${prompt.gradient} opacity-0 group-hover:opacity-10 transition-opacity duration-300`}></div>

                  {/* 内容 */}
                  <div className="relative space-y-3">
                    {/* Icon and Badge */}
                    <div className="flex items-start justify-between">
                      <div className="relative">
                        <div className={`w-12 h-12 rounded-xl bg-gradient-to-br ${prompt.gradient} shadow-lg flex items-center justify-center group-hover:scale-110 transition-transform duration-300`}>
                          <prompt.icon className="h-6 w-6 text-white" />
                        </div>
                        {/* Icon光晕 */}
                        <div className={`absolute inset-0 rounded-xl blur-md opacity-0 group-hover:opacity-50 transition-opacity duration-300 bg-gradient-to-br ${prompt.gradient}`}></div>
                      </div>
                      <span className="px-2 py-1 text-xs font-medium bg-slate-500/10 dark:bg-slate-400/10 text-slate-600 dark:text-slate-400 rounded-full border border-slate-500/20">
                        {prompt.badge}
                      </span>
                    </div>

                    {/* Title */}
                    <h4 className="text-base font-semibold text-slate-900 dark:text-white group-hover:text-violet-600 dark:group-hover:text-violet-400 transition-colors">
                      {prompt.title}
                    </h4>

                    {/* Description */}
                    <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed line-clamp-3">
                      {prompt.description}
                    </p>
                  </div>

                  {/* 底部渐变线 */}
                  <div className={`absolute bottom-0 left-4 right-4 h-px bg-gradient-to-r ${prompt.gradient} opacity-0 group-hover:opacity-30 transition-opacity duration-300`}></div>
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
