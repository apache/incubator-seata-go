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

import React, { useCallback, useState, useEffect } from 'react';
import ReactFlow, {
  addEdge,
  useNodesState,
  useEdgesState,
  Controls,
  MiniMap,
  Background,
  BackgroundVariant,
} from 'reactflow';
import type { Node, Connection } from 'reactflow';
import { Play, Pause, RotateCcw, Workflow, Code2, Eye, Info, FileText, Cog, Wrench, CheckSquare, Plug } from 'lucide-react';
import { useWebSocket } from '../hooks/useWebSocket';
import './WorkflowVisualization.css';

// 默认的空状态节点
const defaultNodes: Node[] = [
  {
    id: 'placeholder',
    type: 'default',
    position: { x: 250, y: 150 },
    data: { 
      label: '等待工作流设计...',
      description: '请在聊天界面描述您的业务场景'
    },
    style: {
      background: 'var(--card-bg)',
      color: 'var(--text-secondary)',
      border: '2px dashed var(--border-color)',
      borderRadius: '12px',
      padding: '20px',
      textAlign: 'center',
      width: '300px'
    },
    draggable: false
  }
];

const WorkflowVisualization: React.FC = () => {
  const [nodes, setNodes, onNodesChange] = useNodesState(defaultNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [isPlaying, setIsPlaying] = useState(false);
  const [viewMode, setViewMode] = useState<'workflow' | 'json'>('workflow');
  
  const { currentResponse } = useWebSocket();

  // 根据Agent响应更新工作流图
  useEffect(() => {
    if (currentResponse?.graph) {
      const { nodes: graphNodes, edges: graphEdges } = currentResponse.graph;
      
      if (graphNodes && graphNodes.length > 0) {
        // 转换节点格式，确保兼容ReactFlow
        const convertedNodes = graphNodes.map(node => ({
          ...node,
          style: {
            ...node.style,
            borderRadius: '8px',
            padding: '10px',
            minWidth: '120px',
            textAlign: 'center' as const,
            fontFamily: 'inherit'
          }
        }));
        
        setNodes(convertedNodes);
      }
      
      if (graphEdges && graphEdges.length > 0) {
        // 转换边格式
        const convertedEdges = graphEdges.map(edge => ({
          ...edge,
          animated: edge.type === 'default',
          style: {
            strokeWidth: 2,
            ...edge.style
          }
        }));
        
        setEdges(convertedEdges);
      }
    } else if (currentResponse && !currentResponse.graph) {
      // 如果当前阶段没有图形数据，显示占位符
      const phaseTexts: { [key: number]: string } = {
        1: '正在分析业务需求...',
        2: '正在分解事务步骤...',
        3: '即将生成依赖关系图',
        4: '即将添加补偿设计',
        5: '即将完善异常处理',
        6: '即将生成完整工作流'
      };
      
      const placeholderText = phaseTexts[currentResponse.phase] || '正在设计工作流...';
      
      setNodes([{
        id: 'placeholder',
        type: 'default',
        position: { x: 250, y: 150 },
        data: { 
          label: placeholderText,
          description: '图形将在后续步骤中生成'
        },
        style: {
          background: 'var(--card-bg)',
          color: 'var(--warning-color)',
          border: '2px dashed var(--warning-color)',
          borderRadius: '12px',
          padding: '20px',
          textAlign: 'center',
          width: '300px'
        },
        draggable: false
      }]);
      setEdges([]);
    }
  }, [currentResponse, setNodes, setEdges]);

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges],
  );

  const handlePlayPause = () => {
    setIsPlaying(!isPlaying);
  };

  const handleReset = () => {
    if (currentResponse?.graph) {
      const { nodes: graphNodes, edges: graphEdges } = currentResponse.graph;
      setNodes(graphNodes || defaultNodes);
      setEdges(graphEdges || []);
    } else {
      setNodes(defaultNodes);
      setEdges([]);
    }
    setIsPlaying(false);
  };

  const getWorkflowStatus = () => {
    if (!currentResponse) {
      return { text: '等待连接', color: 'var(--text-secondary)', icon: <Plug size={16} /> };
    }
    
    if (currentResponse.phase <= 2) {
      return { text: '需求分析中', color: 'var(--warning-color)', icon: <FileText size={16} /> };
    } else if (currentResponse.phase <= 4) {
      return { text: '设计中', color: 'var(--accent-color)', icon: <Cog size={16} /> };
    } else if (currentResponse.phase < 6) {
      return { text: '完善中', color: 'var(--success-color)', icon: <Wrench size={16} /> };
    } else {
      return { text: '已完成', color: 'var(--success-color)', icon: <CheckSquare size={16} /> };
    }
  };

  const workflowStatus = getWorkflowStatus();

  const renderWorkflowView = () => (
    <div className="workflow-container">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        connectionLineStyle={{ stroke: 'var(--accent-color)', strokeWidth: 2 }}
        defaultViewport={{ x: 0, y: 0, zoom: 0.8 }}
        minZoom={0.2}
        maxZoom={2}
        attributionPosition="bottom-left"
        fitView
        fitViewOptions={{ padding: 0.2 }}
      >
        <Background 
          variant={BackgroundVariant.Dots} 
          gap={20} 
          size={1} 
          color="var(--border-color)"
        />
        <Controls 
          className="workflow-controls"
          showZoom={true}
          showFitView={true}
          showInteractive={true}
        />
        <MiniMap
          className="workflow-minimap"
          nodeColor={(node) => {
            if (node.style?.background) {
              return node.style.background as string;
            }
            return 'var(--accent-color)';
          }}
          maskColor="rgba(26, 26, 46, 0.8)"
          position="bottom-right"
        />
      </ReactFlow>
    </div>
  );

  const renderJsonView = () => (
    <div className="json-container">
      <pre>{JSON.stringify(currentResponse?.seata_json || {}, null, 2)}</pre>
    </div>
  );

  return (
    <div className="workflow-visualization">
      <div className="workflow-header">
        <div className="workflow-title">
          <h2><Workflow size={20} /> Saga 状态图</h2>
          <div className="workflow-status">
            <span className="status-dot" style={{ background: workflowStatus.color }} />
            <span style={{display: 'flex', alignItems: 'center', gap: '4px'}}>
              {workflowStatus.icon}
              {workflowStatus.text}
            </span>
          </div>
        </div>
        
        <div className="workflow-controls-top">
          <div className="view-toggle">
            <button 
              className={viewMode === 'workflow' ? 'active' : ''}
              onClick={() => setViewMode('workflow')}
            >
              <Eye size={16} />
              流程图
            </button>
            <button 
              className={viewMode === 'json' ? 'active' : ''}
              onClick={() => setViewMode('json')}
            >
              <Code2 size={16} />
              Seata JSON
            </button>
          </div>
          
          <div className="playback-controls">
            <button onClick={handlePlayPause} disabled={!currentResponse?.graph}>
              {isPlaying ? <Pause size={16} /> : <Play size={16} />}
              {isPlaying ? '暂停' : '播放'}
            </button>
            <button onClick={handleReset} disabled={!currentResponse}>
              <RotateCcw size={16} />
              重置
            </button>
          </div>
        </div>
      </div>
      
      <div className="workflow-content">
        {viewMode === 'workflow' ? renderWorkflowView() : renderJsonView()}
      </div>
      
      {currentResponse?.graph && (
        <div className="workflow-legend">
          <div className="legend-item">
            <div className="legend-dot" style={{ background: '#52C41A' }} />
            <span>开始/结束</span>
          </div>
          <div className="legend-item">
            <div className="legend-dot" style={{ background: '#5B8FF9' }} />
            <span>服务任务</span>
          </div>
          <div className="legend-item">
            <div className="legend-dot" style={{ background: '#FF6B3B' }} />
            <span>补偿操作</span>
          </div>
          <div className="legend-item">
            <div className="legend-dot" style={{ background: '#FFC53D' }} />
            <span>决策节点</span>
          </div>
        </div>
      )}
      
      {!currentResponse?.graph && currentResponse && (
        <div className="workflow-info">
          <Info size={16} />
          <span>图形将在第{Math.max(3, currentResponse.phase)}步开始生成</span>
        </div>
      )}
    </div>
  );
};

export default WorkflowVisualization;