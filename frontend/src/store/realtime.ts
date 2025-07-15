// Real-time data store using Zustand
import { create } from 'zustand'
import { subscribeWithSelector } from 'zustand/middleware'
import type { 
  RealtimeEvent, 
  LateSwapNotification, 
  AlertNotification,
  WebSocketEventHandlers 
} from '@/services/websocketService'
import { 
  websocketService,
  connectToRealtimeEvents,
  connectToLateSwapNotifications,
  connectToAlertNotifications
} from '@/services/websocketService'

export interface RealtimeStore {
  // Connection state
  isConnected: boolean
  connectionStatus: Record<string, boolean>
  
  // Real-time events
  events: RealtimeEvent[]
  eventsLoading: boolean
  
  // Late swap notifications
  lateSwapNotifications: LateSwapNotification[]
  unreadLateSwaps: number
  
  // Alert notifications
  alertNotifications: AlertNotification[]
  unreadAlerts: number
  
  // Actions
  connect: () => void
  disconnect: () => void
  addEvent: (event: RealtimeEvent) => void
  addLateSwapNotification: (notification: LateSwapNotification) => void
  addAlertNotification: (alert: AlertNotification) => void
  markLateSwapAsRead: (id: string) => void
  markAlertAsRead: (id: string) => void
  markAllLateSwapsAsRead: () => void
  markAllAlertsAsRead: () => void
  clearEvents: () => void
  clearNotifications: () => void
  dismissAlert: (id: string) => void
  setConnectionStatus: (status: Record<string, boolean>) => void
}

export const useRealtimeStore = create<RealtimeStore>()(
  subscribeWithSelector((set, get) => ({
    // Initial state
    isConnected: false,
    connectionStatus: {},
    events: [],
    eventsLoading: false,
    lateSwapNotifications: [],
    unreadLateSwaps: 0,
    alertNotifications: [],
    unreadAlerts: 0,

    // Connect to WebSocket services
    connect: () => {
      console.log('Connecting to real-time services...')
      
      const handlers: WebSocketEventHandlers = {
        onConnect: () => {
          console.log('Real-time WebSocket connected')
          set((state) => ({ 
            isConnected: true,
            connectionStatus: { ...state.connectionStatus, realtime: true }
          }))
        },
        
        onDisconnect: () => {
          console.log('Real-time WebSocket disconnected')
          set((state) => ({ 
            isConnected: false,
            connectionStatus: { ...state.connectionStatus, realtime: false }
          }))
        },
        
        onError: (error) => {
          console.error('Real-time WebSocket error:', error)
          set((state) => ({ 
            connectionStatus: { ...state.connectionStatus, realtime: false }
          }))
        },
        
        onRealtimeEvent: (event) => {
          console.log('Received real-time event:', event)
          get().addEvent(event)
        },
        
        onReconnect: () => {
          console.log('Real-time WebSocket reconnected')
          set((state) => ({ 
            connectionStatus: { ...state.connectionStatus, realtime: true }
          }))
        }
      }

      const lateSwapHandlers: WebSocketEventHandlers = {
        onConnect: () => {
          console.log('Late swap WebSocket connected')
          set((state) => ({ 
            connectionStatus: { ...state.connectionStatus, lateswap: true }
          }))
        },
        
        onDisconnect: () => {
          console.log('Late swap WebSocket disconnected')
          set((state) => ({ 
            connectionStatus: { ...state.connectionStatus, lateswap: false }
          }))
        },
        
        onLateSwapNotification: (notification) => {
          console.log('Received late swap notification:', notification)
          get().addLateSwapNotification(notification)
        }
      }

      const alertHandlers: WebSocketEventHandlers = {
        onConnect: () => {
          console.log('Alert WebSocket connected')
          set((state) => ({ 
            connectionStatus: { ...state.connectionStatus, alerts: true }
          }))
        },
        
        onDisconnect: () => {
          console.log('Alert WebSocket disconnected')
          set((state) => ({ 
            connectionStatus: { ...state.connectionStatus, alerts: false }
          }))
        },
        
        onAlertNotification: (alert) => {
          console.log('Received alert notification:', alert)
          get().addAlertNotification(alert)
        }
      }

      // Connect to all real-time endpoints
      connectToRealtimeEvents(handlers)
      connectToLateSwapNotifications(lateSwapHandlers)
      connectToAlertNotifications(alertHandlers)
    },

    // Disconnect from WebSocket services
    disconnect: () => {
      console.log('Disconnecting from real-time services...')
      websocketService.disconnectAll()
      set({
        isConnected: false,
        connectionStatus: {}
      })
    },

    // Add a new real-time event
    addEvent: (event: RealtimeEvent) => {
      set((state) => ({
        events: [event, ...state.events.slice(0, 99)] // Keep last 100 events
      }))
    },

    // Add a new late swap notification
    addLateSwapNotification: (notification: LateSwapNotification) => {
      set((state) => ({
        lateSwapNotifications: [notification, ...state.lateSwapNotifications],
        unreadLateSwaps: state.unreadLateSwaps + 1
      }))
    },

    // Add a new alert notification
    addAlertNotification: (alert: AlertNotification) => {
      set((state) => ({
        alertNotifications: [alert, ...state.alertNotifications],
        unreadAlerts: state.unreadAlerts + 1
      }))
    },

    // Mark a late swap notification as read
    markLateSwapAsRead: (id: string) => {
      set((state) => {
        const notification = state.lateSwapNotifications.find(n => n.id === id)
        if (notification) {
          return {
            unreadLateSwaps: Math.max(0, state.unreadLateSwaps - 1)
          }
        }
        return state
      })
    },

    // Mark an alert notification as read
    markAlertAsRead: (id: string) => {
      set((state) => {
        const alert = state.alertNotifications.find(a => a.id === id)
        if (alert) {
          return {
            unreadAlerts: Math.max(0, state.unreadAlerts - 1)
          }
        }
        return state
      })
    },

    // Mark all late swap notifications as read
    markAllLateSwapsAsRead: () => {
      set({ unreadLateSwaps: 0 })
    },

    // Mark all alert notifications as read
    markAllAlertsAsRead: () => {
      set({ unreadAlerts: 0 })
    },

    // Clear all events
    clearEvents: () => {
      set({ events: [] })
    },

    // Clear all notifications
    clearNotifications: () => {
      set({
        lateSwapNotifications: [],
        alertNotifications: [],
        unreadLateSwaps: 0,
        unreadAlerts: 0
      })
    },

    // Dismiss an alert (remove from list)
    dismissAlert: (id: string) => {
      set((state) => ({
        alertNotifications: state.alertNotifications.filter(alert => alert.id !== id)
      }))
    },

    // Update connection status
    setConnectionStatus: (status: Record<string, boolean>) => {
      set((state) => ({
        connectionStatus: { ...state.connectionStatus, ...status },
        isConnected: Object.values(status).some(Boolean)
      }))
    }
  }))
)

// Selector hooks for better performance
export const useRealtimeEvents = () => useRealtimeStore((state) => state.events)
export const useLateSwapNotifications = () => useRealtimeStore((state) => state.lateSwapNotifications)
export const useAlertNotifications = () => useRealtimeStore((state) => state.alertNotifications)
export const useUnreadCounts = () => useRealtimeStore((state) => ({
  lateSwaps: state.unreadLateSwaps,
  alerts: state.unreadAlerts
}))
export const useConnectionStatus = () => useRealtimeStore((state) => ({
  isConnected: state.isConnected,
  status: state.connectionStatus
}))

// Effect hooks for automatic connection management
export const useRealtimeConnection = () => {
  const { connect, disconnect } = useRealtimeStore()
  
  return {
    connect,
    disconnect,
    // Auto-connect when component mounts, disconnect when unmounts
    useEffect: () => {
      connect()
      return () => disconnect()
    }
  }
}