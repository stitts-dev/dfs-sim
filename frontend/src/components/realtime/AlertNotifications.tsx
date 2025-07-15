// Alert notifications component
import React, { useEffect } from 'react'
import { useRealtimeStore, useAlertNotifications, useUnreadCounts } from '@/store/realtime'
import type { AlertNotification } from '@/services/websocketService'

interface AlertNotificationsProps {
  className?: string
  maxAlerts?: number
  autoConnect?: boolean
  showActions?: boolean
}

const AlertNotifications: React.FC<AlertNotificationsProps> = ({ 
  className = '', 
  maxAlerts = 15,
  autoConnect = true,
  showActions = true
}) => {
  const alerts = useAlertNotifications()
  const { alerts: unreadCount } = useUnreadCounts()
  const { connect, disconnect, markAlertAsRead, markAllAlertsAsRead, dismissAlert } = useRealtimeStore()

  useEffect(() => {
    if (autoConnect) {
      connect()
      return () => disconnect()
    }
  }, [autoConnect, connect, disconnect])

  const displayAlerts = alerts.slice(0, maxAlerts)

  const getAlertIcon = (alertType: string, priority: string): string => {
    if (priority === 'critical') return '🚨'
    
    switch (alertType) {
      case 'player_injury': return '🏥'
      case 'weather_alert': return '🌦️'
      case 'lineup_alert': return '📋'
      case 'contest_alert': return '🏆'
      case 'optimization_alert': return '⚡'
      case 'ownership_alert': return '📊'
      case 'value_alert': return '💰'
      case 'news_alert': return '📰'
      case 'system_alert': return '⚙️'
      default: return '🔔'
    }
  }

  const getAlertColor = (priority: string): string => {
    switch (priority) {
      case 'critical': return 'border-l-red-500 bg-red-50'
      case 'high': return 'border-l-orange-500 bg-orange-50'
      case 'medium': return 'border-l-yellow-500 bg-yellow-50'
      case 'low': return 'border-l-blue-500 bg-blue-50'
      default: return 'border-l-gray-500 bg-gray-50'
    }
  }

  const getPriorityTextColor = (priority: string): string => {
    switch (priority) {
      case 'critical': return 'text-red-700'
      case 'high': return 'text-orange-700'
      case 'medium': return 'text-yellow-700'
      case 'low': return 'text-blue-700'
      default: return 'text-gray-700'
    }
  }

  const formatAlertTime = (timestamp: string): string => {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    
    if (diffMins < 1) return 'Just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`
    return date.toLocaleDateString()
  }

  const isExpired = (expiresAt?: string): boolean => {
    if (!expiresAt) return false
    return new Date(expiresAt) <= new Date()
  }

  const formatExpiryTime = (expiresAt?: string): string => {
    if (!expiresAt) return ''
    
    const now = new Date()
    const expiry = new Date(expiresAt)
    const diffMs = expiry.getTime() - now.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    
    if (diffMins <= 0) return 'Expired'
    if (diffMins < 60) return `Expires in ${diffMins}m`
    if (diffMins < 1440) return `Expires in ${Math.floor(diffMins / 60)}h`
    return `Expires in ${Math.floor(diffMins / 1440)}d`
  }

  const handleDismiss = (alert: AlertNotification) => {
    markAlertAsRead(alert.id)
    dismissAlert(alert.id)
  }

  const handleMarkAsRead = (alert: AlertNotification) => {
    markAlertAsRead(alert.id)
  }

  return (
    <div className={`bg-white rounded-lg shadow-sm border ${className}`}>
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <h3 className="text-lg font-semibold text-gray-900">Alerts</h3>
            {unreadCount > 0 && (
              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800">
                {unreadCount} new
              </span>
            )}
          </div>
          
          <div className="flex items-center space-x-2">
            <span className="text-sm text-gray-500">{alerts.length} total</span>
            {alerts.length > 0 && showActions && (
              <button
                onClick={markAllAlertsAsRead}
                className="text-xs text-gray-400 hover:text-gray-600 transition-colors"
              >
                Mark all read
              </button>
            )}
          </div>
        </div>
      </div>

      <div className="max-h-96 overflow-y-auto">
        {displayAlerts.length === 0 ? (
          <div className="p-6 text-center text-gray-500">
            <div className="text-2xl mb-2">🔔</div>
            <p>No alerts</p>
            <p className="text-sm">Alert notifications will appear here</p>
          </div>
        ) : (
          <div className="space-y-1">
            {displayAlerts.map((alert) => {
              const expired = isExpired(alert.expires_at)
              
              return (
                <div 
                  key={alert.id} 
                  className={`p-4 border-l-4 ${getAlertColor(alert.priority)} ${expired ? 'opacity-60' : ''}`}
                >
                  <div className="flex items-start space-x-3">
                    <div className="text-lg">
                      {getAlertIcon(alert.alert_type, alert.priority)}
                    </div>
                    
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between mb-1">
                        <p className="text-sm font-medium text-gray-900">
                          {alert.title}
                        </p>
                        <div className="flex items-center space-x-2">
                          <span className={`text-xs font-medium uppercase ${getPriorityTextColor(alert.priority)}`}>
                            {alert.priority}
                          </span>
                          <span className="text-xs text-gray-500">
                            {formatAlertTime(alert.created_at)}
                          </span>
                        </div>
                      </div>
                      
                      <p className="text-sm text-gray-600 mb-2">
                        {alert.message}
                      </p>
                      
                      {alert.expires_at && (
                        <p className={`text-xs mb-2 ${expired ? 'text-red-500' : 'text-gray-500'}`}>
                          {formatExpiryTime(alert.expires_at)}
                        </p>
                      )}
                      
                      {/* Display additional data if available */}
                      {alert.data && Object.keys(alert.data).length > 0 && (
                        <div className="text-xs text-gray-500 mb-2">
                          {Object.entries(alert.data).map(([key, value]) => (
                            <span key={key} className="mr-3">
                              {key}: {String(value)}
                            </span>
                          ))}
                        </div>
                      )}
                      
                      {showActions && (
                        <div className="flex items-center space-x-2">
                          <button
                            onClick={() => handleMarkAsRead(alert)}
                            className="text-xs text-blue-600 hover:text-blue-800 transition-colors"
                          >
                            Mark as read
                          </button>
                          
                          <button
                            onClick={() => handleDismiss(alert)}
                            className="text-xs text-gray-400 hover:text-gray-600 transition-colors"
                          >
                            Dismiss
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

export default AlertNotifications