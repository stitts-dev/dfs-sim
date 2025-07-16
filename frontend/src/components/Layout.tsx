import React, { ReactNode, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { StackedLayout } from '@/catalyst/StackedLayout'
import { Navbar, NavbarSection, NavbarItem, NavbarSpacer } from '@/catalyst/Navbar'
import { cn } from '@/lib/catalyst'
import QuickReferenceGuide from '@/components/ui/QuickReferenceGuide'
import HelpIcon from '@/components/ui/HelpIcon'
import PreferencesModal from '@/components/settings/PreferencesModal'
import { BeginnerModeToggle } from '@/components/ui/BeginnerModeToggle'
import { BeginnerTips } from '@/components/ui/BeginnerTips'
import { usePreferencesStore } from '@/store/preferences'
import { useUnifiedAuthStore } from '@/store/unifiedAuth'
import { toast } from 'react-hot-toast'

interface LayoutProps {
  children: ReactNode
}

export default function Layout({ children }: LayoutProps) {
  const location = useLocation()
  const navigate = useNavigate()
  const { beginnerMode } = usePreferencesStore()
  const { user, logout } = useUnifiedAuthStore()
  const [isLoggingOut, setIsLoggingOut] = useState(false)
  const [showGuide, setShowGuide] = useState(false)
  const [showPreferences, setShowPreferences] = useState(false)
  const [showUserMenu, setShowUserMenu] = useState(false)
  
  // Handle logout
  const handleLogout = async () => {
    setIsLoggingOut(true)
    try {
      await logout()
      toast.success('Logged out successfully')
      navigate('/auth/login')
    } catch (error) {
      toast.error('Failed to log out')
    } finally {
      setIsLoggingOut(false)
      setShowUserMenu(false)
    }
  }

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

  // Close user menu when clicking outside
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      // Don't close if clicking inside the user menu or its trigger
      const target = event.target as HTMLElement
      if (target.closest('[data-user-menu]') || target.closest('[data-user-menu-trigger]')) {
        return
      }
      
      if (showUserMenu) {
        setShowUserMenu(false)
      }
    }
    
    if (showUserMenu) {
      document.addEventListener('click', handleClickOutside)
    }
    
    return () => document.removeEventListener('click', handleClickOutside)
  }, [showUserMenu])

  const navigation = [
    { name: 'Dashboard', href: '/dashboard', icon: '📊' },
    { name: 'Optimizer', href: '/optimizer', icon: '🎯' },
    { name: 'Lineups', href: '/lineups', icon: '📋' },
  ]

  const navbar = (
    <Navbar>
      <NavbarSection>
        <h1 className="text-xl font-bold">
          DFS Lineup Optimizer
        </h1>
      </NavbarSection>
      
      <NavbarSpacer />
      
      <NavbarSection>
        {navigation.map((item) => (
          <NavbarItem
            key={item.name}
            href={item.href}
            current={location.pathname === item.href}
            className={cn(
              beginnerMode && location.pathname === item.href && 'ring-2 ring-blue-400 ring-offset-2'
            )}
          >
            <span className="mr-2">{item.icon}</span>
            {item.name}
          </NavbarItem>
        ))}
        
        {/* Settings Button */}
        <NavbarItem
          onClick={() => setShowPreferences(true)}
          title="Preferences & Settings"
        >
          <span className="text-lg mr-2">⚙️</span>
          <span className="hidden sm:inline">Settings</span>
        </NavbarItem>
        
        {/* Help Button */}
        <NavbarItem
          onClick={() => setShowGuide(true)}
          title="Quick Reference Guide (Press F1)"
        >
          <HelpIcon size="md" className="mr-2" />
          <span className="hidden sm:inline">Help</span>
        </NavbarItem>
        
        {/* Temporary Logout Button (Backup) */}
        <NavbarItem
          onClick={handleLogout}
          title="Sign out"
          className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
        >
          <span className="text-lg mr-2">🚪</span>
          <span className="hidden sm:inline">Logout</span>
        </NavbarItem>
        
        {/* User Profile & Logout */}
        <div className="relative" data-user-menu-trigger>
          <button
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              setShowUserMenu(!showUserMenu)
            }}
            title={`Logged in as ${user?.phone_number || user?.email || 'User'}`}
            className="flex items-center space-x-2 px-3 py-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          >
            <div className="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center text-white text-sm font-medium">
              {user?.first_name ? user.first_name[0].toUpperCase() : '👤'}
            </div>
            <span className="hidden md:inline text-sm">
              {user?.first_name || user?.phone_number || user?.email || 'User'}
            </span>
            <span className="text-xs">▼</span>
          </button>
          
          {/* User Menu Dropdown */}
          {showUserMenu && (
            <div 
              data-user-menu
              className="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-md shadow-lg border border-gray-200 dark:border-gray-700 z-[9999]"
            >
              <div className="py-1">
                <div className="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 border-b border-gray-100 dark:border-gray-700">
                  <div className="font-medium">{user?.first_name || 'User'}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">{user?.phone_number || user?.email}</div>
                  {user?.subscription_tier && (
                    <div className="text-xs text-blue-600 dark:text-blue-400 capitalize">{user.subscription_tier} Plan</div>
                  )}
                </div>
                <button
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    setShowPreferences(true)
                    setShowUserMenu(false)
                  }}
                  className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                >
                  Account Settings
                </button>
                <button
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    handleLogout()
                  }}
                  disabled={isLoggingOut}
                  className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 transition-colors"
                >
                  {isLoggingOut ? 'Signing out...' : 'Sign out'}
                </button>
              </div>
            </div>
          )}
        </div>
      </NavbarSection>
    </Navbar>
  )

  // Empty sidebar for StackedLayout requirement
  const sidebar = null

  return (
    <StackedLayout
      navbar={navbar}
      sidebar={sidebar}
    >
      {children}
      
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
    </StackedLayout>
  )
}