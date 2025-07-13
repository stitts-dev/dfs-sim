import React, { ReactNode, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { StackedLayout } from '@/catalyst-ui-kit/typescript/stacked-layout'
import { Navbar, NavbarSection, NavbarItem, NavbarSpacer } from '@/catalyst-ui-kit/typescript/navbar'
import { cn } from '@/lib/catalyst'
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