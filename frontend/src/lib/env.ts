// Environment variable utilities for type-safe access

interface EnvConfig {
  apiUrl: string
  golfOnlyMode: boolean
  supportedSports: string[]
  enableOptimization: boolean
  enableSimulation: boolean
  enableAIRecommendations: boolean
  enableDebugLogs: boolean
}

// Parse comma-separated environment variable into array
function parseStringArray(value: string | undefined): string[] {
  if (!value) return []
  return value.split(',').map(s => s.trim()).filter(Boolean)
}

// Parse boolean environment variable
function parseBoolean(value: string | undefined): boolean {
  return value === 'true'
}

// Get environment configuration with fallbacks
export const env: EnvConfig = {
  apiUrl: import.meta.env.VITE_API_URL || '/api/v1',
  golfOnlyMode: parseBoolean(import.meta.env.VITE_GOLF_ONLY_MODE),
  supportedSports: parseStringArray(import.meta.env.VITE_SUPPORTED_SPORTS),
  enableOptimization: parseBoolean(import.meta.env.VITE_ENABLE_OPTIMIZATION ?? 'true'),
  enableSimulation: parseBoolean(import.meta.env.VITE_ENABLE_SIMULATION ?? 'true'),
  enableAIRecommendations: parseBoolean(import.meta.env.VITE_ENABLE_AI_RECOMMENDATIONS ?? 'true'),
  enableDebugLogs: parseBoolean(import.meta.env.VITE_ENABLE_DEBUG_LOGS),
}

// Helper function to check if a sport is enabled by environment
export function isSportEnabledByEnv(sportId: string): boolean {
  if (env.golfOnlyMode) {
    return sportId === 'golf'
  }
  
  if (env.supportedSports.length === 0) {
    // If no sports specified in env, default to backend configuration
    return true
  }
  
  return env.supportedSports.includes(sportId)
}

// Helper to get fallback sports if API fails
export function getFallbackSports() {
  if (env.golfOnlyMode) {
    return [{ id: 'golf', name: 'Golf', icon: '⛳', enabled: true }]
  }
  
  const allSports = [
    { id: 'golf', name: 'Golf', icon: '⛳' },
    { id: 'nba', name: 'NBA', icon: '🏀' },
    { id: 'nfl', name: 'NFL', icon: '🏈' },
    { id: 'mlb', name: 'MLB', icon: '⚾' },
    { id: 'nhl', name: 'NHL', icon: '🏒' },
  ]
  
  if (env.supportedSports.length === 0) {
    return allSports.map(sport => ({ ...sport, enabled: true }))
  }
  
  return allSports.map(sport => ({
    ...sport,
    enabled: env.supportedSports.includes(sport.id)
  })).filter(sport => sport.enabled)
}

// Debug logging utility
export function debugLog(...args: unknown[]) {
  if (env.enableDebugLogs) {
    console.log('[DEBUG]', ...args)
  }
}