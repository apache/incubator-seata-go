import type {
  CreateSessionRequest,
  CreateSessionResponse,
  Session,
  SessionListResponse,
  OrchestrationResult,
  ProgressEvent,
  HealthCheckResponse,
} from '../types/api';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081';

// Helper function to handle API responses
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || error.message || 'API request failed');
  }
  return response.json();
}

// Health check
export async function healthCheck(): Promise<HealthCheckResponse> {
  const response = await fetch(`${API_BASE_URL}/health`);
  return handleResponse<HealthCheckResponse>(response);
}

// Create a new orchestration session
export async function createSession(
  request: CreateSessionRequest
): Promise<CreateSessionResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/sessions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });
  return handleResponse<CreateSessionResponse>(response);
}

// List all sessions
export async function listSessions(): Promise<SessionListResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/sessions`);
  return handleResponse<SessionListResponse>(response);
}

// Get session status
export async function getSessionStatus(sessionId: string): Promise<Session> {
  const response = await fetch(`${API_BASE_URL}/api/v1/sessions/${sessionId}`);
  return handleResponse<Session>(response);
}

// Get session result
export async function getSessionResult(sessionId: string): Promise<OrchestrationResult> {
  const response = await fetch(`${API_BASE_URL}/api/v1/sessions/${sessionId}/result`);
  return handleResponse<OrchestrationResult>(response);
}

// Get complete session history (all events and data)
export async function getSessionHistory(sessionId: string): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/api/v1/sessions/${sessionId}/history`);
  return handleResponse<any>(response);
}

// Stream session progress using Server-Sent Events
export function streamSessionProgress(
  sessionId: string,
  onProgress: (event: ProgressEvent) => void,
  onError?: (error: Event) => void,
  onComplete?: () => void
): EventSource {
  const eventSource = new EventSource(
    `${API_BASE_URL}/api/v1/sessions/${sessionId}/stream`
  );

  eventSource.addEventListener('connected', (e) => {
    console.log('Connected to session stream:', JSON.parse(e.data));
  });

  eventSource.addEventListener('progress', (e) => {
    try {
      const progressEvent: ProgressEvent = JSON.parse(e.data);
      onProgress(progressEvent);

      // Close connection when completed or failed
      if (progressEvent.status === 'completed' || progressEvent.status === 'failed') {
        eventSource.close();
        onComplete?.();
      }
    } catch (error) {
      console.error('Failed to parse progress event:', error);
    }
  });

  eventSource.addEventListener('close', () => {
    eventSource.close();
    onComplete?.();
  });

  eventSource.addEventListener('error', (e) => {
    console.error('SSE error:', e);
    onError?.(e);
    eventSource.close();
  });

  return eventSource;
}

// Legacy synchronous orchestration (for reference)
export async function orchestrateWorkflow(
  request: CreateSessionRequest
): Promise<OrchestrationResult> {
  const response = await fetch(`${API_BASE_URL}/api/v1/orchestrate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });
  return handleResponse<OrchestrationResult>(response);
}
