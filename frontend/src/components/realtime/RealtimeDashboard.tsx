// Real-time dashboard component that integrates all real-time features
import React, { useEffect, useState } from 'react'
import { useRealtimeStore, useConnectionStatus, useUnreadCounts } from '@/store/realtime'
import RealtimeEvents from './RealtimeEvents'
import LateSwapNotifications from './LateSwapNotifications'
import AlertNotifications from './AlertNotifications'

interface RealtimeDashboardProps {
  className?: string
  showHeader?: boolean
  defaultTab?: 'events' | 'lateswaps' | 'alerts'
}

const RealtimeDashboard: React.FC<RealtimeDashboardProps> = ({ 
  className = '',
  showHeader = true,
  defaultTab = 'events'
}) => {
  const [activeTab, setActiveTab] = useState(defaultTab)
  const { isConnected, status } = useConnectionStatus()
  const { lateSwaps, alerts } = useUnreadCounts()
  const { connect, disconnect } = useRealtimeStore()

  useEffect(() => {
    // Auto-connect when component mounts
    connect()
    
    // Cleanup on unmount
    return () => disconnect()
  }, [connect, disconnect])

  const tabs = [
    {
      id: 'events',
      label: 'Live Events',
      icon: '⚡',
      badge: null,
      component: RealtimeEvents
    },
    {
      id: 'lateswaps',
      label: 'Late Swaps',
      icon: '🔄',
      badge: lateSwaps > 0 ? lateSwaps : null,
      component: LateSwapNotifications
    },
    {
      id: 'alerts',
      label: 'Alerts',
      icon: '🔔',
      badge: alerts > 0 ? alerts : null,
      component: AlertNotifications
    }
  ]

  const getConnectionStatusColor = (): string => {
    if (!isConnected) return 'bg-red-500'
    
    const connectedServices = Object.values(status).filter(Boolean).length
    const totalServices = Object.keys(status).length
    
    if (connectedServices === totalServices) return 'bg-green-500'
    if (connectedServices > 0) return 'bg-yellow-500'
    return 'bg-red-500'
  }

  const getConnectionStatusText = (): string => {
    if (!isConnected) return 'Disconnected'
    
    const connectedServices = Object.values(status).filter(Boolean).length
    const totalServices = Object.keys(status).length
    
    if (connectedServices === totalServices) return 'All Services Connected'
    if (connectedServices > 0) return `${connectedServices}/${totalServices} Connected`
    return 'Disconnected'
  }

  const ActiveComponent = tabs.find(tab => tab.id === activeTab)?.component || RealtimeEvents

  return (
    <div className={`bg-white rounded-lg shadow-sm border ${className}`}>
      {showHeader && (
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center space-x-3">
              <h2 className="text-xl font-semibold text-gray-900">Real-Time Dashboard</h2>
              <div className="flex items-center space-x-2">
                <div className={`h-2 w-2 rounded-full ${getConnectionStatusColor()}`} />
                <span className="text-sm text-gray-600">{getConnectionStatusText()}</span>
              </div>
            </div>
            
            <div className="flex items-center space-x-2">
              {!isConnected && (
                <button
                  onClick={connect}
                  className="inline-flex items-center px-3 py-1 border border-gray-300 text-xs font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                >
                  Reconnect
                </button>
              )}
              
              <button
                onClick={disconnect}
                className="inline-flex items-center px-3 py-1 border border-gray-300 text-xs font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              >
                Disconnect
              </button>
            </div>
          </div>
          
          {/* Connection Status Details */}
          <div className="grid grid-cols-3 gap-4 text-xs">
            {Object.entries(status).map(([service, connected]) => (
              <div key={service} className="flex items-center space-x-2">
                <div className={`h-1.5 w-1.5 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} />
                <span className={connected ? 'text-green-700' : 'text-red-700'}>
                  {service.charAt(0).toUpperCase() + service.slice(1)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tab Navigation */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8 px-4" aria-label="Tabs">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`
                py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap flex items-center space-x-2
                ${activeTab === tab.id
                  ? 'border-indigo-500 text-indigo-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }
              `}
            >
              <span>{tab.icon}</span>
              <span>{tab.label}</span>
              {tab.badge && (
                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                  {tab.badge}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="p-0">
        <ActiveComponent 
          className="border-0 shadow-none rounded-none"
          autoConnect={false} // Dashboard handles connection
        />
      </div>
    </div>
  )
}

export default RealtimeDashboard