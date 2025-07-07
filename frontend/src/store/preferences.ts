import { create } from 'zustand'
import { persist } from 'zustand/middleware'

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
  
  // Actions
  setBeginnerMode: (enabled: boolean) => void
  updateTooltipSettings: (settings: Partial<PreferencesState['tooltipSettings']>) => void
  updateAISettings: (settings: Partial<PreferencesState['aiSettings']>) => void
  completeTutorialStep: (stepId: string) => void
  skipTutorialStep: (stepId: string) => void
  resetTutorial: () => void
  setPreferredSports: (sports: string[]) => void
}

export const usePreferencesStore = create<PreferencesState>()(
  persist(
    (set) => ({
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
    }),
    {
      name: 'dfs-preferences',
      version: 1,
    }
  )
)