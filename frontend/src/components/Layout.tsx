import React, { ReactNode, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { cn } from '@/lib/utils'
import QuickReferenceGuide from '@/components/ui/QuickReferenceGuide'
import HelpIcon from '@/components/ui/HelpIcon'
import PreferencesModal from '@/components/settings/PreferencesModal'
import { BeginnerModeToggle } from '@/components/ui/BeginnerModeToggle'
import { BeginnerTips } from '@/components/ui/BeginnerTips'
import { usePreferencesStore } from '@/store/preferences'

interface LayoutProps {
  children: ReactNode
}

export default function Layout({ children }: LayoutProps) {
  const location = useLocation()
  const { beginnerMode } = usePreferencesStore()
  const [showGuide, setShowGuide] = useState(false)
  const [showPreferences, setShowPreferences] = useState(false)
  
  // Add keyboard shortcut for F1
  React.useEffect(() => {
    const handleKeyPress = (e: KeyboardEvent) => {
      if (e.key === 'F1') {
        e.preventDefault()
        setShowGuide(true)
      }
    }
    
    window.addEventListener('keydown', handleKeyPress)
    return () => window.removeEventListener('keydown', handleKeyPress)
  }, [])

  const navigation = [
    { name: 'Dashboard', href: '/dashboard', icon: '📊' },
    { name: 'Optimizer', href: '/optimizer', icon: '🎯' },
    { name: 'Lineups', href: '/lineups', icon: '📋' },
  ]

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 shadow">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center justify-between">
            <div className="flex items-center">
              <h1 className="text-xl font-bold text-gray-900 dark:text-white">
                DFS Lineup Optimizer
              </h1>
            </div>
            <nav className="flex space-x-4">
              {navigation.map((item) => (
                <Link
                  key={item.name}
                  to={item.href}
                  className={cn(
                    'flex items-center rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    location.pathname === item.href
                      ? 'bg-gray-900 text-white dark:bg-gray-700'
                      : 'text-gray-700 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-700 dark:hover:text-white',
                    beginnerMode && location.pathname === item.href && 'ring-2 ring-blue-400 ring-offset-2'
                  )}
                >
                  <span className="mr-2">{item.icon}</span>
                  {item.name}
                </Link>
              ))}
              
              {/* Settings Button */}
              <button
                onClick={() => setShowPreferences(true)}
                className="ml-4 flex items-center rounded-md px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-700 dark:hover:text-white transition-colors"
                title="Preferences & Settings"
              >
                <span className="text-lg mr-2">⚙️</span>
                <span className="hidden sm:inline">Settings</span>
              </button>
              
              {/* Help Button */}
              <button
                onClick={() => setShowGuide(true)}
                className="flex items-center rounded-md px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-700 dark:hover:text-white transition-colors"
                title="Quick Reference Guide (Press F1)"
              >
                <HelpIcon size="md" className="mr-2" />
                <span className="hidden sm:inline">Help</span>
              </button>
            </nav>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        {children}
      </main>
      
      {/* Quick Reference Guide Modal */}
      <QuickReferenceGuide 
        isOpen={showGuide} 
        onClose={() => setShowGuide(false)} 
      />
      
      {/* Preferences Modal */}
      <PreferencesModal
        isOpen={showPreferences}
        onClose={() => setShowPreferences(false)}
      />
      
      {/* Beginner Mode Toggle */}
      <BeginnerModeToggle />
      
      {/* Beginner Tips */}
      <BeginnerTips />
    </div>
  )
}