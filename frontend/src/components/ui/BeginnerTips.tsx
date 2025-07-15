import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { XMarkIcon, LightBulbIcon, ChevronRightIcon } from '@heroicons/react/24/outline'
import { useLocation } from 'react-router-dom'
import { usePreferencesStore } from '../../store/preferences'

interface Tip {
  id: string
  title: string
  content: string
  action?: {
    label: string
    onClick: () => void
  }
}

const tipsByRoute: Record<string, Tip[]> = {
  '/dashboard': [
    {
      id: 'dashboard-welcome',
      title: 'Welcome to DFS Optimizer!',
      content: 'Start by selecting a sport and contest type. We\'ll help you build winning lineups step by step.',
    },
    {
      id: 'dashboard-sport',
      title: 'Choose Your Sport',
      content: 'Pick from NFL, NBA, MLB, or NHL. Each sport has different strategies and player positions.',
    },
  ],
  '/optimizer': [
    {
      id: 'optimizer-player-pool',
      title: 'Select Your Players',
      content: 'Browse available players and add them to your lineup. Look for high projected points and good value.',
    },
    {
      id: 'optimizer-salary-cap',
      title: 'Watch Your Salary',
      content: 'Stay within the salary cap while maximizing projected points. Look for undervalued players!',
    },
    {
      id: 'optimizer-positions',
      title: 'Fill All Positions',
      content: 'Each lineup needs specific positions filled. The optimizer will help you find the best combinations.',
    },
  ],
  '/lineups': [
    {
      id: 'lineups-review',
      title: 'Review Your Lineups',
      content: 'Check your generated lineups before exporting. You can make manual adjustments if needed.',
    },
    {
      id: 'lineups-export',
      title: 'Export to DFS Sites',
      content: 'Download your lineups in CSV format to upload directly to DraftKings or FanDuel.',
    },
  ],
}

export const BeginnerTips: React.FC = () => {
  const location = useLocation()
  const { beginnerMode, tutorialProgress, completeTutorialStep } = usePreferencesStore()
  const [currentTipIndex, setCurrentTipIndex] = useState(0)
  const [isVisible, setIsVisible] = useState(true)
  // const [_hasInteracted, _setHasInteracted] = useState(false)
  const _setHasInteracted = (_: boolean) => {}

  const tips = tipsByRoute[location.pathname] || []
  const currentTip = tips[currentTipIndex]

  useEffect(() => {
    // Reset on route change
    setCurrentTipIndex(0)
    setIsVisible(true)
    _setHasInteracted(false)
  }, [location.pathname])

  useEffect(() => {
    // Auto-hide tips if already completed
    if (currentTip && tutorialProgress.completed.includes(currentTip.id)) {
      setIsVisible(false)
    }
  }, [currentTip, tutorialProgress.completed])

  const handleNext = () => {
    if (currentTip) {
      completeTutorialStep(currentTip.id)
    }
    
    if (currentTipIndex < tips.length - 1) {
      setCurrentTipIndex(currentTipIndex + 1)
    } else {
      setIsVisible(false)
    }
    _setHasInteracted(true)
  }

  const handleDismiss = () => {
    if (currentTip) {
      completeTutorialStep(currentTip.id)
    }
    setIsVisible(false)
    _setHasInteracted(true)
  }

  if (!beginnerMode || !currentTip || !isVisible) {
    return null
  }

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: 20 }}
        transition={{ duration: 0.3 }}
        className="fixed bottom-24 left-1/2 transform -translate-x-1/2 z-40 max-w-md w-full px-4"
      >
        <div className="bg-gradient-to-r from-blue-500 to-blue-600 rounded-lg shadow-xl p-6 text-white">
          <div className="flex items-start justify-between mb-3">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-white/20 rounded-lg">
                <LightBulbIcon className="h-6 w-6" />
              </div>
              <h3 className="text-lg font-semibold">{currentTip.title}</h3>
            </div>
            <button
              onClick={handleDismiss}
              className="p-1 hover:bg-white/20 rounded-lg transition-colors"
            >
              <XMarkIcon className="h-5 w-5" />
            </button>
          </div>
          
          <p className="text-white/90 mb-4">{currentTip.content}</p>
          
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              {tips.map((_, index) => (
                <div
                  key={index}
                  className={`h-2 rounded-full transition-all duration-300 ${
                    index === currentTipIndex
                      ? 'w-8 bg-white'
                      : index < currentTipIndex
                      ? 'w-2 bg-white/60'
                      : 'w-2 bg-white/30'
                  }`}
                />
              ))}
            </div>
            
            <div className="flex items-center gap-2">
              {currentTip.action && (
                <button
                  onClick={currentTip.action.onClick}
                  className="px-4 py-2 bg-white text-blue-600 rounded-lg font-medium hover:bg-white/90 transition-colors"
                >
                  {currentTip.action.label}
                </button>
              )}
              
              {currentTipIndex < tips.length - 1 ? (
                <button
                  onClick={handleNext}
                  className="flex items-center gap-1 px-4 py-2 bg-white/20 rounded-lg font-medium hover:bg-white/30 transition-colors"
                >
                  Next
                  <ChevronRightIcon className="h-4 w-4" />
                </button>
              ) : (
                <button
                  onClick={handleDismiss}
                  className="px-4 py-2 bg-white/20 rounded-lg font-medium hover:bg-white/30 transition-colors"
                >
                  Got it!
                </button>
              )}
            </div>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  )
}