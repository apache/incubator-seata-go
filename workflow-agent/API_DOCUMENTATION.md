<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

# Workflow Agent API Documentation

## Overview

Workflow Agent provides a RESTful API for intelligent agent workflow orchestration. The system uses AI-powered analysis to discover suitable agents, match capabilities, and generate complete workflow definitions in Seata Saga format with React Flow visualizations.

## Base URL

```
http://localhost:8081
```

## Authentication

Currently, no authentication is required. CORS is enabled for cross-origin requests.

## API Modes

The Workflow Agent provides two operational modes:

1. **Legacy Synchronous Mode**: Single blocking request that returns the complete workflow
2. **Session-Based Streaming Mode** (Recommended): Real-time progress updates via Server-Sent Events (SSE)

## Endpoints

### Legacy Synchronous Mode

#### 1. Orchestrate Workflow (Blocking)

Create an intelligent multi-agent workflow based on a natural language description. This endpoint blocks until the orchestration is complete (may take 1-3 minutes).

**Endpoint**: `POST /api/v1/orchestrate`

**Request Headers**:
```
Content-Type: application/json
```

**Request Body**:
```json
{
  "description": "string (required) - Natural language description of the workflow task",
  "context": {
    "key": "value (optional) - Additional context as key-value pairs"
  }
}
```

**Request Example**:
```json
{
  "description": "Create a multi-agent workflow to generate a comprehensive research report. The workflow should: 1) Use a web search agent to gather information from multiple sources, 2) Use a data analysis agent to process and summarize the collected data, 3) Use a document generation agent to create a well-formatted report.",
  "context": {
    "task_type": "research",
    "collaboration": "multi-agent",
    "output": "document"
  }
}
```

**Response Format**:
```json
{
  "success": boolean,
  "message": "string - Human-readable status message",
  "retry_count": number,
  "capabilities": [
    {
      "requirement": {
        "name": "string",
        "description": "string",
        "required": boolean
      },
      "agent": {
        "name": "string",
        "description": "string",
        "url": "string",
        "version": "string",
        "skills": [...]
      },
      "found": boolean
    }
  ],
  "workflow": {
    "seata_workflow": {
      "Name": "string",
      "Comment": "string",
      "StartState": "string",
      "Version": "string",
      "States": {...}
    },
    "react_flow": {
      "nodes": [...],
      "edges": [...]
    }
  },
  "error": "string (optional) - Error message if orchestration failed"
}
```

**Success Response Example**:
```json
{
  "success": true,
  "message": "Workflow orchestration completed successfully",
  "retry_count": 0,
  "capabilities": [
    {
      "requirement": {
        "name": "Web Search",
        "description": "Search and retrieve information from multiple web sources",
        "required": true
      },
      "agent": {
        "name": "web-search-agent",
        "description": "Web search agent that can search and retrieve information from the internet",
        "url": "http://localhost:9001",
        "version": "1.0.0",
        "skills": [...]
      },
      "found": true
    }
  ],
  "workflow": {
    "seata_workflow": {
      "Name": "ResearchReportGeneration",
      "Comment": "Generate a comprehensive research report",
      "StartState": "WebSearch",
      "Version": "0.0.1",
      "States": {...}
    },
    "react_flow": {
      "nodes": [...],
      "edges": [...]
    }
  }
}
```

**Error Response Example**:
```json
{
  "success": false,
  "message": "Workflow orchestration failed: insufficient capabilities",
  "retry_count": 3,
  "capabilities": [...],
  "error": "orchestration error: maximum retries exceeded"
}
```

**Status Codes**:
- `200 OK` - Request processed (check `success` field for orchestration result)
- `400 Bad Request` - Invalid request body or missing required fields
- `405 Method Not Allowed` - Non-POST request to this endpoint
- `500 Internal Server Error` - Server error

### Session-Based Streaming Mode (Recommended)

The session-based streaming mode provides real-time progress updates during workflow orchestration, offering a much better user experience for long-running operations.

#### 2. Create New Session

Create a new orchestration session and start the workflow generation process.

**Endpoint**: `POST /api/v1/sessions`

**Request Body**:
```json
{
  "description": "string (required) - Natural language description of the workflow task",
  "context": {
    "key": "value (optional) - Additional context as key-value pairs"
  }
}
```

**Response**:
```json
{
  "session_id": "uuid-string",
  "message": "Session created and orchestration started"
}
```

**Status Codes**:
- `201 Created` - Session created successfully
- `400 Bad Request` - Invalid request body

**Example**:
```bash
curl -X POST http://localhost:8081/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Create a research report workflow",
    "context": {"task_type": "research"}
  }'
```

#### 3. Stream Progress Events (SSE)

Stream real-time progress updates for an active orchestration session using Server-Sent Events.

**Endpoint**: `GET /api/v1/sessions/{session_id}/stream`

**Response Type**: `text/event-stream`

**Event Types**:
- `connected`: Initial connection confirmation
- `progress`: Progress update with workflow changes
- `close`: Session completed or closed

**Progress Event Format**:
```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "step": 1,
  "action": "analyzing|discovering|generating|completed|failed",
  "message": "Human-readable progress message",
  "status": "pending|analyzing|discovering|generating|completed|failed",
  "capabilities": [...],
  "workflow": {
    "seata_workflow": {...},
    "react_flow": {...}
  }
}
```

**JavaScript Example**:
```javascript
const sessionId = "your-session-id";
const eventSource = new EventSource(`http://localhost:8081/api/v1/sessions/${sessionId}/stream`);

eventSource.addEventListener('connected', (e) => {
  console.log('Connected:', JSON.parse(e.data));
});

eventSource.addEventListener('progress', (e) => {
  const progress = JSON.parse(e.data);
  console.log(`[${progress.action}] ${progress.message}`);

  // Update UI with workflow changes
  if (progress.workflow) {
    updateWorkflowVisualization(progress.workflow);
  }

  // Close connection when complete
  if (progress.status === 'completed' || progress.status === 'failed') {
    eventSource.close();
  }
});

eventSource.addEventListener('error', (e) => {
  console.error('SSE error:', e);
  eventSource.close();
});
```

**cURL Example**:
```bash
# Use -N flag to disable buffering for streaming
curl -N http://localhost:8081/api/v1/sessions/{session_id}/stream
```

**Status Codes**:
- `200 OK` - Stream started
- `404 Not Found` - Session not found

#### 4. Get Session Status

Get the current status of an orchestration session.

**Endpoint**: `GET /api/v1/sessions/{session_id}`

**Response**:
```json
{
  "id": "uuid-string",
  "description": "string",
  "status": "pending|analyzing|discovering|generating|completed|failed",
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:32:15Z",
  "event_count": 12
}
```

**Status Codes**:
- `200 OK` - Status retrieved
- `404 Not Found` - Session not found

#### 5. Get Session Result

Retrieve the final workflow result for a completed session.

**Endpoint**: `GET /api/v1/sessions/{session_id}/result`

**Response**:
```json
{
  "success": boolean,
  "message": "string",
  "retry_count": number,
  "capabilities": [...],
  "workflow": {
    "seata_workflow": {...},
    "react_flow": {...}
  },
  "error": "string (optional)"
}
```

**Status Codes**:
- `200 OK` - Result available
- `202 Accepted` - Session still in progress
- `404 Not Found` - Session not found

#### 6. List All Sessions

Get a list of all active sessions.

**Endpoint**: `GET /api/v1/sessions`

**Response**:
```json
{
  "sessions": [
    {
      "id": "uuid-string",
      "description": "string",
      "status": "string",
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:32:15Z",
      "event_count": 12
    }
  ],
  "count": 5
}
```

**Status Codes**:
- `200 OK` - Sessions retrieved

### Health Check

#### 7. Health Check

Check if the service is running and healthy.

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy",
  "component": "workflow-agent"
}
```

**Status Codes**:
- `200 OK` - Service is healthy

#### 8. Service Info

Get basic information about the service.

**Endpoint**: `GET /`

**Response**:
```json
{
  "service": "workflow-agent",
  "version": "1.0.0",
  "status": "running"
}
```

## Workflow Output Formats

### Seata Saga Workflow

The generated Seata Saga workflow follows the standard Seata state machine format:

```json
{
  "Name": "WorkflowName",
  "Comment": "Description",
  "StartState": "FirstState",
  "Version": "0.0.1",
  "States": {
    "StateName": {
      "Type": "ServiceTask",
      "ServiceName": "agent-name",
      "ServiceMethod": "methodName",
      "CompensateState": "CompensateStateName",
      "IsForUpdate": true,
      "Input": ["$.[businessKey]", "$.[amount]", "$.[params]"],
      "Output": {"result": "$.#root"},
      "Status": {
        "#root == true": "SU",
        "#root == false": "FA",
        "$Exception{java.lang.Throwable}": "UN"
      },
      "Catch": [{
        "Exceptions": ["java.lang.Throwable"],
        "Next": "CompensationTrigger"
      }],
      "Next": "NextState"
    }
  }
}
```

**Key Features**:
- Complete state machine definition
- Service task states for each agent operation
- Compensation states for rollback
- Error handling with Catch blocks
- Status mappings for success/failure
- Terminal states (Succeed/Fail)

### React Flow Visualization

The React Flow format provides a visual representation of the workflow:

```json
{
  "nodes": [
    {
      "id": "nodeId",
      "type": "input|default|output",
      "position": {"x": number, "y": number},
      "data": {"label": "string"},
      "style": {...}
    }
  ],
  "edges": [
    {
      "id": "edgeId",
      "source": "sourceNodeId",
      "target": "targetNodeId",
      "type": "default|straight|step",
      "label": "string",
      "style": {...}
    }
  ]
}
```

**Node Types**:
- `input`: Start node
- `default`: Regular task/compensation nodes
- `output`: End nodes (Succeed/Fail)

**Node Colors**:
- Green (#52C41A): Start/Succeed nodes
- Blue (#5B8FF9): Regular task nodes
- Orange (#FF6B3B): Compensation nodes
- Yellow (#FFC53D): Compensation trigger
- Red (#FF4D4F): Fail node

## Complete Workflow Examples

### Session-Based Streaming Workflow (Recommended)

This example demonstrates the complete session-based streaming workflow:

```javascript
// 1. Create a new session
async function createSession(description, context = {}) {
  const response = await fetch('http://localhost:8081/api/v1/sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ description, context })
  });

  const result = await response.json();
  return result.session_id;
}

// 2. Stream progress with SSE
function streamProgress(sessionId, onProgress, onComplete) {
  const eventSource = new EventSource(
    `http://localhost:8081/api/v1/sessions/${sessionId}/stream`
  );

  eventSource.addEventListener('connected', (e) => {
    console.log('Stream connected:', JSON.parse(e.data));
  });

  eventSource.addEventListener('progress', (e) => {
    const progress = JSON.parse(e.data);
    onProgress(progress);

    // Close when done
    if (progress.status === 'completed' || progress.status === 'failed') {
      eventSource.close();
      onComplete(progress);
    }
  });

  eventSource.addEventListener('error', (e) => {
    console.error('Stream error:', e);
    eventSource.close();
  });

  return eventSource;
}

// 3. Get final result
async function getResult(sessionId) {
  const response = await fetch(
    `http://localhost:8081/api/v1/sessions/${sessionId}/result`
  );
  return await response.json();
}

// Complete workflow
async function orchestrateWithProgress() {
  const description = "Create a multi-agent workflow for research report generation";
  const context = { task_type: "research" };

  // Create session
  const sessionId = await createSession(description, context);
  console.log(`Session created: ${sessionId}`);

  // Stream progress
  streamProgress(
    sessionId,
    (progress) => {
      console.log(`[${progress.action}] ${progress.message}`);

      // Update UI with incremental workflow changes
      if (progress.workflow) {
        updateWorkflowVisualization(progress.workflow);
      }
    },
    async (finalProgress) => {
      console.log('Orchestration complete!');

      // Get final result with complete workflow
      const result = await getResult(sessionId);
      console.log('Final workflow:', result.workflow);
    }
  );
}

function updateWorkflowVisualization(workflow) {
  // Update your UI with the latest workflow state
  if (workflow.react_flow) {
    // Render React Flow graph
    console.log('Workflow nodes:', workflow.react_flow.nodes.length);
  }
}
```

### Python SSE Client Example

```python
import requests
import json
import sseclient  # pip install sseclient-py

def create_session(description, context=None):
    """Create a new orchestration session"""
    url = "http://localhost:8081/api/v1/sessions"
    payload = {"description": description, "context": context or {}}
    response = requests.post(url, json=payload)
    return response.json()["session_id"]

def stream_progress(session_id, on_progress):
    """Stream progress events using SSE"""
    url = f"http://localhost:8081/api/v1/sessions/{session_id}/stream"
    response = requests.get(url, stream=True)
    client = sseclient.SSEClient(response)

    for event in client.events():
        if event.event == 'connected':
            print(f"Connected: {event.data}")
        elif event.event == 'progress':
            progress = json.loads(event.data)
            on_progress(progress)

            # Stop when complete
            if progress['status'] in ['completed', 'failed']:
                break
        elif event.event == 'close':
            break

def get_result(session_id):
    """Get final orchestration result"""
    url = f"http://localhost:8081/api/v1/sessions/{session_id}/result"
    response = requests.get(url)
    return response.json()

# Complete workflow
def orchestrate_with_progress():
    description = "Create a data processing pipeline"
    context = {"pipeline_type": "etl"}

    # Create session
    session_id = create_session(description, context)
    print(f"Session created: {session_id}")

    # Stream progress
    def handle_progress(progress):
        print(f"[{progress['action']}] {progress['message']}")
        if progress.get('workflow'):
            print(f"  Workflow updated: {len(progress['workflow'].get('react_flow', {}).get('nodes', []))} nodes")

    stream_progress(session_id, handle_progress)

    # Get final result
    result = get_result(session_id)
    print("Orchestration complete!")
    return result

# Run
orchestrate_with_progress()
```

## Legacy Usage Examples

### cURL Example

```bash
curl -X POST http://localhost:8081/api/v1/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Create a workflow to analyze customer feedback and generate insights",
    "context": {
      "task_type": "analysis",
      "data_source": "customer_feedback"
    }
  }'
```

### JavaScript/Fetch Example

```javascript
async function orchestrateWorkflow(description, context = {}) {
  const response = await fetch('http://localhost:8081/api/v1/orchestrate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      description,
      context
    })
  });

  const result = await response.json();

  if (result.success) {
    console.log('Workflow generated:', result.workflow);
    return result.workflow;
  } else {
    console.error('Orchestration failed:', result.message);
    throw new Error(result.error);
  }
}

// Usage
orchestrateWorkflow(
  "Create a data processing pipeline with validation and error handling",
  { pipeline_type: "etl" }
);
```

### Python Example

```python
import requests
import json

def orchestrate_workflow(description, context=None):
    url = "http://localhost:8081/api/v1/orchestrate"
    payload = {
        "description": description,
        "context": context or {}
    }

    response = requests.post(url, json=payload)
    result = response.json()

    if result["success"]:
        print("Workflow generated successfully!")
        return result["workflow"]
    else:
        print(f"Orchestration failed: {result['message']}")
        raise Exception(result.get("error", "Unknown error"))

# Usage
workflow = orchestrate_workflow(
    "Build a multi-agent system for automated content moderation",
    {"content_type": "social_media"}
)
```

## ReAct Orchestration Process

The system uses the ReAct (Reasoning + Acting) pattern for intelligent orchestration:

1. **Analyze Capabilities**: LLM analyzes the scenario to identify required agent capabilities
2. **Discover Agents**: System searches the AgentHub for suitable agents using keyword matching
3. **Verify Requirements**: LLM checks if discovered capabilities meet minimum requirements
4. **Retry Discovery** (if needed): Refine capability descriptions and retry discovery
5. **Orchestrate Workflow**: Generate Seata Saga workflow and React Flow visualization
6. **Complete**: Return the complete workflow definition

## Error Handling

The API implements comprehensive error handling:

- **Validation Errors** (400): Invalid input, missing required fields
- **Orchestration Failures**: Reported with `success: false` but HTTP 200
  - Insufficient capabilities
  - Maximum retries exceeded
  - LLM generation errors
- **Server Errors** (500): Internal server issues

All errors include descriptive messages in the `error` field.

## Rate Limiting

Currently no rate limiting is implemented. For production use, consider implementing rate limiting based on:
- Requests per IP address
- Requests per time window
- Concurrent orchestration operations

## Timeouts

- **Request Timeout**: 180 seconds (3 minutes)
- **LLM Generation**: Varies based on model response time
- **Agent Discovery**: Typically < 1 second

Long-running orchestrations (complex scenarios with multiple retries) may take 1-3 minutes.

## Best Practices

1. **Clear Descriptions**: Provide detailed, clear descriptions of the workflow task
2. **Structured Context**: Use context fields to provide additional information
3. **Error Handling**: Always check the `success` field before processing results
4. **Workflow Validation**: Validate generated workflows before execution
5. **Agent Availability**: Ensure required agents are registered in AgentHub

## Integration with AgentHub

The Workflow Agent requires AgentHub to be running for agent discovery. Ensure:

1. AgentHub is running on the configured URL (default: `http://localhost:8080`)
2. Required agents are registered in AgentHub
3. Agents have well-defined skills with descriptive tags

## Development and Testing

For local development:

```bash
# Start AgentHub (in agenthub directory)
cd ../agenthub
go run cmd/main.go

# Register test agents
cd ../workflow-agent
go run cmd/register_agents/main.go

# Start Workflow Agent
go run cmd/agent/main.go

# Test the API
curl http://localhost:8081/health
```

## Support and Feedback

For issues and feature requests, please use the project's issue tracker.
