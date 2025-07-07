import { Contest } from './contest'
import { Player } from './player'

export interface Lineup {
  id: number
  user_id: number
  contest_id: number
  name: string
  total_salary: number
  projected_points: number
  actual_points?: number
  simulated_ceiling: number
  simulated_floor: number
  simulated_mean: number
  ownership: number
  is_submitted: boolean
  is_optimized: boolean
  optimization_rank?: number
  created_at: string
  updated_at: string
  contest?: Contest
  players: Player[]
}

export interface LineupValidation {
  valid: boolean
  errors: string[]
  warnings: string[]
  total_salary: number
  salary_cap: number
  remaining_salary: number
  projected_points: number
  position_counts: Record<string, number>
}

export interface LineupExport {
  lineup_id: number
  name: string
  total_salary: number
  projected_points: number
  players: Array<{
    id: number
    name: string
    position: string
    team: string
    opponent: string
    salary: number
    projected_points: number
    ownership?: number
    floor_points?: number
    ceiling_points?: number
  }>
  team_exposure: Record<string, number>
  game_exposure: Record<string, number>
}