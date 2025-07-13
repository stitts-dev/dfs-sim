import { Player } from './player'

export interface OptimizeConfig {
  contest_id: number
  num_lineups: number
  min_different_players: number
  use_correlations: boolean
  correlation_weight: number
  stacking_rules: StackingRule[]
  locked_players: number[]
  excluded_players: number[]
  min_exposure: Record<number, number>
  max_exposure: Record<number, number>
}

export interface StackingRule {
  type: 'team' | 'game' | 'mini' | 'qb_stack'
  min_players: number
  max_players: number
  teams?: string[]
}

export interface OptimizerResult {
  lineups: Array<{
    id?: number
    players: Player[]
    total_salary: number
    projected_points: number
    simulated_ceiling: number
    simulated_floor: number
    simulated_mean: number
  }>
  optimization_time_ms: number
  total_combinations: number
  valid_combinations: number
}

export interface OptimizationPreset {
  name: string
  description: string
  config: Partial<OptimizeConfig>
}

export interface LineupConstraints {
  salary_cap: number
  positions: PositionConstraint[]
  team_limits: {
    min_players_per_team: number
    max_players_per_team: number
    min_unique_teams: number
  }
  game_limits: {
    min_players_per_game: number
    max_players_per_game: number
    min_unique_games: number
  }
  total_players: number
  sport: string
  platform: string
}

export interface PositionConstraint {
  position: string
  min_required: number
  max_allowed: number
  eligible_slots: string[]
}