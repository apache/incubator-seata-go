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

import type { LucideIcon } from 'lucide-react';

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  badge?: string;
  gradient?: string;
}

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  badge,
  gradient = 'from-violet-500 via-fuchsia-500 to-cyan-500',
}: EmptyStateProps) {
  return (
    <div className="flex items-center justify-center h-full p-8">
      <div className="text-center space-y-6 max-w-md animate-fade-in">
        {/* Icon with enhanced animations */}
        <div className="relative inline-flex animate-scale-in">
          {/* Multi-layer glow effect */}
          <div className={`absolute inset-0 bg-gradient-to-r ${gradient} rounded-3xl blur-3xl opacity-20 animate-float`}></div>
          <div className={`absolute inset-0 bg-gradient-to-r ${gradient} rounded-3xl blur-2xl opacity-30 animate-pulse`}></div>

          {/* Glass icon container with breathing effect */}
          <div className="relative w-24 h-24 mx-auto rounded-3xl glass-strong flex items-center justify-center hover-lift animate-float" style={{ animationDelay: '200ms' }}>
            <Icon className="h-12 w-12 text-slate-400 dark:text-slate-500 transition-transform duration-500 hover:scale-110" />

            {/* Rotating gradient border */}
            <div className={`absolute inset-0 rounded-3xl bg-gradient-to-r ${gradient} opacity-0 hover:opacity-20 transition-opacity duration-500`}></div>
          </div>
        </div>

        {/* Text content with staggered animation */}
        <div className="space-y-3 animate-fade-in-up" style={{ animationDelay: '200ms' }}>
          <h3 className="text-2xl font-bold text-slate-900 dark:text-white">
            {title}
          </h3>
          <p className="text-base text-slate-600 dark:text-slate-400 leading-relaxed">
            {description}
          </p>
        </div>

        {/* Badge if provided */}
        {badge && (
          <div className={`inline-flex items-center gap-2 px-4 py-2 rounded-full glass border border-white/20 dark:border-white/10 text-sm font-medium animate-fade-in-up`} style={{ animationDelay: '300ms' }}>
            <div className={`w-2 h-2 rounded-full bg-gradient-to-r ${gradient} animate-pulse`}></div>
            <span className="text-slate-700 dark:text-slate-300">{badge}</span>
          </div>
        )}

        {/* Action button if provided */}
        {action && (
          <button
            onClick={action.onClick}
            className="group relative inline-flex items-center gap-2 px-6 py-3 rounded-2xl overflow-hidden transition-all duration-300 hover-lift animate-fade-in-up"
            style={{ animationDelay: '400ms' }}
          >
            {/* Gradient background */}
            <div className={`absolute inset-0 bg-gradient-to-r ${gradient} opacity-100 group-hover:opacity-90 transition-opacity`}></div>

            {/* Glow layer */}
            <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 glow-violet"></div>

            {/* Content */}
            <span className="relative text-white font-semibold">{action.label}</span>
          </button>
        )}

        {/* Animated dots */}
        <div className="flex justify-center gap-2 animate-fade-in-up" style={{ animationDelay: '500ms' }}>
          {[0, 200, 400].map((delay, i) => (
            <div
              key={i}
              className={`w-2 h-2 rounded-full bg-gradient-to-r ${gradient} animate-pulse`}
              style={{ animationDelay: `${delay}ms` }}
            ></div>
          ))}
        </div>
      </div>
    </div>
  );
}
