import { useState, useEffect } from 'react'
import { Dialog, DialogTitle, DialogBody, DialogActions } from '@/catalyst-ui-kit/typescript/dialog'
import { Button } from '@/catalyst-ui-kit/typescript/button'
import { cn } from '@/lib/catalyst'
import { usePreferencesStore } from '@/store/preferences'
import { usePreferences } from '@/hooks/usePreferences'

interface PreferencesModalProps {
  isOpen: boolean
  onClose: () => void
}

type TabType = 'display' | 'ai' | 'sports' | 'advanced'

const SPORTS = [
  { id: 'nfl', name: 'NFL', emoji: '🏈' },
  { id: 'nba', name: 'NBA', emoji: '🏀' },
  { id: 'mlb', name: 'MLB', emoji: '⚾' },
  { id: 'nhl', name: 'NHL', emoji: '🏒' },
  { id: 'pga', name: 'PGA', emoji: '⛳' },
  { id: 'nascar', name: 'NASCAR', emoji: '🏁' },
  { id: 'mma', name: 'MMA', emoji: '🥊' },
  { id: 'soccer', name: 'Soccer', emoji: '⚽' },
]

export default function PreferencesModal({ isOpen, onClose }: PreferencesModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>('display')
  const [unsavedChanges, setUnsavedChanges] = useState(false)
  
  const store = usePreferencesStore()
  const { preferences, updatePreferences, resetPreferences, isLoading, error } = usePreferences()
  
  // Local state for form values
  const [formValues, setFormValues] = useState({
    beginnerMode: store.beginnerMode,
    tooltipSettings: { ...store.tooltipSettings },
    aiSettings: { ...store.aiSettings },
    preferredSports: store.preferredSports || [],
  })

  // Sync form values with store/API preferences
  useEffect(() => {
    if (preferences) {
      setFormValues({
        beginnerMode: preferences.beginner_mode,
        tooltipSettings: {
          enabled: preferences.show_tooltips,
          delay: preferences.tooltip_delay,
          showOnMobile: store.tooltipSettings.showOnMobile, // Keep local-only setting
        },
        aiSettings: {
          enabled: preferences.ai_suggestions_enabled,
          autoSuggest: store.aiSettings.autoSuggest, // Keep local-only setting
          confidenceThreshold: store.aiSettings.confidenceThreshold, // Keep local-only setting
        },
        preferredSports: preferences.preferred_sports || [],
      })
    }
  }, [preferences, store.tooltipSettings.showOnMobile, store.aiSettings.autoSuggest, store.aiSettings.confidenceThreshold])

  const handleSave = async () => {
    // Update local store
    store.setBeginnerMode(formValues.beginnerMode)
    store.updateTooltipSettings(formValues.tooltipSettings)
    store.updateAISettings(formValues.aiSettings)
    
    // Update API
    await updatePreferences({
      beginner_mode: formValues.beginnerMode,
      show_tooltips: formValues.tooltipSettings.enabled,
      tooltip_delay: formValues.tooltipSettings.delay,
      preferred_sports: formValues.preferredSports,
      ai_suggestions_enabled: formValues.aiSettings.enabled,
    })
    
    setUnsavedChanges(false)
    onClose()
  }

  const handleReset = async () => {
    if (confirm('Are you sure you want to reset all preferences to defaults?')) {
      await resetPreferences()
      store.resetTutorial()
      setUnsavedChanges(false)
    }
  }

  const handleCancel = () => {
    if (unsavedChanges && !confirm('You have unsaved changes. Are you sure you want to close?')) {
      return
    }
    onClose()
  }

  const updateFormValue = (path: string, value: any) => {
    setFormValues(prev => {
      const newValues = { ...prev }
      const keys = path.split('.')
      let current: any = newValues
      
      for (let i = 0; i < keys.length - 1; i++) {
        current = current[keys[i]]
      }
      
      current[keys[keys.length - 1]] = value
      return newValues
    })
    setUnsavedChanges(true)
  }

  const toggleSport = (sportId: string) => {
    setFormValues(prev => ({
      ...prev,
      preferredSports: prev.preferredSports.includes(sportId)
        ? prev.preferredSports.filter(s => s !== sportId)
        : [...prev.preferredSports, sportId]
    }))
    setUnsavedChanges(true)
  }

  return (
    <Dialog 
      open={isOpen} 
      onClose={handleCancel}
      size="4xl"
      className="max-h-[85vh] overflow-hidden"
    >
      {/* Header */}
      <div className="border-b border-gray-200/20 pb-6 dark:border-gray-700/20">
        <div className="flex items-center justify-between">
          <div>
            <DialogTitle className="text-2xl font-bold flex items-center gap-2">
              <span className="text-3xl">⚙️</span>
              Preferences & Settings
            </DialogTitle>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              Customize your DFS Optimizer experience
            </p>
          </div>
          <Button
            plain
            onClick={handleCancel}
            aria-label="Close preferences"
          >
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </Button>
        </div>
          
        {/* Tabs */}
        <div className="mt-6 flex space-x-1">
          {([
            { id: 'display', label: 'Display', icon: '🎨' },
            { id: 'ai', label: 'AI Settings', icon: '🤖' },
            { id: 'sports', label: 'Sports', icon: '🏆' },
            { id: 'advanced', label: 'Advanced', icon: '🔧' },
          ] as const).map((tab) => 
            activeTab === tab.id ? (
              <Button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                color="blue"
                className="text-sm"
              >
                <span className="mr-2">{tab.icon}</span>
                {tab.label}
              </Button>
            ) : (
              <Button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                plain
                className="text-sm"
              >
                <span className="mr-2">{tab.icon}</span>
                {tab.label}
              </Button>
            )
          )}
        </div>
      </div>
        
      {/* Content */}
      <DialogBody className="overflow-y-auto max-h-[calc(85vh-240px)]">
          {error && (
            <div className="mb-6 glass bg-red-500/10 border-red-500/20 rounded-lg p-4">
              <p className="text-sm text-red-600 dark:text-red-400">
                Error: {error}
              </p>
            </div>
          )}

          {activeTab === 'display' && (
            <div className="space-y-6">
              {/* Beginner Mode */}
              <div className="glass rounded-lg p-6 hover:shadow-lg transition-all duration-200">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <h3 className="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                      <span className="text-xl">🎓</span>
                      Beginner Mode
                    </h3>
                    <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                      Show additional help and simplified options for new DFS players
                    </p>
                  </div>
                  <button
                    onClick={() => updateFormValue('beginnerMode', !formValues.beginnerMode)}
                    className={cn(
                      'relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200',
                      formValues.beginnerMode ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
                    )}
                  >
                    <span
                      className={cn(
                        'inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200',
                        formValues.beginnerMode ? 'translate-x-6' : 'translate-x-1'
                      )}
                    />
                  </button>
                </div>
              </div>

              {/* Tooltips */}
              <div className="glass rounded-lg p-6 hover:shadow-lg transition-all duration-200">
                <div className="space-y-4">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <h3 className="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                        <span className="text-xl">💡</span>
                        Show Tooltips
                      </h3>
                      <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                        Display helpful tooltips when hovering over UI elements
                      </p>
                    </div>
                    <button
                      onClick={() => updateFormValue('tooltipSettings.enabled', !formValues.tooltipSettings.enabled)}
                      className={cn(
                        'relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200',
                        formValues.tooltipSettings.enabled ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
                      )}
                    >
                      <span
                        className={cn(
                          'inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200',
                          formValues.tooltipSettings.enabled ? 'translate-x-6' : 'translate-x-1'
                        )}
                      />
                    </button>
                  </div>

                  {formValues.tooltipSettings.enabled && (
                    <>
                      <div className="ml-7">
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                          Tooltip Delay (ms)
                        </label>
                        <div className="flex items-center gap-4">
                          <input
                            type="range"
                            min="0"
                            max="2000"
                            step="100"
                            value={formValues.tooltipSettings.delay}
                            onChange={(e) => updateFormValue('tooltipSettings.delay', Number(e.target.value))}
                            className="flex-1"
                          />
                          <span className="w-16 text-sm text-gray-600 dark:text-gray-400">
                            {formValues.tooltipSettings.delay}ms
                          </span>
                        </div>
                      </div>

                      <div className="ml-7 flex items-center justify-between">
                        <div>
                          <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
                            Show on Mobile
                          </p>
                          <p className="text-xs text-gray-500 dark:text-gray-500">
                            Display tooltips on touch devices
                          </p>
                        </div>
                        <button
                          onClick={() => updateFormValue('tooltipSettings.showOnMobile', !formValues.tooltipSettings.showOnMobile)}
                          className={cn(
                            'relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200',
                            formValues.tooltipSettings.showOnMobile ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
                          )}
                        >
                          <span
                            className={cn(
                              'inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200',
                              formValues.tooltipSettings.showOnMobile ? 'translate-x-6' : 'translate-x-1'
                            )}
                          />
                        </button>
                      </div>
                    </>
                  )}
                </div>
              </div>
            </div>
          )}

          {activeTab === 'ai' && (
            <div className="space-y-6">
              {/* AI Suggestions */}
              <div className="glass rounded-lg p-6 hover:shadow-lg transition-all duration-200">
                <div className="space-y-4">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <h3 className="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                        <span className="text-xl">🤖</span>
                        Enable AI Suggestions
                      </h3>
                      <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                        Get intelligent lineup recommendations and optimization tips
                      </p>
                    </div>
                    <button
                      onClick={() => updateFormValue('aiSettings.enabled', !formValues.aiSettings.enabled)}
                      className={cn(
                        'relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200',
                        formValues.aiSettings.enabled ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
                      )}
                    >
                      <span
                        className={cn(
                          'inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200',
                          formValues.aiSettings.enabled ? 'translate-x-6' : 'translate-x-1'
                        )}
                      />
                    </button>
                  </div>

                  {formValues.aiSettings.enabled && (
                    <>
                      <div className="ml-7 flex items-center justify-between">
                        <div>
                          <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
                            Auto-Suggest
                          </p>
                          <p className="text-xs text-gray-500 dark:text-gray-500">
                            Automatically show suggestions while building lineups
                          </p>
                        </div>
                        <button
                          onClick={() => updateFormValue('aiSettings.autoSuggest', !formValues.aiSettings.autoSuggest)}
                          className={cn(
                            'relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200',
                            formValues.aiSettings.autoSuggest ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
                          )}
                        >
                          <span
                            className={cn(
                              'inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200',
                              formValues.aiSettings.autoSuggest ? 'translate-x-6' : 'translate-x-1'
                            )}
                          />
                        </button>
                      </div>

                      <div className="ml-7">
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                          Confidence Threshold
                        </label>
                        <div className="flex items-center gap-4">
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={formValues.aiSettings.confidenceThreshold}
                            onChange={(e) => updateFormValue('aiSettings.confidenceThreshold', Number(e.target.value))}
                            className="flex-1"
                          />
                          <span className="w-16 text-sm text-gray-600 dark:text-gray-400">
                            {(formValues.aiSettings.confidenceThreshold * 100).toFixed(0)}%
                          </span>
                        </div>
                        <p className="mt-1 text-xs text-gray-500 dark:text-gray-500">
                          Only show suggestions above this confidence level
                        </p>
                      </div>
                    </>
                  )}
                </div>
              </div>

              {/* AI Features Info */}
              <div className="glass rounded-lg p-6 bg-blue-500/5 border-blue-500/20">
                <h4 className="font-semibold text-blue-900 dark:text-blue-300 mb-3 flex items-center gap-2">
                  <span>💡</span>
                  AI Features Include:
                </h4>
                <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500">✓</span>
                    <span>Smart player recommendations based on matchups and projections</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500">✓</span>
                    <span>Optimal stacking suggestions for GPP tournaments</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500">✓</span>
                    <span>Risk assessment and lineup diversity analysis</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500">✓</span>
                    <span>Real-time optimization feedback and improvements</span>
                  </li>
                </ul>
              </div>
            </div>
          )}

          {activeTab === 'sports' && (
            <div className="space-y-6">
              <div className="glass rounded-lg p-6">
                <h3 className="font-semibold text-gray-900 dark:text-white mb-4">
                  Preferred Sports
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
                  Select your favorite sports to see them first in contest selection
                </p>
                
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  {SPORTS.map((sport) => 
                    formValues.preferredSports.includes(sport.id) ? (
                      <Button
                        key={sport.id}
                        onClick={() => toggleSport(sport.id)}
                        color="blue"
                        className="p-4 h-auto"
                      >
                        <div className="text-center">
                          <span className="text-3xl">{sport.emoji}</span>
                          <p className="mt-2 font-medium">
                            {sport.name}
                          </p>
                          <div className="mt-2">
                            <span className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-white/20 text-white">
                              ✓
                            </span>
                          </div>
                        </div>
                      </Button>
                    ) : (
                      <Button
                        key={sport.id}
                        onClick={() => toggleSport(sport.id)}
                        outline
                        className="p-4 h-auto"
                      >
                        <div className="text-center">
                          <span className="text-3xl">{sport.emoji}</span>
                          <p className="mt-2 font-medium">
                            {sport.name}
                          </p>
                        </div>
                      </Button>
                    )
                  )}
                </div>
              </div>
            </div>
          )}

          {activeTab === 'advanced' && (
            <div className="space-y-6">
              {/* Tutorial Progress */}
              <div className="glass rounded-lg p-6">
                <h3 className="font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                  <span className="text-xl">📚</span>
                  Tutorial Progress
                </h3>
                <div className="space-y-3">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-gray-600 dark:text-gray-400">
                      Completed Steps
                    </span>
                    <span className="font-medium text-gray-900 dark:text-white">
                      {store.tutorialProgress.completed.length}
                    </span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-gray-600 dark:text-gray-400">
                      Skipped Steps
                    </span>
                    <span className="font-medium text-gray-900 dark:text-white">
                      {store.tutorialProgress.skipped.length}
                    </span>
                  </div>
                  <Button
                    onClick={() => {
                      if (confirm('Are you sure you want to reset tutorial progress?')) {
                        store.resetTutorial()
                      }
                    }}
                    plain
                    className="mt-4 w-full"
                  >
                    Reset Tutorial Progress
                  </Button>
                </div>
              </div>

              {/* Data & Privacy */}
              <div className="glass rounded-lg p-6">
                <h3 className="font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                  <span className="text-xl">🔒</span>
                  Data & Privacy
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                  Your preferences are stored locally and synced with your account for a consistent experience across devices.
                </p>
                <Button
                  color="red"
                  onClick={() => {
                    if (confirm('This will clear all local data. Are you sure?')) {
                      localStorage.clear()
                      window.location.reload()
                    }
                  }}
                >
                  Clear Local Data
                </Button>
              </div>
            </div>
          )}
      </DialogBody>
      
      {/* Footer */}
      <DialogActions className="border-t border-gray-200/20 pt-6 dark:border-gray-700/20">
        <Button
          onClick={handleReset}
          disabled={isLoading}
          plain
          className="mr-auto"
        >
          Reset to Defaults
        </Button>
        
        <Button
          onClick={handleCancel}
          disabled={isLoading}
          plain
        >
          Cancel
        </Button>
        
        <Button
          onClick={handleSave}
          disabled={isLoading || !unsavedChanges}
          color={unsavedChanges ? 'blue' : 'zinc'}
        >
          {isLoading ? 'Saving...' : 'Save Changes'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}