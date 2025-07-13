export interface GolfTournament {
  id: string
  external_id: string
  name: string
  start_date: string
  end_date: string
  purse: number
  winner_share: number
  fedex_points: number
  course_id: string
  course_name: string
  course_par: number
  course_yards: number
  status: TournamentStatus
  current_round: number
  cut_line: number
  cut_rule: string
  weather_conditions: WeatherConditions
  field_strength: number
  created_at: string
  updated_at: string
  player_entries?: GolfPlayerEntry[]
}

export type TournamentStatus = 'scheduled' | 'in_progress' | 'completed' | 'postponed' | 'cancelled'

export interface WeatherConditions {
  temperature: number
  wind_speed: number
  wind_direction: string
  conditions: string
  humidity: number
}

export interface GolfPlayerEntry {
  id: string
  player_id: number
  tournament_id: string
  status: PlayerEntryStatus
  starting_position: number
  current_position: number
  total_score: number
  thru_holes: number
  rounds_scores: number[]
  tee_times: string[]
  playing_partners: string[]
  dk_salary: number
  fd_salary: number
  dk_ownership: number
  fd_ownership: number
  created_at: string
  updated_at: string
  player?: GolfPlayer
  round_scores?: GolfRoundScore[]
}

export type PlayerEntryStatus = 'entered' | 'withdrawn' | 'cut' | 'active' | 'completed'

export interface GolfPlayer {
  id: number
  external_id: string
  name: string
  team: string // Country in golf
  position: string // Always 'G' for golfer
  world_rank?: number
  recent_form?: string
  salary: number
  projected_points: number
  floor_points: number
  ceiling_points: number
  ownership: number
  cut_probability?: number
  top10_probability?: number
  top25_probability?: number
  win_probability?: number
  expected_score?: number
  confidence?: number
  status?: string
  current_position?: number
  total_score?: number
  thru_holes?: number
  rounds_scores?: number[]
}

export interface GolfRoundScore {
  id: string
  entry_id: string
  round_number: number
  holes_completed: number
  score: number
  strokes: number
  birdies: number
  eagles: number
  bogeys: number
  double_bogeys: number
  hole_scores: Record<string, number>
  started_at?: string
  completed_at?: string
  created_at: string
}

export interface GolfCourseHistory {
  id: string
  player_id: number
  course_id: string
  tournaments_played: number
  rounds_played: number
  total_strokes: number
  scoring_avg: number
  adj_scoring_avg: number
  best_finish: number
  worst_finish: number
  cuts_made: number
  missed_cuts: number
  top_10s: number
  top_25s: number
  wins: number
  strokes_gained_total: number
  sg_tee_to_green: number
  sg_putting: number
  last_played?: string
  created_at: string
  updated_at: string
}

export interface GolfProjection {
  player_id: string
  tournament_id: string
  expected_score: number
  cut_probability: number
  top10_probability: number
  top25_probability: number
  win_probability: number
  dk_points: number
  fd_points: number
  confidence: number
}

export interface GolfLeaderboard {
  tournament: GolfTournament
  entries: GolfPlayerEntry[]
  cut_line: number
  updated_at: string
}

export interface GolfOptimizationParams {
  tournament_id: string
  num_lineups: number
  platform: 'draftkings' | 'fanduel'
  min_cut_probability?: number
  max_exposure?: number
  locked_player_ids?: number[]
  excluded_player_ids?: number[]
}

export interface GolfLineup {
  id: string
  players: GolfPlayer[]
  total_salary: number
  projected_points: number
  floor_points: number
  ceiling_points: number
  cut_probability: number
  correlation_score: number
}