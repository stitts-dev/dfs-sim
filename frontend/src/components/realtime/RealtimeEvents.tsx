// Real-time events display component
import React, { useEffect } from 'react'
import { useRealtimeStore, useRealtimeEvents, useConnectionStatus } from '@/store/realtime'
import type { RealtimeEvent } from '@/services/websocketService'

interface RealtimeEventsProps {
  className?: string
  maxEvents?: number
  autoConnect?: boolean
}

const RealtimeEvents: React.FC<RealtimeEventsProps> = ({ 
  className = '', 
  maxEvents = 20,
  autoConnect = true 
}) => {
  const events = useRealtimeEvents()
  const { isConnected } = useConnectionStatus()
  const { connect, disconnect, clearEvents } = useRealtimeStore()

  useEffect(() => {
    if (autoConnect) {
      connect()
      return () => disconnect()
    }
  }, [autoConnect, connect, disconnect])

  const displayEvents = events.slice(0, maxEvents)

  const getEventIcon = (eventType: string): string => {
    switch (eventType) {
      case 'player_injury': return '🏥'
      case 'weather_update': return '🌦️'
      case 'ownership_change': return '📊'
      case 'news_update': return '📰'
      case 'line_movement': return '📈'
      case 'contest_update': return '🏆'
      default: return '⚡'
    }
  }

  const getEventColor = (impactRating: number): string => {
    if (impactRating >= 8) return 'text-red-600 bg-red-50'
    if (impactRating >= 6) return 'text-orange-600 bg-orange-50'
    if (impactRating >= 4) return 'text-yellow-600 bg-yellow-50'
    return 'text-blue-600 bg-blue-50'
  }

  const formatEventTime = (timestamp: string): string => {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    
    if (diffMins < 1) return 'Just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`
    return date.toLocaleDateString()
  }

  const formatEventDescription = (event: RealtimeEvent): string => {
    const { event_type, event_data } = event
    
    switch (event_type) {
      case 'player_injury':
        return `${event_data.player_name} - ${event_data.injury_status || 'Injury update'}`
      case 'weather_update':
        return `Weather: ${event_data.condition} at ${event_data.venue}`
      case 'ownership_change':
        return `${event_data.player_name} ownership: ${event_data.old_ownership}% → ${event_data.new_ownership}%`
      case 'news_update':
        return event_data.headline || 'Breaking news update'
      case 'line_movement':
        return `Line moved: ${event_data.old_line} → ${event_data.new_line}`
      case 'contest_update':
        return `Contest: ${event_data.contest_name} - ${event_data.update_type}`
      default:
        return event_data.description || 'Real-time update'
    }
  }

  return (
    <div className={`bg-white rounded-lg shadow-sm border ${className}`}>
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <h3 className="text-lg font-semibold text-gray-900">Live Events</h3>
            <div className={`h-2 w-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
            <span className={`text-xs ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
              {isConnected ? 'Connected' : 'Disconnected'}
            </span>
          </div>
          
          <div className="flex items-center space-x-2">
            <span className="text-sm text-gray-500">{events.length} events</span>
            {events.length > 0 && (
              <button
                onClick={clearEvents}
                className="text-xs text-gray-400 hover:text-gray-600 transition-colors"
              >
                Clear
              </button>
            )}
          </div>
        </div>
      </div>

      <div className="max-h-96 overflow-y-auto">
        {displayEvents.length === 0 ? (
          <div className="p-6 text-center text-gray-500">
            <div className="text-2xl mb-2">⚡</div>
            <p>No live events yet</p>
            <p className="text-sm">Real-time updates will appear here</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {displayEvents.map((event) => (
              <div key={event.id} className="p-4 hover:bg-gray-50 transition-colors">
                <div className="flex items-start space-x-3">
                  <div className="text-lg">{getEventIcon(event.event_type)}</div>
                  
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between mb-1">
                      <p className="text-sm font-medium text-gray-900 truncate">
                        {formatEventDescription(event)}
                      </p>
                      <span className="text-xs text-gray-500 ml-2">
                        {formatEventTime(event.created_at)}
                      </span>
                    </div>
                    
                    <div className="flex items-center space-x-2">
                      <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getEventColor(event.impact_rating)}`}>
                        Impact: {event.impact_rating}/10
                      </span>
                      
                      <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                        {Math.round(event.confidence * 100)}% confidence
                      </span>
                      
                      {event.player_id && (
                        <span className="text-xs text-gray-500">
                          Player ID: {event.player_id}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {!isConnected && (
        <div className="p-3 bg-yellow-50 border-t border-yellow-200">
          <div className="flex items-center space-x-2">
            <div className="text-yellow-600">⚠️</div>
            <p className="text-sm text-yellow-800">
              Disconnected from live events. 
              <button 
                onClick={connect} 
                className="ml-1 text-yellow-900 underline hover:no-underline"
              >
                Reconnect
              </button>
            </p>
          </div>
        </div>
      )}
    </div>
  )
}

export default RealtimeEvents