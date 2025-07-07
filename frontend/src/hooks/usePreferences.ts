import { useState, useEffect, useCallback } from 'react'
import { usePreferencesStore } from '@/store/preferences'
import * as preferencesService from '@/services/preferences'

export function usePreferences() {
  const [preferences, setPreferences] = useState<preferencesService.UserPreferences | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  
  const store = usePreferencesStore()

  // Fetch preferences on mount
  useEffect(() => {
    fetchPreferences()
  }, [])

  const fetchPreferences = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const prefs = await preferencesService.getPreferences()
      setPreferences(prefs)
      
      // Sync with local store
      store.setBeginnerMode(prefs.beginner_mode)
      store.updateTooltipSettings({
        enabled: prefs.show_tooltips,
        delay: prefs.tooltip_delay,
      })
      store.updateAISettings({
        enabled: prefs.ai_suggestions_enabled,
      })
      store.setPreferredSports(prefs.preferred_sports || [])
    } catch (err) {
      // If user is not authenticated or preferences don't exist,
      // we'll just use local store defaults
      console.warn('Failed to fetch preferences:', err)
      
      // Create mock preferences from local store
      const mockPrefs: preferencesService.UserPreferences = {
        user_id: 1,
        beginner_mode: store.beginnerMode,
        show_tooltips: store.tooltipSettings.enabled,
        tooltip_delay: store.tooltipSettings.delay,
        preferred_sports: store.preferredSports,
        ai_suggestions_enabled: store.aiSettings.enabled,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      setPreferences(mockPrefs)
    } finally {
      setIsLoading(false)
    }
  }, [store])

  const updatePreferences = useCallback(async (
    updates: preferencesService.UpdatePreferencesRequest
  ) => {
    try {
      setIsLoading(true)
      setError(null)
      
      // Optimistic update in local store
      if (updates.beginner_mode !== undefined) {
        store.setBeginnerMode(updates.beginner_mode)
      }
      if (updates.show_tooltips !== undefined || updates.tooltip_delay !== undefined) {
        store.updateTooltipSettings({
          enabled: updates.show_tooltips,
          delay: updates.tooltip_delay,
        })
      }
      if (updates.ai_suggestions_enabled !== undefined) {
        store.updateAISettings({
          enabled: updates.ai_suggestions_enabled,
        })
      }
      
      const updatedPrefs = await preferencesService.updatePreferences(updates)
      setPreferences(updatedPrefs)
      
      return updatedPrefs
    } catch (err) {
      // Revert optimistic updates on error
      if (preferences) {
        store.setBeginnerMode(preferences.beginner_mode)
        store.updateTooltipSettings({
          enabled: preferences.show_tooltips,
          delay: preferences.tooltip_delay,
        })
        store.updateAISettings({
          enabled: preferences.ai_suggestions_enabled,
        })
      }
      
      console.error('Failed to update preferences:', err)
      setError('Failed to save preferences')
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [preferences, store])

  const resetPreferences = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      
      const defaultPrefs = await preferencesService.resetPreferences()
      setPreferences(defaultPrefs)
      
      // Sync with local store
      store.setBeginnerMode(defaultPrefs.beginner_mode)
      store.updateTooltipSettings({
        enabled: defaultPrefs.show_tooltips,
        delay: defaultPrefs.tooltip_delay,
      })
      store.updateAISettings({
        enabled: defaultPrefs.ai_suggestions_enabled,
      })
      store.setPreferredSports(defaultPrefs.preferred_sports || [])
      
      return defaultPrefs
    } catch (err) {
      console.error('Failed to reset preferences:', err)
      setError('Failed to reset preferences')
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [store])

  return {
    preferences,
    isLoading,
    error,
    updatePreferences,
    resetPreferences,
    refetch: fetchPreferences,
  }
}