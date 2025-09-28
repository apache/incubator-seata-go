import { useWebSocketContext } from '../contexts/WebSocketContext';

export const useWebSocket = () => {
  return useWebSocketContext();
};