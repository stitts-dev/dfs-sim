import axios from 'axios'

// Use relative URL so Vite proxy can handle it
const API_BASE = '/api/v1'

const api = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth token to requests if available
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export interface UserPreferences {
  user_id: number
  beginner_mode: boolean
  show_tooltips: boolean
  tooltip_delay: number
  preferred_sports: string[]
  ai_suggestions_enabled: boolean
  created_at: string
  updated_at: string
}

export interface UpdatePreferencesRequest {
  beginner_mode?: boolean
  show_tooltips?: boolean
  tooltip_delay?: number
  preferred_sports?: string[]
  ai_suggestions_enabled?: boolean
}

// Get user preferences
export const getPreferences = async (): Promise<UserPreferences> => {
  const { data } = await api.get('/user/preferences')
  return data.data
}

// Update user preferences
export const updatePreferences = async (
  updates: UpdatePreferencesRequest
): Promise<UserPreferences> => {
  const { data } = await api.put('/user/preferences', updates)
  return data.data
}

// Reset preferences to defaults
export const resetPreferences = async (): Promise<UserPreferences> => {
  const { data } = await api.post('/user/preferences/reset')
  return data.data
}

export interface MigratePreferencesRequest {
  beginner_mode: boolean
  show_tooltips: boolean
  tooltip_delay: number
  preferred_sports: string[]
  ai_suggestions_enabled: boolean
}

// Migrate anonymous preferences to authenticated user during signup
export const migratePreferences = async (
  preferences: MigratePreferencesRequest
): Promise<UserPreferences> => {
  const { data } = await api.post('/user/preferences/migrate', preferences)
  return data.data
}

// Helper to sync preferences with local storage
export const syncPreferencesWithLocalStorage = (_preferences: UserPreferences) => {
  // This is handled by the zustand persist middleware
  // but we can add additional sync logic here if needed
}

// Helper to get preference value with fallback
export const getPreferenceValue = <T>(
  preferences: UserPreferences | null,
  key: keyof UserPreferences,
  defaultValue: T
): T => {
  if (!preferences) return defaultValue
  return (preferences[key] as T) ?? defaultValue
}