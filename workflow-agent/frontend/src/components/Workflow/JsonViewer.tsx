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

import { Copy, Download, Code, FileJson, BarChart3, Sparkles } from 'lucide-react';
import type { SeataStateMachine } from '../../types/api';
import { EmptyState } from '../Common/EmptyState';

interface ToastMethods {
  success: (message: string, duration?: number) => string;
  error: (message: string, duration?: number) => string;
}

interface JsonViewerProps {
  data: SeataStateMachine | null;
  title?: string;
  toast: ToastMethods;
}

export function JsonViewer({ data, title = 'Seata Saga 工作流定义', toast }: JsonViewerProps) {

  if (!data) {
    return (
      <EmptyState
        icon={FileJson}
        title="暂无数据"
        description="Saga工作流定义将在编排完成后显示，您可以在此查看完整的JSON配置"
        badge="等待生成"
        gradient="from-amber-500 via-orange-500 to-rose-500"
      />
    );
  }

  const jsonString = JSON.stringify(data, null, 2);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(jsonString);
      toast.success('已复制到剪贴板', 2000);
    } catch (error) {
      toast.error('复制失败，请重试');
    }
  };

  const handleDownload = () => {
    const blob = new Blob([jsonString], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${data.Name || 'workflow'}.json`;
    link.click();
    URL.revokeObjectURL(url);
    toast.success('JSON文件已下载', 2000);
  };

  const statesCount = Object.keys(data.States || {}).length;
  const serviceTaskCount = Object.values(data.States || {}).filter((s: any) => s.Type === 'ServiceTask').length;
  const succeedCount = Object.values(data.States || {}).filter((s: any) => s.Type === 'Succeed').length;
  const failCount = Object.values(data.States || {}).filter((s: any) => s.Type === 'Fail').length;
  const compensateCount = Object.keys(data.States || {}).filter(key => key.includes('Compensate')).length;
  const errorHandlerCount = Object.values(data.States || {}).filter((s: any) => s.Catch && s.Catch.length > 0).length;

  return (
    <div className="flex flex-col h-full">
      {/* Header with glass effect */}
      <div className="glass border-b border-white/10 dark:border-white/5 px-6 py-4 animate-fade-in-down">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
              <Code className="h-5 w-5 text-violet-500" />
              {title}
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              {data.Name} · v{data.Version} · {statesCount} 个状态
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handleCopy}
              className="group relative overflow-hidden rounded-xl px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-white/50 dark:hover:bg-white/5 transition-all duration-300 focus-ring"
              title="复制到剪贴板"
            >
              <div className="relative flex items-center gap-2">
                <Copy className="h-4 w-4" />
                <span>复制</span>
              </div>
            </button>
            <button
              onClick={handleDownload}
              className="group relative overflow-hidden rounded-xl px-4 py-2 text-sm font-medium text-white transition-all duration-300 focus-ring"
              title="下载JSON文件"
            >
              {/* Gradient background */}
              <div className="absolute inset-0 bg-gradient-to-r from-violet-500 to-fuchsia-500 opacity-100 group-hover:opacity-90 transition-opacity"></div>

              {/* Glow layer */}
              <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 glow-violet"></div>

              {/* Content */}
              <div className="relative flex items-center gap-2">
                <Download className="h-4 w-4 transition-transform group-hover:-translate-y-0.5" />
                <span>下载</span>
              </div>
            </button>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        <div className="max-w-6xl mx-auto space-y-6">
          {/* Metadata Cards with Individual Gradients */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {/* Card 1: Workflow Info */}
            <div className="relative group animate-fade-in-up hover-lift transition-all duration-300">
              <div className="absolute inset-0 bg-gradient-to-br from-violet-500/10 to-fuchsia-500/10 rounded-2xl"></div>
              <div className="relative glass-strong p-5 rounded-2xl">
                <div className="flex items-start gap-4">
                  <div className="relative flex-shrink-0">
                    <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center shadow-lg">
                      <FileJson className="h-6 w-6 text-white" />
                    </div>
                    {/* Icon glow */}
                    <div className="absolute inset-0 rounded-xl blur-md opacity-0 group-hover:opacity-50 transition-opacity duration-300 bg-gradient-to-br from-violet-500 to-fuchsia-500"></div>
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-xs text-slate-600 dark:text-slate-400 mb-1 font-medium">工作流信息</div>
                    <div className="text-lg font-bold text-transparent bg-clip-text bg-gradient-to-r from-violet-500 to-fuchsia-500 mb-2 truncate">
                      {data.Name}
                    </div>
                    <div className="space-y-1 text-xs text-slate-700 dark:text-slate-300">
                      <div className="truncate">描述: {data.Comment || '无'}</div>
                      <div>版本: {data.Version}</div>
                      <div className="truncate">起始: {data.StartState}</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Card 2: State Statistics */}
            <div className="relative group animate-fade-in-up hover-lift transition-all duration-300" style={{ animationDelay: '100ms' }}>
              <div className="absolute inset-0 bg-gradient-to-br from-cyan-500/10 to-violet-500/10 rounded-2xl"></div>
              <div className="relative glass-strong p-5 rounded-2xl">
                <div className="flex items-start gap-4">
                  <div className="relative flex-shrink-0">
                    <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-cyan-500 to-violet-500 flex items-center justify-center shadow-lg">
                      <BarChart3 className="h-6 w-6 text-white" />
                    </div>
                    {/* Icon glow */}
                    <div className="absolute inset-0 rounded-xl blur-md opacity-0 group-hover:opacity-50 transition-opacity duration-300 bg-gradient-to-br from-cyan-500 to-violet-500"></div>
                  </div>
                  <div className="flex-1">
                    <div className="text-xs text-slate-600 dark:text-slate-400 mb-1 font-medium">状态统计</div>
                    <div className="text-lg font-bold text-transparent bg-clip-text bg-gradient-to-r from-cyan-500 to-violet-500 mb-2">
                      {statesCount}
                    </div>
                    <div className="space-y-1 text-xs text-slate-700 dark:text-slate-300">
                      <div className="flex justify-between items-center">
                        <span>服务任务:</span>
                        <span className="px-2 py-0.5 rounded-full bg-violet-500/10 dark:bg-violet-400/10 text-violet-600 dark:text-violet-400 border border-violet-500/20 font-medium">
                          {serviceTaskCount}
                        </span>
                      </div>
                      <div className="flex justify-between items-center">
                        <span>成功/失败:</span>
                        <span className="flex gap-1">
                          <span className="px-2 py-0.5 rounded-full bg-emerald-500/10 dark:bg-emerald-400/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-medium">
                            {succeedCount}
                          </span>
                          <span className="px-2 py-0.5 rounded-full bg-rose-500/10 dark:bg-rose-400/10 text-rose-600 dark:text-rose-400 border border-rose-500/20 font-medium">
                            {failCount}
                          </span>
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Card 3: Compensation Mechanism */}
            <div className="relative group animate-fade-in-up hover-lift transition-all duration-300" style={{ animationDelay: '200ms' }}>
              <div className="absolute inset-0 bg-gradient-to-br from-amber-500/10 to-rose-500/10 rounded-2xl"></div>
              <div className="relative glass-strong p-5 rounded-2xl">
                <div className="flex items-start gap-4">
                  <div className="relative flex-shrink-0">
                    <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-amber-500 to-rose-500 flex items-center justify-center shadow-lg">
                      <svg className="h-6 w-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                      </svg>
                    </div>
                    {/* Icon glow */}
                    <div className="absolute inset-0 rounded-xl blur-md opacity-0 group-hover:opacity-50 transition-opacity duration-300 bg-gradient-to-br from-amber-500 to-rose-500"></div>
                  </div>
                  <div className="flex-1">
                    <div className="text-xs text-slate-600 dark:text-slate-400 mb-1 font-medium">补偿机制</div>
                    <div className="text-lg font-bold text-transparent bg-clip-text bg-gradient-to-r from-amber-500 to-rose-500 mb-2">
                      {compensateCount}
                    </div>
                    <div className="space-y-1 text-xs text-slate-700 dark:text-slate-300">
                      <div>补偿状态数量</div>
                      <div>错误处理: {errorHandlerCount} 个状态</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* JSON Code Block with Glass Effect */}
          <div className="relative group animate-fade-in-up" style={{ animationDelay: '300ms' }}>
            {/* Gradient border glow */}
            <div className="absolute -inset-0.5 bg-gradient-to-r from-violet-500 via-fuchsia-500 to-cyan-500 rounded-2xl opacity-20 group-hover:opacity-30 transition-opacity duration-300 blur"></div>

            {/* Glass code block */}
            <div className="relative glass-strong p-6 rounded-2xl overflow-hidden hover-lift transition-all duration-300">
              <div className="flex items-center gap-2 mb-4 pb-3 border-b border-white/10 dark:border-white/5">
                <Sparkles className="h-4 w-4 text-violet-500" />
                <span className="text-sm font-semibold text-slate-900 dark:text-white">JSON 定义</span>
                <div className="ml-auto px-2 py-1 rounded-full bg-slate-500/10 dark:bg-slate-400/10 border border-slate-500/20 text-xs font-medium text-slate-600 dark:text-slate-400">
                  {jsonString.split('\n').length} 行
                </div>
              </div>
              <pre className="overflow-x-auto text-sm leading-relaxed">
                <code className="text-slate-800 dark:text-slate-200 font-mono">{jsonString}</code>
              </pre>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
