// Late swap notifications component
import React, { useEffect, useState } from 'react'
import { useRealtimeStore, useLateSwapNotifications, useUnreadCounts } from '@/store/realtime'
import { apiPost, apiDelete } from '@/services/apiClient'
import type { LateSwapNotification } from '@/services/websocketService'

interface LateSwapNotificationsProps {
  className?: string
  maxNotifications?: number
  autoConnect?: boolean
}

const LateSwapNotifications: React.FC<LateSwapNotificationsProps> = ({ 
  className = '', 
  maxNotifications = 10,
  autoConnect = true 
}) => {
  const notifications = useLateSwapNotifications()
  const { lateSwaps: unreadCount } = useUnreadCounts()
  const { connect, disconnect, markLateSwapAsRead, markAllLateSwapsAsRead } = useRealtimeStore()
  const [processingIds, setProcessingIds] = useState<Set<string>>(new Set())

  useEffect(() => {
    if (autoConnect) {
      connect()
      return () => disconnect()
    }
  }, [autoConnect, connect, disconnect])

  const displayNotifications = notifications.slice(0, maxNotifications)

  const handleApprove = async (notification: LateSwapNotification) => {
    if (processingIds.has(notification.id)) return
    
    setProcessingIds(prev => new Set([...prev, notification.id]))
    
    try {
      await apiPost(`/lateswap/recommendations/${notification.id}/approve`)
      markLateSwapAsRead(notification.id)
      
      // Show success message
      console.log('Late swap approved successfully')
    } catch (error) {
      console.error('Failed to approve late swap:', error)
      // Handle error (show notification, etc.)
    } finally {
      setProcessingIds(prev => {
        const newSet = new Set(prev)
        newSet.delete(notification.id)
        return newSet
      })
    }
  }

  const handleReject = async (notification: LateSwapNotification) => {
    if (processingIds.has(notification.id)) return
    
    setProcessingIds(prev => new Set([...prev, notification.id]))
    
    try {
      await apiPost(`/lateswap/recommendations/${notification.id}/reject`)
      markLateSwapAsRead(notification.id)
      
      // Show success message
      console.log('Late swap rejected successfully')
    } catch (error) {
      console.error('Failed to reject late swap:', error)
      // Handle error (show notification, etc.)
    } finally {
      setProcessingIds(prev => {
        const newSet = new Set(prev)
        newSet.delete(notification.id)
        return newSet
      })
    }
  }

  const getRiskColor = (riskScore: number): string => {
    if (riskScore >= 0.7) return 'text-red-600 bg-red-50'
    if (riskScore >= 0.4) return 'text-orange-600 bg-orange-50'
    return 'text-green-600 bg-green-50'
  }

  const getRecommendationIcon = (type: string): string => {
    switch (type) {
      case 'injury': return '🏥'
      case 'weather': return '🌦️'
      case 'ownership': return '📊'
      case 'projection': return '📈'
      case 'value': return '💰'
      case 'news': return '📰'
      default: return '🔄'
    }
  }

  const formatTimeRemaining = (expiresAt: string): string => {
    const now = new Date()
    const expiry = new Date(expiresAt)
    const diffMs = expiry.getTime() - now.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    
    if (diffMins <= 0) return 'Expired'
    if (diffMins < 60) return `${diffMins}m left`
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h left`
    return `${Math.floor(diffMins / 1440)}d left`
  }

  const isExpired = (expiresAt: string): boolean => {
    return new Date(expiresAt) <= new Date()
  }

  return (
    <div className={`bg-white rounded-lg shadow-sm border ${className}`}>
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <h3 className="text-lg font-semibold text-gray-900">Late Swap Recommendations</h3>
            {unreadCount > 0 && (
              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                {unreadCount} new
              </span>
            )}
          </div>
          
          <div className="flex items-center space-x-2">
            <span className="text-sm text-gray-500">{notifications.length} total</span>
            {notifications.length > 0 && (
              <button
                onClick={markAllLateSwapsAsRead}
                className="text-xs text-gray-400 hover:text-gray-600 transition-colors"
              >
                Mark all read
              </button>
            )}
          </div>
        </div>
      </div>

      <div className="max-h-96 overflow-y-auto">
        {displayNotifications.length === 0 ? (
          <div className="p-6 text-center text-gray-500">
            <div className="text-2xl mb-2">🔄</div>
            <p>No late swap recommendations</p>
            <p className="text-sm">Recommendations will appear here when available</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {displayNotifications.map((notification) => {
              const isProcessing = processingIds.has(notification.id)
              const expired = isExpired(notification.expires_at)
              
              return (
                <div key={notification.id} className={`p-4 ${expired ? 'opacity-60' : ''}`}>
                  <div className="flex items-start space-x-3">
                    <div className="text-lg">
                      {getRecommendationIcon(notification.recommendation_type)}
                    </div>
                    
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between mb-2">
                        <p className="text-sm font-medium text-gray-900">
                          Late Swap Recommendation
                        </p>
                        <span className={`text-xs ${expired ? 'text-red-500' : 'text-gray-500'}`}>
                          {formatTimeRemaining(notification.expires_at)}
                        </span>
                      </div>
                      
                      <p className="text-sm text-gray-600 mb-3">
                        {notification.reason}
                      </p>
                      
                      <div className="flex items-center space-x-2 mb-3">
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                          Impact: {notification.impact_score.toFixed(1)}
                        </span>
                        
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                          {Math.round(notification.confidence_score * 100)}% confidence
                        </span>
                        
                        <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getRiskColor(notification.risk_score)}`}>
                          Risk: {Math.round(notification.risk_score * 100)}%
                        </span>
                        
                        {notification.auto_approval_eligible && (
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                            Auto-eligible
                          </span>
                        )}
                      </div>
                      
                      <div className="text-xs text-gray-500 mb-3">
                        Original Player ID: {notification.original_player_id} → 
                        Recommended Player ID: {notification.recommended_player_id}
                      </div>
                      
                      {!expired && (
                        <div className="flex items-center space-x-2">
                          <button
                            onClick={() => handleApprove(notification)}
                            disabled={isProcessing}
                            className="inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            {isProcessing ? 'Processing...' : 'Approve'}
                          </button>
                          
                          <button
                            onClick={() => handleReject(notification)}
                            disabled={isProcessing}
                            className="inline-flex items-center px-3 py-1 border border-gray-300 text-xs font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            {isProcessing ? 'Processing...' : 'Reject'}
                          </button>
                          
                          <button
                            onClick={() => markLateSwapAsRead(notification.id)}
                            className="text-xs text-gray-400 hover:text-gray-600 transition-colors"
                          >
                            Mark as read
                          </button>
                        </div>
                      )}
                      
                      {expired && (
                        <div className="text-xs text-red-500 font-medium">
                          This recommendation has expired
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

export default LateSwapNotifications