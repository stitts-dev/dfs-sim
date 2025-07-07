import React, { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { UserIcon, AcademicCapIcon } from '@heroicons/react/24/outline'
import { usePreferencesStore } from '../../store/preferences'

export const BeginnerModeToggle: React.FC = () => {
  const { beginnerMode, setBeginnerMode } = usePreferencesStore()
  const [showTooltip, setShowTooltip] = useState(false)

  const handleToggle = () => {
    setBeginnerMode(!beginnerMode)
  }

  return (
    <div className="fixed bottom-6 left-6 z-40">
      <div
        className="relative"
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
      >
        <motion.button
          onClick={handleToggle}
          className={`
            flex h-14 w-14 items-center justify-center rounded-full shadow-lg
            transform hover:scale-110 transition-all duration-200 animate-float
            ${beginnerMode 
              ? 'bg-gradient-to-r from-blue-600 to-teal-600 text-white hover:shadow-xl' 
              : 'bg-gradient-to-r from-gray-600 to-gray-700 text-gray-100 hover:shadow-xl'
            }
          `}
          whileHover={{ scale: 1.1 }}
          whileTap={{ scale: 0.95 }}
        >
          <motion.div
            initial={false}
            animate={{ rotate: beginnerMode ? 0 : 180 }}
            transition={{ duration: 0.3 }}
          >
            {beginnerMode ? (
              <AcademicCapIcon className="h-6 w-6" />
            ) : (
              <UserIcon className="h-6 w-6" />
            )}
          </motion.div>
        </motion.button>
        
        {/* Pulse indicator for beginner mode */}
        {beginnerMode && (
          <div className="absolute -top-1 -right-1">
            <span className="flex h-3 w-3">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-teal-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-teal-500"></span>
            </span>
          </div>
        )}

        <AnimatePresence>
          {showTooltip && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 10 }}
              className="absolute bottom-full left-0 mb-3 w-72 p-4 bg-gray-800 text-white rounded-lg shadow-xl"
            >
              <div className="absolute bottom-0 left-8 transform translate-y-1/2 rotate-45 w-3 h-3 bg-gray-800"></div>
              <h4 className="font-semibold mb-2">
                {beginnerMode ? 'Beginner Mode Active' : 'Expert Mode Active'}
              </h4>
              <p className="text-sm text-gray-300">
                {beginnerMode 
                  ? 'Simplified interface with helpful tips and essential features only. Perfect for learning DFS basics.'
                  : 'Full interface with all advanced features, metrics, and customization options available.'
                }
              </p>
              <p className="text-xs text-gray-400 mt-2">
                Click to switch to {beginnerMode ? 'Expert' : 'Beginner'} Mode
              </p>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  )
}