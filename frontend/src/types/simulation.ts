export interface SimulationConfig {
  lineup_id: number
  num_simulations: number
  use_correlations: boolean
  contest_size: number
  payout_structure?: PayoutTier[]
  entry_fee: number
}

export interface PayoutTier {
  min_rank: number
  max_rank: number
  payout: number
}

export interface SimulationResult {
  id: number
  lineup_id: number
  contest_id: number
  num_simulations: number
  mean: number
  median: number
  standard_deviation: number
  min: number
  max: number
  percentile_25: number
  percentile_75: number
  percentile_90: number
  percentile_95: number
  percentile_99: number
  top_percent_finishes: {
    top_1: number
    top_10: number
    top_20: number
    top_50: number
  }
  win_probability: number
  cash_probability: number
  roi: number
  created_at: string
  updated_at: string
}

export interface SimulationProgress {
  lineup_id: number
  completed: number
  total: number
  percentage: number
  eta_seconds: number
}