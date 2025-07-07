// AI Recommendation Types

export interface PlayerRecommendationRequest {
  contest_id: number
  contest_type: 'GPP' | 'Cash'
  sport: string
  remaining_budget: number
  current_lineup: number[]
  positions_needed: string[]
  beginner_mode?: boolean
  optimize_for?: 'ceiling' | 'floor' | 'balanced'
}

export interface PlayerRecommendation {
  player_id: number
  player_name: string
  position: string
  team: string
  salary: number
  projected_points: number
  confidence: number
  reasoning: string
  beginner_tip?: string
  stack_with?: string[]
  avoid_with?: string[]
}

export interface LineupAnalysisRequest {
  lineup_id: number
  contest_type: 'GPP' | 'Cash'
  sport: string
}

export interface LineupAnalysis {
  overall_score: number
  strengths: string[]
  weaknesses: string[]
  improvements: string[]
  stacking_analysis: Record<string, any>
  risk_level: 'low' | 'medium' | 'high'
  beginner_insights?: string[]
}

export interface AIRecommendationResponse {
  recommendations: PlayerRecommendation[]
  request: {
    contest_id: number
    contest_type: string
    sport: string
    remaining_budget: number
    optimize_for: string
    positions_needed: string[]
  }
}

export interface AIAnalysisResponse {
  analysis: LineupAnalysis
  lineup_id: number
}

export interface RecommendationHistoryItem {
  id: number
  created_at: string
  recommendations: PlayerRecommendation[]
  contest_type: string
  sport: string
}