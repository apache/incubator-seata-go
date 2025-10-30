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

import { useState, useEffect } from 'react';
import { Workflow, Activity, Sun, Moon, Sparkles } from 'lucide-react';

export function Header() {
  const [theme, setTheme] = useState<'light' | 'dark'>('light');

  useEffect(() => {
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null;
    if (savedTheme) {
      setTheme(savedTheme);
      if (savedTheme === 'dark') {
        document.documentElement.classList.add('dark');
      }
    }
  }, []);

  const toggleTheme = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);

    if (newTheme === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }

    localStorage.setItem('theme', newTheme);
  };

  return (
    <header className="sticky top-0 z-50 animate-fade-in-down">
      {/* 玻璃拟态背景 */}
      <div className="glass border-b border-white/20 dark:border-white/5">
        <div className="max-w-full px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo & Title */}
            <div className="flex items-center gap-4 animate-slide-in-right">
              {/* Logo with Glow */}
              <div className="relative group">
                {/* 背景光晕 */}
                <div className="absolute inset-0 bg-gradient-aurora rounded-2xl blur-2xl opacity-0 group-hover:opacity-40 transition-opacity duration-500"></div>

                {/* Logo容器 */}
                <div className="relative w-11 h-11 rounded-2xl gradient-aurora flex items-center justify-center shadow-[0_8px_32px_-4px_rgba(139,92,246,0.3)] group-hover:shadow-[0_8px_32px_-4px_rgba(139,92,246,0.5)] transition-shadow duration-300">
                  <Workflow className="h-6 w-6 text-white animate-float" />
                </div>
              </div>

              {/* Title with Gradient */}
              <div>
                <h1 className="text-xl font-bold text-gradient-aurora tracking-tight">
                  工作流编排平台
                </h1>
                <p className="text-xs text-slate-500 dark:text-slate-400 font-medium tracking-wide mt-0.5">
                  AI驱动的智能工作流生成
                </p>
              </div>
            </div>

            {/* Status & Controls */}
            <div className="flex items-center gap-3 animate-fade-in">
              {/* Status Badge with Pulse */}
              <div className="relative px-3 py-1.5 rounded-full bg-emerald-500/10 dark:bg-emerald-400/10 border border-emerald-500/20 dark:border-emerald-400/20 backdrop-blur-sm">
                <div className="flex items-center gap-2">
                  <div className="relative">
                    <Activity className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400" />
                    {/* 脉冲圆点 */}
                    <span className="absolute -top-1 -right-1 flex h-2 w-2">
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                      <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                    </span>
                  </div>
                  <span className="text-xs font-semibold text-emerald-700 dark:text-emerald-300">
                    系统运行中
                  </span>
                </div>
              </div>

              {/* Theme Switcher with Glass Effect */}
              <button
                onClick={toggleTheme}
                className="relative group w-10 h-10 rounded-xl glass-strong hover:shadow-[0_8px_32px_-4px_rgba(139,92,246,0.3)] transition-all duration-300 focus-ring"
                aria-label="Toggle theme"
              >
                <div className="absolute inset-0 rounded-xl bg-gradient-aurora opacity-0 group-hover:opacity-10 transition-opacity duration-300"></div>
                <div className="relative flex items-center justify-center w-full h-full">
                  {theme === 'light' ? (
                    <Moon className="h-5 w-5 text-slate-700 dark:text-slate-200 transition-transform group-hover:rotate-12" />
                  ) : (
                    <Sun className="h-5 w-5 text-slate-700 dark:text-slate-200 transition-transform group-hover:rotate-12" />
                  )}
                </div>
              </button>

              {/* Sparkles Button (Optional - for future features) */}
              <button
                className="relative group w-10 h-10 rounded-xl glass-strong hover:shadow-[0_8px_32px_-4px_rgba(139,92,246,0.3)] transition-all duration-300 focus-ring"
                aria-label="Features"
              >
                <div className="absolute inset-0 rounded-xl bg-gradient-aurora opacity-0 group-hover:opacity-10 transition-opacity duration-300"></div>
                <div className="relative flex items-center justify-center w-full h-full">
                  <Sparkles className="h-5 w-5 text-slate-700 dark:text-slate-200 transition-transform group-hover:scale-110" />
                </div>
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* 底部渐变分隔线 */}
      <div className="h-px bg-gradient-to-r from-transparent via-violet-500/20 to-transparent"></div>
    </header>
  );
}
