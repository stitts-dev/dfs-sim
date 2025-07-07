export interface Contest {
  id: number
  platform: 'draftkings' | 'fanduel'
  sport: 'nba' | 'nfl' | 'mlb' | 'nhl'
  contest_type: 'gpp' | 'cash'
  name: string
  entry_fee: number
  prize_pool: number
  max_entries: number
  total_entries: number
  salary_cap: number
  start_time: string
  is_active: boolean
  is_multi_entry: boolean
  max_lineups_per_user: number
  position_requirements: Record<string, number>
  created_at: string
  updated_at: string
}

export interface ContestStats {
  total_lineups: number
  unique_users: number
  avg_projected: number
  highest_projected: number
  fill_percentage: number
}