export interface Contest {
  id: number
  platform: 'draftkings' | 'fanduel'
  sport: 'nba' | 'nfl' | 'mlb' | 'nhl' | 'golf'
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
  tournament_id?: string
  tournament?: {
    id: string
    external_id: string
    name: string
    start_date: string
    end_date: string
    status: string
    course_name: string
    course_par: number
    course_yards: number
    purse: number
    fedex_points: number
    field_strength: number
  }
}

export interface ContestStats {
  total_lineups: number
  unique_users: number
  avg_projected: number
  highest_projected: number
  fill_percentage: number
}