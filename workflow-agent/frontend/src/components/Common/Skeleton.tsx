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

interface SkeletonProps {
  className?: string;
  variant?: 'text' | 'circular' | 'rectangular' | 'rounded';
  width?: string | number;
  height?: string | number;
  animated?: boolean;
  style?: React.CSSProperties;
}

export function Skeleton({
  className = '',
  variant = 'text',
  width,
  height,
  animated = true,
  style: customStyle,
}: SkeletonProps) {
  const baseClasses = 'bg-gradient-to-r from-slate-200 via-slate-300 to-slate-200 dark:from-slate-700 dark:via-slate-600 dark:to-slate-700';
  const animationClasses = animated ? 'animate-pulse' : '';

  const variantClasses = {
    text: 'rounded h-4',
    circular: 'rounded-full',
    rectangular: 'rounded-none',
    rounded: 'rounded-2xl',
  };

  const style: React.CSSProperties = {
    width: width || undefined,
    height: height || undefined,
    ...customStyle,
  };

  return (
    <div
      className={`${baseClasses} ${variantClasses[variant]} ${animationClasses} ${className}`}
      style={style}
    />
  );
}

// Skeleton Card for Session List
export function SessionListSkeleton() {
  return (
    <div className="space-y-3 p-4">
      {[1, 2, 3, 4, 5].map((i) => (
        <div key={i} className="glass p-4 rounded-2xl space-y-3 animate-fade-in" style={{ animationDelay: `${i * 100}ms` }}>
          <div className="flex items-start gap-3">
            <Skeleton variant="circular" width={48} height={48} />
            <div className="flex-1 space-y-2">
              <Skeleton width="60%" height={20} />
              <Skeleton width="40%" height={16} />
            </div>
            <Skeleton variant="rounded" width={60} height={24} />
          </div>
          <Skeleton width="100%" height={14} />
          <Skeleton width="80%" height={14} />
        </div>
      ))}
    </div>
  );
}

// Skeleton for Progress Timeline
export function ProgressTimelineSkeleton() {
  return (
    <div className="space-y-6 p-6">
      {[1, 2, 3].map((i) => (
        <div key={i} className="flex gap-4 animate-fade-in" style={{ animationDelay: `${i * 150}ms` }}>
          <div className="flex flex-col items-center gap-2">
            <Skeleton variant="rounded" width={56} height={56} />
            {i < 3 && <div className="w-0.5 h-16 bg-slate-200 dark:bg-slate-700"></div>}
          </div>
          <div className="flex-1 space-y-3">
            <div className="flex items-center gap-3">
              <Skeleton width="30%" height={24} />
              <Skeleton variant="rounded" width={80} height={24} />
            </div>
            <Skeleton width="100%" height={16} />
            <Skeleton width="90%" height={16} />
            <Skeleton width="70%" height={16} />
          </div>
        </div>
      ))}
    </div>
  );
}

// Skeleton for Workflow Canvas
export function WorkflowCanvasSkeleton() {
  return (
    <div className="flex items-center justify-center h-full p-12">
      <div className="w-full max-w-4xl space-y-8 animate-fade-in">
        <div className="flex items-center justify-between">
          <div className="space-y-2">
            <Skeleton width={200} height={28} />
            <Skeleton width={150} height={20} />
          </div>
          <Skeleton variant="rounded" width={120} height={40} />
        </div>
        <div className="grid grid-cols-3 gap-6">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <Skeleton key={i} variant="rounded" height={120} className="animate-fade-in" style={{ animationDelay: `${i * 100}ms` }} />
          ))}
        </div>
        <div className="flex items-center justify-center gap-2">
          <Skeleton variant="circular" width={8} height={8} />
          <div className="flex-1 h-0.5">
            <Skeleton height={2} />
          </div>
          <Skeleton variant="circular" width={8} height={8} />
        </div>
      </div>
    </div>
  );
}

// Skeleton for JSON Viewer
export function JsonViewerSkeleton() {
  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between animate-fade-in">
        <div className="space-y-2">
          <Skeleton width={240} height={28} />
          <Skeleton width={180} height={20} />
        </div>
        <div className="flex gap-2">
          <Skeleton variant="rounded" width={100} height={40} />
          <Skeleton variant="rounded" width={100} height={40} />
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="glass-strong p-5 rounded-2xl space-y-3 animate-fade-in" style={{ animationDelay: `${i * 100}ms` }}>
            <div className="flex items-start gap-3">
              <Skeleton variant="rounded" width={48} height={48} />
              <div className="flex-1 space-y-2">
                <Skeleton width="60%" height={16} />
                <Skeleton width="80%" height={20} />
                <Skeleton width="50%" height={14} />
              </div>
            </div>
          </div>
        ))}
      </div>

      <div className="glass-strong p-6 rounded-2xl space-y-4 animate-fade-in" style={{ animationDelay: '400ms' }}>
        <div className="flex items-center gap-2">
          <Skeleton variant="circular" width={16} height={16} />
          <Skeleton width={120} height={20} />
        </div>
        {[1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
          <Skeleton key={i} width={`${100 - i * 5}%`} height={14} />
        ))}
      </div>
    </div>
  );
}

// Skeleton for Session History Loading
export function SessionHistorySkeleton() {
  return (
    <div className="flex items-center justify-center h-full">
      <div className="text-center space-y-6 animate-fade-in">
        {/* Animated icon skeleton */}
        <div className="relative inline-flex">
          <Skeleton variant="rounded" width={80} height={80} className="animate-pulse" />
        </div>

        {/* Text skeletons */}
        <div className="space-y-2">
          <Skeleton width={200} height={24} className="mx-auto" />
          <Skeleton width={280} height={16} className="mx-auto" />
        </div>

        {/* Dots animation */}
        <div className="flex justify-center gap-2">
          <Skeleton variant="circular" width={8} height={8} />
          <Skeleton variant="circular" width={8} height={8} />
          <Skeleton variant="circular" width={8} height={8} />
        </div>
      </div>
    </div>
  );
}
