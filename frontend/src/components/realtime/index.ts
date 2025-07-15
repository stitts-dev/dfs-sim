// Real-time components exports
export { default as RealtimeEvents } from './RealtimeEvents'
export { default as LateSwapNotifications } from './LateSwapNotifications'
export { default as AlertNotifications } from './AlertNotifications'
export { default as RealtimeDashboard } from './RealtimeDashboard'

// Re-export store hooks for convenience
export {
  useRealtimeStore,
  useRealtimeEvents,
  useLateSwapNotifications,
  useAlertNotifications,
  useUnreadCounts,
  useConnectionStatus,
  useRealtimeConnection
} from '@/store/realtime'

// Re-export WebSocket service types for convenience
export type {
  WebSocketMessage,
  RealtimeEvent,
  LateSwapNotification,
  AlertNotification,
  WebSocketEventType,
  WebSocketEventHandlers
} from '@/services/websocketService'

export {
  websocketService,
  connectToRealtimeEvents,
  connectToLateSwapNotifications,
  connectToAlertNotifications,
  connectToOptimizationProgress,
  connectToAIRecommendations
} from '@/services/websocketService'