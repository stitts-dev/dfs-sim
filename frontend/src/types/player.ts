export interface Player {
  id: number
  external_id: string
  name: string
  team: string
  opponent: string
  position: string
  salary: number
  projected_points: number
  floor_points: number
  ceiling_points: number
  ownership: number
  sport: string
  contest_id: number
  game_time: string
  is_injured: boolean
  injury_status?: string
  created_at: string
  updated_at: string
}

export interface PlayerStats {
  player_id: number
  last_5_games: GameStat[]
  season_avg: number
  home_avg: number
  away_avg: number
  vs_opponent_avg: number
}

export interface GameStat {
  date: string
  points: number
  opponent: string
}