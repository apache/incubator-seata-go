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

import React, { useState, useEffect } from 'react';
import { ChevronDown, ChevronRight, CheckCircle2, Circle, Clock, Target, Layers, GitBranch, Shield, Bug, Rocket, Eye, Code, Brain } from 'lucide-react';
import { useWebSocket } from '../hooks/useWebSocket';
import type { TaskPhase } from '../types';
import MarkdownRenderer from './MarkdownRenderer';
import './TaskPlanning.css';

// 6个阶段的定义，基于system_prompt.md
const phaseDefinitions: TaskPhase[] = [
  {
    id: 1,
    name: '需求分析',
    description: '深入理解用户的业务场景，识别所有涉及的服务、操作和数据实体',
    status: 'pending'
  },
  {
    id: 2,
    name: '事务分解',
    description: '将复杂业务流程分解为独立的事务步骤，确保每个步骤的职责单一且明确',
    status: 'pending'
  },
  {
    id: 3,
    name: '依赖分析',
    description: '分析步骤间的依赖关系，确定执行顺序，绘制依赖关系图',
    status: 'pending'
  },
  {
    id: 4,
    name: '补偿设计',
    description: '为每个正向操作设计对应的补偿操作，确保补偿操作的幂等性',
    status: 'pending'
  },
  {
    id: 5,
    name: '异常处理',
    description: '设计各种失败场景的处理策略，包括重试策略和兜底方案',
    status: 'pending'
  },
  {
    id: 6,
    name: '方案输出',
    description: '生成最终的完整 Saga 工作流定义，整合前面所有步骤的设计',
    status: 'pending'
  }
];

const TaskPlanning: React.FC = () => {
  const [expandedPhases, setExpandedPhases] = useState<Set<number>>(new Set([1]));
  const [viewMode, setViewMode] = useState<'phases' | 'json'>('phases');
  const [phases, setPhases] = useState<TaskPhase[]>(phaseDefinitions);
  
  const { currentResponse } = useWebSocket();

  // 根据Agent响应更新阶段状态
  useEffect(() => {
    if (currentResponse?.phase && currentResponse.phase >= 1 && currentResponse.phase <= 6) {
      setPhases(prevPhases => 
        prevPhases.map(phase => ({
          ...phase,
          status: phase.id < currentResponse.phase ? 'completed' : 
                 phase.id === currentResponse.phase ? 'in-progress' : 'pending'
        }))
      );
      
      // 自动展开当前阶段
      setExpandedPhases(prev => new Set([...prev, currentResponse.phase]));
    }
  }, [currentResponse]);

  const togglePhase = (phaseId: number) => {
    const newExpanded = new Set(expandedPhases);
    if (newExpanded.has(phaseId)) {
      newExpanded.delete(phaseId);
    } else {
      newExpanded.add(phaseId);
    }
    setExpandedPhases(newExpanded);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle2 className="status-icon completed" />;
      case 'in-progress':
        return <Clock className="status-icon in-progress" />;
      default:
        return <Circle className="status-icon pending" />;
    }
  };

  const getPhaseIcon = (phaseId: number) => {
    const icons = [Target, Layers, GitBranch, Shield, Bug, Rocket];
    const IconComponent = icons[phaseId - 1];
    return IconComponent ? <IconComponent size={20} /> : <Target size={20} />;
  };

  const renderPhaseView = () => (
    <div className="phases-container">
      {phases.map((phase) => (
        <div key={phase.id} className={`phase-card ${phase.status}`}>
          <div 
            className="phase-header"
            onClick={() => togglePhase(phase.id)}
          >
            <div className="phase-title">
              {expandedPhases.has(phase.id) ? (
                <ChevronDown className="expand-icon" />
              ) : (
                <ChevronRight className="expand-icon" />
              )}
              {getStatusIcon(phase.status)}
              <div className="phase-icon">
                {getPhaseIcon(phase.id)}
              </div>
              <div className="phase-info">
                <h3>第{phase.id}步：{phase.name}</h3>
                <p>{phase.description}</p>
              </div>
            </div>
          </div>
          
          {expandedPhases.has(phase.id) && (
            <div className="phase-details">
              {currentResponse?.phase === phase.id && currentResponse?.text && (
                <div className="phase-analysis">
                  <h4><Eye size={16} /> 当前分析</h4>
                  <div className="analysis-content">
                    <MarkdownRenderer content={currentResponse.text} />
                  </div>
                </div>
              )}
              
              {phase.status === 'pending' && (
                <div className="phase-waiting">
                  <p><Brain size={16} style={{display: 'inline', marginRight: '6px'}} />等待用户输入业务需求...</p>
                </div>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  );

  const renderJsonView = () => (
    <div className="json-container">
      <pre>{JSON.stringify(currentResponse || {}, null, 2)}</pre>
    </div>
  );

  return (
    <div className="task-planning">
      <div className="task-planning-header">
        <h2><Target size={20} /> 任务规划</h2>
        <div className="view-toggle">
          <button 
            className={viewMode === 'phases' ? 'active' : ''}
            onClick={() => setViewMode('phases')}
          >
            <Eye size={16} />
            阶段视图
          </button>
          <button 
            className={viewMode === 'json' ? 'active' : ''}
            onClick={() => setViewMode('json')}
          >
            <Code size={16} />
            JSON视图
          </button>
        </div>
      </div>
      
      <div className="task-planning-content">
        {viewMode === 'phases' ? renderPhaseView() : renderJsonView()}
      </div>
      
      <div className="task-planning-summary">
        <div className="summary-stats">
          <div className="stat">
            <span className="stat-number">{phases.filter(p => p.status === 'completed').length}</span>
            <span className="stat-label">已完成</span>
          </div>
          <div className="stat">
            <span className="stat-number">{phases.filter(p => p.status === 'in-progress').length}</span>
            <span className="stat-label">进行中</span>
          </div>
          <div className="stat">
            <span className="stat-number">{phases.filter(p => p.status === 'pending').length}</span>
            <span className="stat-label">待开始</span>
          </div>
        </div>
        
        {currentResponse?.phase && (
          <div className="current-phase-indicator">
            <span>当前阶段：第{currentResponse.phase}步 - {phases[currentResponse.phase - 1]?.name}</span>
          </div>
        )}
      </div>
    </div>
  );
};

export default TaskPlanning;