import { useState, useEffect, useCallback } from 'react'
import { usePreferencesStore } from '@/store/preferences'
import * as preferencesService from '@/services/preferences'
import { isAuthenticated } from '@/services/auth'

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
      
      // Check if user is authenticated
      const authenticated = isAuthenticated()
      
      if (!authenticated) {
        // For anonymous users, use local store only
        const mockPrefs: preferencesService.UserPreferences = {
          user_id: 0, // 0 indicates anonymous user
          beginner_mode: store.beginnerMode,
          show_tooltips: store.tooltipSettings.enabled,
          tooltip_delay: store.tooltipSettings.delay,
          preferred_sports: store.preferredSports,
          ai_suggestions_enabled: store.aiSettings.enabled,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }
        setPreferences(mockPrefs)
        return
      }
      
      // For authenticated users, fetch from API
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
      // If API fails, fall back to local store
      console.warn('Failed to fetch preferences:', err)
      
      // Create mock preferences from local store
      const mockPrefs: preferencesService.UserPreferences = {
        user_id: isAuthenticated() ? 1 : 0,
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
      
      // Check if user is authenticated
      const authenticated = isAuthenticated()
      
      if (!authenticated) {
        // For anonymous users, only update local store
        const mockPrefs: preferencesService.UserPreferences = {
          user_id: 0, // 0 indicates anonymous user
          beginner_mode: store.beginnerMode,
          show_tooltips: store.tooltipSettings.enabled,
          tooltip_delay: store.tooltipSettings.delay,
          preferred_sports: store.preferredSports,
          ai_suggestions_enabled: store.aiSettings.enabled,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }
        setPreferences(mockPrefs)
        return mockPrefs
      }
      
      // For authenticated users, sync with API
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
      
      // Check if user is authenticated
      const authenticated = isAuthenticated()
      
      // Define default preferences
      const defaultPrefs: preferencesService.UserPreferences = {
        user_id: authenticated ? 1 : 0,
        beginner_mode: true,
        show_tooltips: true,
        tooltip_delay: 500,
        preferred_sports: [],
        ai_suggestions_enabled: true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      
      if (!authenticated) {
        // For anonymous users, reset local store only
        store.setBeginnerMode(defaultPrefs.beginner_mode)
        store.updateTooltipSettings({
          enabled: defaultPrefs.show_tooltips,
          delay: defaultPrefs.tooltip_delay,
        })
        store.updateAISettings({
          enabled: defaultPrefs.ai_suggestions_enabled,
        })
        store.setPreferredSports(defaultPrefs.preferred_sports || [])
        setPreferences(defaultPrefs)
        return defaultPrefs
      }
      
      // For authenticated users, reset via API
      const resetPrefs = await preferencesService.resetPreferences()
      setPreferences(resetPrefs)
      
      // Sync with local store
      store.setBeginnerMode(resetPrefs.beginner_mode)
      store.updateTooltipSettings({
        enabled: resetPrefs.show_tooltips,
        delay: resetPrefs.tooltip_delay,
      })
      store.updateAISettings({
        enabled: resetPrefs.ai_suggestions_enabled,
      })
      store.setPreferredSports(resetPrefs.preferred_sports || [])
      
      return resetPrefs
    } catch (err) {
      console.error('Failed to reset preferences:', err)
      setError('Failed to reset preferences')
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [store])

  const migratePreferences = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      
      // Get current local preferences
      const migrationData: preferencesService.MigratePreferencesRequest = {
        beginner_mode: store.beginnerMode,
        show_tooltips: store.tooltipSettings.enabled,
        tooltip_delay: store.tooltipSettings.delay,
        preferred_sports: store.preferredSports,
        ai_suggestions_enabled: store.aiSettings.enabled,
      }
      
      // Migrate to authenticated user
      const migratedPrefs = await preferencesService.migratePreferences(migrationData)
      setPreferences(migratedPrefs)
      
      return migratedPrefs
    } catch (err) {
      console.error('Failed to migrate preferences:', err)
      setError('Failed to migrate preferences')
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
    migratePreferences,
    refetch: fetchPreferences,
  }
}