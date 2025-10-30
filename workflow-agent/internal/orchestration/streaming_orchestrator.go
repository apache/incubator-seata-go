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

package orchestration

import (
	"context"
	"fmt"

	"seata-go-ai-workflow-agent/pkg/hubclient"
	"seata-go-ai-workflow-agent/pkg/llm"
	"seata-go-ai-workflow-agent/pkg/logger"
	"seata-go-ai-workflow-agent/pkg/session"
)

// StreamingOrchestrator wraps the base orchestrator with progress streaming
type StreamingOrchestrator struct {
	orchestrator *Orchestrator
	sessionMgr   *session.Manager
}

// NewStreamingOrchestrator creates a new streaming orchestrator
func NewStreamingOrchestrator(
	config OrchestratorConfig,
	llmClient *llm.Client,
	hubClient *hubclient.HubClient,
	sessionMgr *session.Manager,
) *StreamingOrchestrator {
	return &StreamingOrchestrator{
		orchestrator: NewOrchestrator(config, llmClient, hubClient),
		sessionMgr:   sessionMgr,
	}
}

// OrchestrateWithSession performs orchestration with real-time progress updates
func (so *StreamingOrchestrator) OrchestrateWithSession(
	ctx context.Context,
	sess *session.Session,
	req ScenarioRequest,
) (*OrchestrationResult, error) {
	stepCounter := 0

	// Helper function to emit progress events
	emitProgress := func(action, message string, status session.SessionStatus, capabilities interface{}, workflow *session.WorkflowSnapshot) {
		stepCounter++
		sess.AddProgressEvent(session.ProgressEvent{
			Step:         stepCounter,
			Action:       action,
			Message:      message,
			Status:       status,
			Capabilities: capabilities,
			Workflow:     workflow,
		})
	}

	// Emit initial event
	emitProgress("initialized", fmt.Sprintf("开始编排工作流: %s", req.Description), session.StatusPending, nil, nil)

	// Emit analyzing event
	emitProgress("analyzing", "正在分析场景需求...", session.StatusAnalyzing, nil, nil)

	// Call the orchestrator with detailed monitoring
	result, err := so.orchestrateWithDetailedProgress(ctx, req, sess, &stepCounter)

	if err != nil {
		// Orchestration failed
		emitProgress("failed", fmt.Sprintf("编排失败: %v", err), session.StatusFailed, nil, nil)
		sess.SetResult(result)
		return result, err
	}

	// Emit final workflow with both formats
	if result.Success && result.SeataWorkflow != nil && result.ReactFlow != nil {
		workflow := &session.WorkflowSnapshot{
			SeataWorkflow: result.SeataWorkflow,
			ReactFlow:     result.ReactFlow,
		}

		emitProgress("workflow_complete", "工作流生成成功", session.StatusGenerating, result.Capabilities, workflow)
		emitProgress("completed", "编排完成", session.StatusCompleted, result.Capabilities, workflow)
	} else {
		emitProgress("failed", result.Message, session.StatusFailed, result.Capabilities, nil)
	}

	// Set final result
	sess.SetResult(result)

	logger.WithField("session_id", sess.ID).WithField("success", result.Success).Info("orchestration session completed")

	return result, nil
}

// orchestrateWithDetailedProgress performs orchestration with detailed step-by-step progress
func (so *StreamingOrchestrator) orchestrateWithDetailedProgress(
	ctx context.Context,
	req ScenarioRequest,
	sess *session.Session,
	stepCounter *int,
) (*OrchestrationResult, error) {

	emitProgress := func(action, message string, status session.SessionStatus, capabilities interface{}) {
		*stepCounter++
		sess.AddProgressEvent(session.ProgressEvent{
			Step:         *stepCounter,
			Action:       action,
			Message:      message,
			Status:       status,
			Capabilities: capabilities,
		})
	}

	result := &OrchestrationResult{
		Success:    false,
		RetryCount: 0,
	}

	// ReAct loop with detailed progress
	var trace ReActTrace
	var analysisResp *AnalysisResponse
	var discoveredCapabilities []DiscoveredCapability
	var minimumRequired int

	maxSteps := 20
	for step := 1; step <= maxSteps; step++ {
		emitProgress("react_thinking", fmt.Sprintf("ReAct步骤 %d: 思考下一步行动...", step), session.StatusAnalyzing, nil)

		// Get next action from LLM
		action, err := so.orchestrator.getNextAction(ctx, req.Description, step, trace.Steps)
		if err != nil {
			emitProgress("error", fmt.Sprintf("无法确定下一步行动: %v", err), session.StatusFailed, nil)
			result.Message = fmt.Sprintf("failed to determine next action: %v", err)
			return result, err
		}

		emitProgress("react_action", fmt.Sprintf("步骤 %d - 行动: %s (思考: %s)", step, action.Action, action.Thought), session.StatusAnalyzing, nil)

		var observation string

		switch action.Action {
		case "analyze_capabilities":
			emitProgress("analyzing_capabilities", "正在分析所需能力...", session.StatusAnalyzing, nil)

			analysisResp, err = so.orchestrator.capabilityAnalyzer.AnalyzeScenario(ctx, req)
			if err != nil {
				observation = fmt.Sprintf("Failed to analyze scenario: %v", err)
				emitProgress("error", observation, session.StatusFailed, nil)
				result.Message = observation
				return result, err
			}
			minimumRequired = analysisResp.MinimumRequiredCount
			observation = fmt.Sprintf("识别出 %d 个所需能力 (最少需要 %d 个)", len(analysisResp.Capabilities), minimumRequired)

			emitProgress("analysis_complete", observation, session.StatusAnalyzing, nil)

		case "discover_agents":
			if analysisResp == nil {
				observation = "错误: 必须先分析能力"
				emitProgress("error", observation, session.StatusFailed, nil)
				break
			}

			emitProgress("discovering_agents", "正在从Agent Hub发现合适的Agent...", session.StatusDiscovering, nil)

			discoveredCapabilities, err = so.orchestrator.capabilityAnalyzer.DiscoverCapabilities(ctx, analysisResp.Capabilities)
			if err != nil {
				observation = fmt.Sprintf("发现Agent失败: %v", err)
				emitProgress("error", observation, session.StatusFailed, nil)
				break
			}

			foundCount := CountFoundCapabilities(discoveredCapabilities)
			observation = fmt.Sprintf("从Hub发现 %d/%d 个能力", foundCount, len(discoveredCapabilities))
			result.Capabilities = discoveredCapabilities

			emitProgress("discovery_complete", observation, session.StatusDiscovering, discoveredCapabilities)

		case "verify_requirements":
			emitProgress("verifying", "正在验证能力是否满足需求...", session.StatusDiscovering, nil)

			if len(discoveredCapabilities) == 0 {
				observation = "错误: 必须先发现Agent"
				emitProgress("error", observation, session.StatusFailed, nil)
				break
			}

			checkResp, err := so.orchestrator.capabilityAnalyzer.CheckMinimumRequirements(
				ctx, req.Description, discoveredCapabilities, minimumRequired)
			if err != nil {
				observation = fmt.Sprintf("验证需求失败: %v", err)
				emitProgress("error", observation, session.StatusFailed, nil)
				break
			}

			if checkResp.Sufficient {
				observation = fmt.Sprintf("✓ 需求已满足: %s", checkResp.Reason)
				emitProgress("verification_passed", observation, session.StatusDiscovering, nil)
			} else {
				observation = fmt.Sprintf("✗ 需求未满足: %s", checkResp.Reason)
				emitProgress("verification_failed", observation, session.StatusDiscovering, nil)
			}

		case "retry_discovery":
			if result.RetryCount >= so.orchestrator.config.MaxRetryCount {
				observation = fmt.Sprintf("已达到最大重试次数 (%d)", so.orchestrator.config.MaxRetryCount)
				emitProgress("max_retries", observation, session.StatusFailed, nil)
				result.Message = observation
				return result, fmt.Errorf("maximum retries exceeded")
			}

			result.RetryCount++
			emitProgress("retrying", fmt.Sprintf("重试发现Agent (第 %d 次)", result.RetryCount), session.StatusDiscovering, nil)

			missing := GetMissingRequiredCapabilities(discoveredCapabilities)
			if len(missing) > 0 {
				refined, err := so.orchestrator.capabilityAnalyzer.RefineCapabilityDescription(ctx, missing[0])
				if err != nil {
					observation = fmt.Sprintf("优化能力描述失败: %v", err)
					emitProgress("error", observation, session.StatusFailed, nil)
					break
				}

				for i := range analysisResp.Capabilities {
					if analysisResp.Capabilities[i].Name == missing[0].Name {
						analysisResp.Capabilities[i] = *refined
						break
					}
				}

				discoveredCapabilities, err = so.orchestrator.capabilityAnalyzer.DiscoverCapabilities(ctx, analysisResp.Capabilities)
				if err != nil {
					observation = fmt.Sprintf("重新发现Agent失败: %v", err)
					emitProgress("error", observation, session.StatusFailed, nil)
					break
				}

				foundCount := CountFoundCapabilities(discoveredCapabilities)
				observation = fmt.Sprintf("重试后发现 %d/%d 个能力", foundCount, len(discoveredCapabilities))
				result.Capabilities = discoveredCapabilities

				emitProgress("retry_complete", observation, session.StatusDiscovering, discoveredCapabilities)
			}

		case "orchestrate_workflow":
			emitProgress("generating_workflow", "正在生成工作流定义...", session.StatusGenerating, nil)

			if len(discoveredCapabilities) == 0 {
				observation = "错误: 没有可用的能力来生成工作流"
				emitProgress("error", observation, session.StatusFailed, nil)
				break
			}

			workflow, err := so.orchestrator.workflowGenerator.GenerateWorkflow(ctx, req.Description, discoveredCapabilities)
			if err != nil {
				observation = fmt.Sprintf("生成工作流失败: %v", err)
				emitProgress("error", observation, session.StatusFailed, nil)
				result.Message = observation
				break
			}

			result.SeataWorkflow = workflow.SeataJSON
			result.ReactFlow = workflow.ReactFlow
			observation = fmt.Sprintf("工作流 '%s' 生成成功: %d 个状态, %d 个节点",
				workflow.WorkflowName, len(workflow.SeataJSON.States), len(workflow.ReactFlow.Nodes))

			workflowSnapshot := &session.WorkflowSnapshot{
				SeataWorkflow: result.SeataWorkflow,
				ReactFlow:     result.ReactFlow,
			}

			emitProgress("workflow_generated", observation, session.StatusGenerating, workflowSnapshot)

		case "complete":
			result.Success = true
			result.Message = "Workflow orchestration completed successfully"
			emitProgress("completing", "编排成功完成", session.StatusCompleted, nil)
			return result, nil

		case "fail":
			result.Success = false
			result.Message = "Workflow orchestration failed: " + action.Thought
			emitProgress("failed", result.Message, session.StatusFailed, nil)
			return result, fmt.Errorf(result.Message)

		default:
			observation = fmt.Sprintf("未知的行动: %s", action.Action)
			emitProgress("unknown_action", observation, session.StatusFailed, nil)
		}

		// Record step in trace
		trace.Steps = append(trace.Steps, ReActStep{
			Thought:     action.Thought,
			Action:      action.Action,
			Observation: observation,
		})

		emitProgress("react_observation", fmt.Sprintf("观察: %s", observation), session.StatusAnalyzing, nil)
	}

	// Max steps reached
	result.Message = "达到最大编排步骤数"
	emitProgress("max_steps", result.Message, session.StatusFailed, nil)
	return result, fmt.Errorf("maximum orchestration steps exceeded")
}

// GetSessionManager returns the session manager
func (so *StreamingOrchestrator) GetSessionManager() *session.Manager {
	return so.sessionMgr
}
