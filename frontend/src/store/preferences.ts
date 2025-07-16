import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { useUnifiedAuthStore } from './unifiedAuth'
import { apiFetch, apiPut } from '@/services/apiClient'

interface PreferencesState {
  beginnerMode: boolean
  tooltipSettings: {
    enabled: boolean
    delay: number
    showOnMobile: boolean
  }
  aiSettings: {
    enabled: boolean
    autoSuggest: boolean
    confidenceThreshold: number
  }
  tutorialProgress: {
    completed: string[]
    skipped: string[]
    currentStep: string | null
  }
  preferredSports: string[]
  isLoading: boolean
  lastSyncedAt: string | null
  
  // Actions
  setBeginnerMode: (enabled: boolean) => void
  updateTooltipSettings: (settings: Partial<PreferencesState['tooltipSettings']>) => void
  updateAISettings: (settings: Partial<PreferencesState['aiSettings']>) => void
  completeTutorialStep: (stepId: string) => void
  skipTutorialStep: (stepId: string) => void
  resetTutorial: () => void
  setPreferredSports: (sports: string[]) => void
  loadUserPreferences: () => Promise<void>
  saveUserPreferences: () => Promise<void>
  syncWithBackend: () => Promise<void>
}

export const usePreferencesStore = create<PreferencesState>()(
  persist(
    (set, get) => ({
      beginnerMode: true, // Default to beginner mode for new users
      tooltipSettings: {
        enabled: true,
        delay: 300,
        showOnMobile: true,
      },
      aiSettings: {
        enabled: true,
        autoSuggest: true,
        confidenceThreshold: 0.7,
      },
      tutorialProgress: {
        completed: [],
        skipped: [],
        currentStep: null,
      },
      preferredSports: [],
      isLoading: false,
      lastSyncedAt: null,
      
      setBeginnerMode: (enabled) => set({ beginnerMode: enabled }),
      
      updateTooltipSettings: (settings) =>
        set((state) => ({
          tooltipSettings: { ...state.tooltipSettings, ...settings },
        })),
      
      updateAISettings: (settings) =>
        set((state) => ({
          aiSettings: { ...state.aiSettings, ...settings },
        })),
      
      completeTutorialStep: (stepId) =>
        set((state) => ({
          tutorialProgress: {
            ...state.tutorialProgress,
            completed: [...state.tutorialProgress.completed, stepId],
          },
        })),
      
      skipTutorialStep: (stepId) =>
        set((state) => ({
          tutorialProgress: {
            ...state.tutorialProgress,
            skipped: [...state.tutorialProgress.skipped, stepId],
          },
        })),
      
      resetTutorial: () =>
        set(() => ({
          tutorialProgress: {
            completed: [],
            skipped: [],
            currentStep: null,
          },
        })),
      
      setPreferredSports: (sports) => set({ preferredSports: sports }),
      
      // Backend integration methods
      loadUserPreferences: async () => {
        const { phoneToken, supabaseSession, authMethod } = useUnifiedAuthStore.getState()
        const token = authMethod === 'phone' ? phoneToken : supabaseSession?.access_token
        if (!token) return
        
        set({ isLoading: true })
        
        try {
          const userPrefs = await apiFetch('/users/preferences')
          
          // Map backend preferences to frontend state
          const typedUserPrefs = userPrefs as any
          set({
            beginnerMode: typedUserPrefs.beginner_mode ?? true,
            tooltipSettings: {
              enabled: typedUserPrefs.tooltips_enabled ?? true,
              delay: 300,
              showOnMobile: true,
            },
            preferredSports: typedUserPrefs.sport_preferences || [],
            lastSyncedAt: new Date().toISOString(),
            isLoading: false
          })
        } catch (error) {
          console.warn('Failed to load user preferences:', error)
          set({ isLoading: false })
        }
      },
      
      saveUserPreferences: async () => {
        const { phoneToken, supabaseSession, authMethod } = useUnifiedAuthStore.getState()
        const token = authMethod === 'phone' ? phoneToken : supabaseSession?.access_token
        if (!token) return
        
        const state = get()
        
        try {
          await apiPut('/users/preferences', {
            beginner_mode: state.beginnerMode,
            tooltips_enabled: state.tooltipSettings.enabled,
            sport_preferences: state.preferredSports,
            tutorial_completed: state.tutorialProgress.completed.length > 0,
          })
          
          set({ lastSyncedAt: new Date().toISOString() })
        } catch (error) {
          console.warn('Failed to save user preferences:', error)
        }
      },
      
      syncWithBackend: async () => {
        const { phoneToken, supabaseSession, authMethod } = useUnifiedAuthStore.getState()
        const token = authMethod === 'phone' ? phoneToken : supabaseSession?.access_token
        if (!token) return
        
        await get().loadUserPreferences()
      },
    }),
    {
      name: 'dfs-preferences',
      version: 1,
      // Only persist user preferences, not loading states
      partialize: (state) => ({
        beginnerMode: state.beginnerMode,
        tooltipSettings: state.tooltipSettings,
        aiSettings: state.aiSettings,
        tutorialProgress: state.tutorialProgress,
        preferredSports: state.preferredSports,
        lastSyncedAt: state.lastSyncedAt,
      }),
    }
  )
)