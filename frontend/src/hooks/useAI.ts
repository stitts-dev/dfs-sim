import { useState, useCallback } from 'react'
import { Player } from '@/types/player'
import { Contest } from '@/types/contest'
import { aiService } from '@/services/ai'
import { 
  PlayerRecommendation, 
  PlayerRecommendationRequest,
  LineupAnalysisRequest 
} from '@/types/ai'
import toast from 'react-hot-toast'

// Use the PlayerRecommendation type from ai types
type AIRecommendation = PlayerRecommendation

interface AIRecommendationRequest {
  contest_id: number
  contest_type: 'GPP' | 'Cash'
  sport: string
  remaining_budget: number
  current_lineup: number[]
  positions_needed: string[]
  optimize_for: 'ceiling' | 'floor'
}

export function useAI() {
  const [isAIOpen, setIsAIOpen] = useState(false)
  const [recommendations, setRecommendations] = useState<AIRecommendation[] | null>(null)
  const [isLoadingRecommendations, setIsLoadingRecommendations] = useState(false)

  const toggleAI = useCallback(() => {
    setIsAIOpen(prev => !prev)
  }, [])

  const getRecommendations = useCallback(async (
    request: AIRecommendationRequest,
    _contest: Contest,
    _availablePlayers: Player[],
    _currentLineup: Player[]
  ) => {
    setIsLoadingRecommendations(true)
    
    try {
      // Create the request for the backend API
      const apiRequest: PlayerRecommendationRequest = {
        contest_id: request.contest_id,
        contest_type: request.contest_type,
        sport: request.sport,
        remaining_budget: request.remaining_budget,
        current_lineup: request.current_lineup,
        positions_needed: request.positions_needed,
        optimize_for: request.optimize_for === 'ceiling' ? 'ceiling' : 'floor',
        beginner_mode: false
      }

      const response = await aiService.getPlayerRecommendations(apiRequest)
      
      // The response already contains properly formatted recommendations
      const recommendations = response.recommendations

      setRecommendations(recommendations)
      
      if (recommendations.length === 0) {
        toast.error('No suitable players found. Try adjusting your constraints.')
      }
    } catch (error) {
      console.error('Failed to get AI recommendations:', error)
      toast.error('Failed to get AI recommendations. Please try again.')
    } finally {
      setIsLoadingRecommendations(false)
    }
  }, [])

  const resetRecommendations = useCallback(() => {
    setRecommendations(null)
  }, [])

  return {
    isAIOpen,
    recommendations,
    isLoadingRecommendations,
    toggleAI,
    getRecommendations,
    resetRecommendations,
  }
}

export function useApplyRecommendations() {
  const applyRecommendations = useCallback((
    recommendations: AIRecommendation[],
    onAddPlayers: (playerIds: number[]) => void
  ) => {
    const playerIds = recommendations.map(rec => rec.player_id)
    onAddPlayers(playerIds)
    toast.success(`Added ${recommendations.length} recommended players to your lineup!`)
  }, [])

  return { applyRecommendations }
}

export function useLineupAnalysis() {
  const [analysis, setAnalysis] = useState<{
    strengths: string[]
    weaknesses: string[]
    suggestions: string[]
    score: number
  } | null>(null)
  const [isAnalyzing, setIsAnalyzing] = useState(false)

  const analyzeLineup = useCallback(async (lineup: Player[], contest: Contest, lineupId: number) => {
    // Validate inputs
    if (!lineup || lineup.length === 0) {
      setAnalysis(null)
      return
    }

    // Check if lineup ID is valid (not a temporary/fake ID)
    if (!lineupId || lineupId <= 0) {
      toast.error('Please save your lineup before analyzing')
      return
    }

    setIsAnalyzing(true)
    
    try {
      // Create the request for the backend API
      const request: LineupAnalysisRequest = {
        lineup_id: lineupId,
        contest_type: contest.contest_type.toUpperCase() as 'GPP' | 'Cash',
        sport: contest.sport
      }

      const response = await aiService.analyzeLineup(request)
      
      // Convert the backend response to the expected format
      setAnalysis({
        strengths: response.analysis.strengths,
        weaknesses: response.analysis.weaknesses,
        suggestions: response.analysis.improvements,
        score: response.analysis.overall_score
      })
    } catch (error: any) {
      console.error('Failed to analyze lineup:', error)
      
      // Provide more specific error messages
      if (error.response?.data?.error?.message) {
        toast.error(error.response.data.error.message)
      } else if (error.response?.status === 404) {
        toast.error('Lineup not found. Please save your lineup first.')
      } else {
        toast.error('Failed to analyze lineup. Please try again.')
      }
    } finally {
      setIsAnalyzing(false)
    }
  }, [])

  return {
    analysis,
    isAnalyzing,
    analyzeLineup,
  }
}