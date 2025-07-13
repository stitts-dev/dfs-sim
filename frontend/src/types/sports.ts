// Sport configuration types

export interface SportInfo {
  id: string
  name: string
  icon: string
  enabled: boolean
  positions?: string[]
}

export interface SportsConfiguration {
  sports: SportInfo[]
  golf_only_mode: boolean
  all_sports: SportInfo[]
}

export type ValidSportId = 'golf' | 'nba' | 'nfl' | 'mlb' | 'nhl'

// Type guard to check if a string is a valid sport ID
export function isValidSportId(value: string): value is ValidSportId {
  return ['golf', 'nba', 'nfl', 'mlb', 'nhl'].includes(value)
}

// Helper to get sport display info
export function getSportDisplayInfo(sportId: string): Partial<SportInfo> {
  const sportMap: Record<string, Partial<SportInfo>> = {
    golf: { name: 'Golf', icon: '⛳' },
    nba: { name: 'NBA', icon: '🏀' },
    nfl: { name: 'NFL', icon: '🏈' },
    mlb: { name: 'MLB', icon: '⚾' },
    nhl: { name: 'NHL', icon: '🏒' },
  }
  
  return sportMap[sportId] || { name: sportId.toUpperCase(), icon: '🏆' }
}

// Position requirements by sport
export const SPORT_POSITIONS: Record<ValidSportId, string[]> = {
  golf: ['G'],
  nba: ['PG', 'SG', 'SF', 'PF', 'C', 'G', 'F', 'UTIL'],
  nfl: ['QB', 'RB', 'WR', 'TE', 'K', 'DST', 'FLEX'],
  mlb: ['C', '1B', '2B', '3B', 'SS', 'OF', 'DH', 'P'],
  nhl: ['C', 'LW', 'RW', 'D', 'G', 'F'],
}