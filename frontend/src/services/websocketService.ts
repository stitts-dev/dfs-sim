// WebSocket service for real-time data integration
import { useAuthStore } from '@/store/auth'

export interface WebSocketMessage {
  type: string
  payload: any
  timestamp: number
}

export interface RealtimeEvent {
  id: string
  event_type: string
  player_id?: number
  impact_rating: number
  confidence: number
  event_data: Record<string, any>
  created_at: string
}

export interface LateSwapNotification {
  id: string
  user_id: number
  recommendation_type: string
  original_player_id: number
  recommended_player_id: number
  impact_score: number
  confidence_score: number
  risk_score: number
  auto_approval_eligible: boolean
  expires_at: string
  reason: string
}

export interface AlertNotification {
  id: string
  user_id: number
  alert_type: string
  priority: string
  title: string
  message: string
  data: Record<string, any>
  created_at: string
  expires_at?: string
}

export type WebSocketEventType = 
  | 'realtime_event'
  | 'lateswap_notification' 
  | 'alert_notification'
  | 'ownership_update'
  | 'optimization_progress'
  | 'ai_recommendation'

export interface WebSocketEventHandlers {
  onRealtimeEvent?: (event: RealtimeEvent) => void
  onLateSwapNotification?: (notification: LateSwapNotification) => void
  onAlertNotification?: (alert: AlertNotification) => void
  onOwnershipUpdate?: (data: any) => void
  onOptimizationProgress?: (progress: any) => void
  onAIRecommendation?: (recommendation: any) => void
  onConnect?: () => void
  onDisconnect?: () => void
  onError?: (error: Event) => void
  onReconnect?: () => void
}

export class WebSocketService {
  private connections: Map<string, WebSocket> = new Map()
  private reconnectAttempts: Map<string, number> = new Map()
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private handlers: Map<string, WebSocketEventHandlers> = new Map()
  private baseUrl: string

  constructor() {
    this.baseUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'
  }

  // Connect to a specific WebSocket endpoint
  connect(endpoint: string, handlers: WebSocketEventHandlers): void {
    const { token, user } = useAuthStore.getState()
    
    if (!token || !user) {
      console.warn('Cannot connect to WebSocket: no authentication token or user')
      return
    }

    this.handlers.set(endpoint, handlers)
    this.createConnection(endpoint, user.id, token)
  }

  // Disconnect from a specific WebSocket endpoint
  disconnect(endpoint: string): void {
    const connection = this.connections.get(endpoint)
    if (connection) {
      connection.close()
      this.connections.delete(endpoint)
      this.handlers.delete(endpoint)
      this.reconnectAttempts.delete(endpoint)
    }
  }

  // Disconnect from all WebSocket connections
  disconnectAll(): void {
    this.connections.forEach((connection, endpoint) => {
      connection.close()
    })
    this.connections.clear()
    this.handlers.clear()
    this.reconnectAttempts.clear()
  }

  // Send message to a specific WebSocket connection
  send(endpoint: string, message: any): void {
    const connection = this.connections.get(endpoint)
    if (connection && connection.readyState === WebSocket.OPEN) {
      connection.send(JSON.stringify(message))
    } else {
      console.warn(`Cannot send message: WebSocket connection for ${endpoint} is not open`)
    }
  }

  // Check if a connection is active
  isConnected(endpoint: string): boolean {
    const connection = this.connections.get(endpoint)
    return connection ? connection.readyState === WebSocket.OPEN : false
  }

  // Get connection status for all connections
  getConnectionStatus(): Record<string, boolean> {
    const status: Record<string, boolean> = {}
    this.connections.forEach((connection, endpoint) => {
      status[endpoint] = connection.readyState === WebSocket.OPEN
    })
    return status
  }

  private createConnection(endpoint: string, userId: string, token: string): void {
    try {
      // Construct WebSocket URL with authentication
      const wsUrl = `${this.baseUrl}/${endpoint}/${userId}?token=${encodeURIComponent(token)}`
      const connection = new WebSocket(wsUrl)

      connection.onopen = () => {
        console.log(`WebSocket connected: ${endpoint}`)
        this.reconnectAttempts.set(endpoint, 0)
        const handlers = this.handlers.get(endpoint)
        handlers?.onConnect?.()
      }

      connection.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          this.handleMessage(endpoint, message)
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
        }
      }

      connection.onclose = (event) => {
        console.log(`WebSocket disconnected: ${endpoint}`, event.code, event.reason)
        this.connections.delete(endpoint)
        
        const handlers = this.handlers.get(endpoint)
        handlers?.onDisconnect?.()

        // Attempt reconnection if not manually closed
        if (event.code !== 1000) {
          this.attemptReconnection(endpoint, userId, token)
        }
      }

      connection.onerror = (error) => {
        console.error(`WebSocket error: ${endpoint}`, error)
        const handlers = this.handlers.get(endpoint)
        handlers?.onError?.(error)
      }

      this.connections.set(endpoint, connection)
    } catch (error) {
      console.error(`Failed to create WebSocket connection for ${endpoint}:`, error)
    }
  }

  private handleMessage(endpoint: string, message: WebSocketMessage): void {
    const handlers = this.handlers.get(endpoint)
    if (!handlers) return

    switch (message.type) {
      case 'realtime_event':
        handlers.onRealtimeEvent?.(message.payload as RealtimeEvent)
        break
      
      case 'lateswap_notification':
        handlers.onLateSwapNotification?.(message.payload as LateSwapNotification)
        break
      
      case 'alert_notification':
        handlers.onAlertNotification?.(message.payload as AlertNotification)
        break
      
      case 'ownership_update':
        handlers.onOwnershipUpdate?.(message.payload)
        break
      
      case 'optimization_progress':
        handlers.onOptimizationProgress?.(message.payload)
        break
      
      case 'ai_recommendation':
        handlers.onAIRecommendation?.(message.payload)
        break
      
      default:
        console.warn(`Unknown WebSocket message type: ${message.type}`)
    }
  }

  private attemptReconnection(endpoint: string, userId: string, token: string): void {
    const attempts = this.reconnectAttempts.get(endpoint) || 0
    
    if (attempts >= this.maxReconnectAttempts) {
      console.error(`Max reconnection attempts reached for ${endpoint}`)
      return
    }

    const delay = this.reconnectDelay * Math.pow(2, attempts) // Exponential backoff
    
    console.log(`Attempting to reconnect to ${endpoint} in ${delay}ms (attempt ${attempts + 1})`)
    
    setTimeout(() => {
      if (!this.connections.has(endpoint)) { // Only reconnect if not already connected
        this.reconnectAttempts.set(endpoint, attempts + 1)
        this.createConnection(endpoint, userId, token)
        
        const handlers = this.handlers.get(endpoint)
        handlers?.onReconnect?.()
      }
    }, delay)
  }
}

// Create singleton instance
export const websocketService = new WebSocketService()

// Convenience functions for specific endpoints
export const connectToRealtimeEvents = (handlers: WebSocketEventHandlers) => {
  websocketService.connect('realtime-events', handlers)
}

export const connectToLateSwapNotifications = (handlers: WebSocketEventHandlers) => {
  websocketService.connect('lateswap-notifications', handlers)
}

export const connectToAlertNotifications = (handlers: WebSocketEventHandlers) => {
  websocketService.connect('alert-notifications', handlers)
}

export const connectToOptimizationProgress = (handlers: WebSocketEventHandlers) => {
  websocketService.connect('optimization-progress', handlers)
}

export const connectToAIRecommendations = (handlers: WebSocketEventHandlers) => {
  websocketService.connect('ai-recommendations', handlers)
}

// Auto-cleanup on page unload
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => {
    websocketService.disconnectAll()
  })
}