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

import { useEffect, useCallback, useMemo } from 'react';
import ReactFlow, {
  type Node,
  type Edge,
  Controls,
  Background,
  MiniMap,
  useNodesState,
  useEdgesState,
  BackgroundVariant,
  MarkerType,
  Panel,
} from 'reactflow';
import 'reactflow/dist/style.css';
import type { ReactFlowGraph } from '../../types/api';
import {
  Download,
  Maximize2,
  Network,
  Sparkles,
  GitBranch,
  LayoutGrid,
} from 'lucide-react';
import { EmptyState } from '../Common/EmptyState';
import { CustomNode } from './CustomNode';
import { getLayoutedElements } from './layoutUtils';

interface ToastMethods {
  success: (message: string, duration?: number) => string;
  error: (message: string, duration?: number) => string;
}

interface WorkflowCanvasProps {
  workflow?: ReactFlowGraph;
  toast: ToastMethods;
}

// Register custom node types
const nodeTypes = {
  custom: CustomNode,
};

export function WorkflowCanvas({ workflow, toast }: WorkflowCanvasProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  // Auto-layout function
  const onLayout = useCallback(() => {
    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
      nodes,
      edges,
      'LR'
    );

    setNodes([...layoutedNodes]);
    setEdges([...layoutedEdges]);

    toast.success('布局已自动优化', 2000);
  }, [nodes, edges, setNodes, setEdges, toast]);

  // Update nodes and edges when workflow changes with auto-layout
  useEffect(() => {
    if (workflow) {
      // Convert API nodes to React Flow nodes with custom type
      const reactFlowNodes: Node[] = workflow.nodes.map((node) => ({
        id: node.id,
        type: 'custom', // Use custom node type
        position: node.position,
        data: {
          label: node.data.label,
          description: node.data.description,
          type: node.type,
        },
      }));

      // Convert API edges to React Flow edges with enhanced styling
      const reactFlowEdges: Edge[] = workflow.edges.map((edge) => {
        // Determine edge color based on target node
        const isErrorEdge = edge.label?.includes('error') || edge.label?.includes('失败');

        return {
          id: edge.id,
          source: edge.source,
          target: edge.target,
          type: 'smoothstep',
          label: edge.label,
          animated: true,
          style: {
            stroke: isErrorEdge
              ? 'rgb(239, 68, 68)' // rose-500
              : 'rgb(139, 92, 246)', // violet-500
            strokeWidth: 2.5,
          },
          labelStyle: {
            fontSize: 12,
            fontWeight: 600,
            fill: 'rgb(71, 85, 105)', // slate-600
          },
          labelBgStyle: {
            fill: 'rgba(255, 255, 255, 0.9)',
            fillOpacity: 0.9,
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            width: 20,
            height: 20,
            color: isErrorEdge
              ? 'rgb(239, 68, 68)'
              : 'rgb(139, 92, 246)',
          },
        };
      });

      // Apply auto-layout immediately
      const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
        reactFlowNodes,
        reactFlowEdges,
        'LR'
      );

      setNodes(layoutedNodes);
      setEdges(layoutedEdges);
    }
  }, [workflow, setNodes, setEdges]);

  const handleDownload = () => {
    if (!workflow) return;

    const dataStr = JSON.stringify(workflow, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'workflow-reactflow.json';
    link.click();
    URL.revokeObjectURL(url);
    toast.success('工作流画布已导出', 2000);
  };

  const proOptions = useMemo(() => ({ hideAttribution: true }), []);

  if (!workflow || workflow.nodes.length === 0) {
    return (
      <EmptyState
        icon={Network}
        title="等待工作流生成"
        description="AI正在为您智能编排工作流，完成后将在此处展示可视化流程图"
        badge="生成中"
        gradient="from-cyan-500 via-blue-500 to-violet-500"
      />
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header with glass effect */}
      <div className="glass border-b border-white/10 dark:border-white/5 px-6 py-4 animate-fade-in-down">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-3">
              <div className="relative">
                <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-cyan-500 to-violet-500 flex items-center justify-center shadow-lg">
                  <Network className="h-5 w-5 text-white" />
                </div>
                <div className="absolute inset-0 rounded-xl blur-md opacity-50 bg-gradient-to-br from-cyan-500 to-violet-500"></div>
              </div>
              <div>
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  工作流可视化
                </h3>
                <p className="text-sm text-slate-600 dark:text-slate-400">
                  {nodes.length} 个节点 · {edges.length} 条连接
                </p>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {/* Info badges */}
            <div className="hidden md:flex items-center gap-2">
              <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full glass border border-white/20 dark:border-white/10 text-slate-600 dark:text-slate-400 text-xs font-medium">
                <Maximize2 className="h-3 w-3" />
                可缩放拖拽
              </div>
              <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full glass border border-white/20 dark:border-white/10 text-slate-600 dark:text-slate-400 text-xs font-medium">
                <GitBranch className="h-3 w-3" />
                智能连接
              </div>
            </div>

            {/* Auto Layout Button */}
            <button
              onClick={onLayout}
              className="group relative overflow-hidden rounded-xl px-4 py-2 text-sm font-medium text-white transition-all duration-300 focus-ring hover-lift"
              title="自动优化布局"
            >
              <div className="absolute inset-0 bg-gradient-to-r from-emerald-500 to-cyan-500 opacity-100 group-hover:opacity-90 transition-opacity"></div>
              <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 glow-emerald"></div>
              <div className="relative flex items-center gap-2">
                <LayoutGrid className="h-4 w-4" />
                <span>自动布局</span>
              </div>
            </button>

            {/* Download Button */}
            <button
              onClick={handleDownload}
              className="group relative overflow-hidden rounded-xl px-4 py-2 text-sm font-medium text-white transition-all duration-300 focus-ring hover-lift"
            >
              <div className="absolute inset-0 bg-gradient-to-r from-violet-500 to-fuchsia-500 opacity-100 group-hover:opacity-90 transition-opacity"></div>
              <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 glow-violet"></div>
              <div className="relative flex items-center gap-2">
                <Download className="h-4 w-4 transition-transform group-hover:-translate-y-0.5" />
                <span>导出</span>
              </div>
            </button>
          </div>
        </div>
      </div>

      {/* React Flow Canvas */}
      <div className="flex-1 relative bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{
            padding: 0.2,
            maxZoom: 1,
          }}
          minZoom={0.1}
          maxZoom={2}
          defaultEdgeOptions={{
            type: 'smoothstep',
            animated: true,
          }}
          proOptions={proOptions}
        >
          {/* Controls with custom styling */}
          <Controls
            className="!shadow-none !border-0"
            showInteractive={false}
          />

          {/* Mini Map with custom styling */}
          <MiniMap
            nodeColor={(node) => {
              const data = node.data as any;
              if (node.type === 'input') return 'rgb(16, 185, 129)'; // emerald-500
              if (data?.label?.includes('Fail')) return 'rgb(239, 68, 68)'; // rose-500
              if (node.type === 'output') return 'rgb(6, 182, 212)'; // cyan-500
              return 'rgb(139, 92, 246)'; // violet-500
            }}
            className="!shadow-none !border-0 !rounded-2xl glass-strong"
            maskColor="rgba(0, 0, 0, 0.05)"
            style={{ width: 200, height: 150 }}
          />

          {/* Background with dots pattern */}
          <Background
            variant={BackgroundVariant.Dots}
            gap={20}
            size={1.5}
            className="opacity-40 dark:opacity-20"
            color="rgb(139, 92, 246)"
          />

          {/* Info Panel */}
          <Panel position="top-left" className="glass-strong rounded-2xl px-4 py-3 shadow-lg">
            <div className="flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-violet-500" />
              <span className="text-sm font-semibold text-slate-900 dark:text-white">
                智能编排工作流
              </span>
            </div>
          </Panel>
        </ReactFlow>

        {/* Custom styled controls */}
        <style>{`
          .react-flow__controls {
            background: rgba(255, 255, 255, 0.9) !important;
            backdrop-filter: blur(24px) saturate(200%) !important;
            -webkit-backdrop-filter: blur(24px) saturate(200%) !important;
            border: 1px solid rgba(255, 255, 255, 0.4) !important;
            border-radius: 1rem !important;
            box-shadow: 0 8px 32px -4px rgba(0, 0, 0, 0.12), 0 16px 64px -8px rgba(0, 0, 0, 0.1), inset 0 1px 0 0 rgba(255, 255, 255, 0.1) !important;
          }

          .dark .react-flow__controls {
            background: rgba(15, 23, 42, 0.9) !important;
            border: 1px solid rgba(255, 255, 255, 0.15) !important;
          }

          .react-flow__controls-button {
            background: transparent !important;
            border: none !important;
            border-bottom: 1px solid rgba(100, 116, 139, 0.1) !important;
            transition: all 0.2s ease !important;
            width: 32px !important;
            height: 32px !important;
          }

          .react-flow__controls-button:hover {
            background: rgba(139, 92, 246, 0.1) !important;
          }

          .react-flow__controls-button:last-child {
            border-bottom: none !important;
          }

          .react-flow__controls-button svg {
            fill: rgb(100, 116, 139) !important;
            width: 16px !important;
            height: 16px !important;
          }

          .dark .react-flow__controls-button svg {
            fill: rgb(148, 163, 184) !important;
          }

          /* Edge styling */
          .react-flow__edge-path {
            stroke-width: 2.5 !important;
          }

          /* Selection styling */
          .react-flow__node.selected {
            box-shadow: none !important;
          }

          /* Handle styling */
          .react-flow__handle {
            transition: all 0.2s ease !important;
          }

          .react-flow__handle:hover {
            transform: scale(1.3) !important;
          }
        `}</style>
      </div>
    </div>
  );
}
