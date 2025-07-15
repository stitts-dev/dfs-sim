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
import { useAuthStore } from '@/store/auth'
import { usePhoneAuth } from '@/hooks/usePhoneAuth'
import { toast } from 'react-hot-toast'

interface LayoutProps {
  children: ReactNode
}

export default function Layout({ children }: LayoutProps) {
  const location = useLocation()
  const navigate = useNavigate()
  const { beginnerMode } = usePreferencesStore()
  const { user } = useAuthStore()
  const { signOut, isLoggingOut } = usePhoneAuth()
  const [showGuide, setShowGuide] = useState(false)
  const [showPreferences, setShowPreferences] = useState(false)
  const [showUserMenu, setShowUserMenu] = useState(false)
  
  // Handle logout
  const handleLogout = async () => {
    try {
      await signOut()
      toast.success('Logged out successfully')
      navigate('/auth/login')
    } catch (error) {
      toast.error('Failed to log out')
    }
    setShowUserMenu(false)
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
    const handleClickOutside = () => {
      if (showUserMenu) {
        setShowUserMenu(false)
      }
    }
    
    document.addEventListener('click', handleClickOutside)
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
        
        {/* User Profile & Logout */}
        <div className="relative">
          <NavbarItem
            onClick={() => setShowUserMenu(!showUserMenu)}
            title={`Logged in as ${user?.phone_number || 'User'}`}
            className="flex items-center space-x-2"
          >
            <div className="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center text-white text-sm font-medium">
              {user?.first_name ? user.first_name[0].toUpperCase() : '👤'}
            </div>
            <span className="hidden md:inline text-sm">
              {user?.first_name || user?.phone_number || 'User'}
            </span>
            <span className="text-xs">▼</span>
          </NavbarItem>
          
          {/* User Menu Dropdown */}
          {showUserMenu && (
            <div className="absolute right-0 mt-2 w-48 bg-white rounded-md shadow-lg border border-gray-200 z-50">
              <div className="py-1">
                <div className="px-4 py-2 text-sm text-gray-700 border-b border-gray-100">
                  <div className="font-medium">{user?.first_name || 'User'}</div>
                  <div className="text-xs text-gray-500">{user?.phone_number}</div>
                  <div className="text-xs text-blue-600 capitalize">{user?.subscription_tier} Plan</div>
                </div>
                <button
                  onClick={() => {
                    setShowPreferences(true)
                    setShowUserMenu(false)
                  }}
                  className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                >
                  Account Settings
                </button>
                <button
                  onClick={handleLogout}
                  disabled={isLoggingOut}
                  className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 disabled:opacity-50"
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