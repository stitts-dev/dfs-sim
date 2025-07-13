import axios from 'axios'
import {
  PlayerRecommendationRequest,
  LineupAnalysisRequest,
  AIRecommendationResponse,
  AIAnalysisResponse,
  RecommendationHistoryItem,
  PlayerRecommendation
} from '@/types/ai'

// Use relative URL so Vite proxy can handle it
const API_BASE_URL = '/api/v1'

// Create axios instance with auth token
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth token to requests
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('authToken')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// AI API Service
export const aiService = {
  // Get player recommendations
  async getPlayerRecommendations(request: PlayerRecommendationRequest): Promise<AIRecommendationResponse> {
    const response = await apiClient.post<{ success: boolean; data: AIRecommendationResponse }>('/ai/recommend-players', request)
    return response.data.data
  },

  // Analyze a lineup
  async analyzeLineup(request: LineupAnalysisRequest): Promise<AIAnalysisResponse> {
    const response = await apiClient.post<{ success: boolean; data: AIAnalysisResponse }>('/ai/analyze-lineup', request)
    return response.data.data
  },

  // Get recommendation history
  async getRecommendationHistory(limit: number = 20): Promise<RecommendationHistoryItem[]> {
    const response = await apiClient.get<{ success: boolean; data: { recommendations: RecommendationHistoryItem[] } }>(
      `/ai/recommendations/history?limit=${limit}`
    )
    return response.data.data.recommendations
  },
}

// Helper function to format AI recommendations
export function formatRecommendation(rec: PlayerRecommendation): string {
  let text = `${rec.player_name} (${rec.position}, ${rec.team})`
  if (rec.confidence >= 0.8) {
    text = `⭐ ${text}`
  }
  return text
}

// Helper function to get confidence color
export function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.8) return 'text-green-600 dark:text-green-400'
  if (confidence >= 0.6) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-gray-600 dark:text-gray-400'
}

// Helper function to get risk level color
export function getRiskLevelColor(riskLevel: string): string {
  switch (riskLevel) {
    case 'low':
      return 'text-green-600 dark:text-green-400'
    case 'medium':
      return 'text-yellow-600 dark:text-yellow-400'
    case 'high':
      return 'text-red-600 dark:text-red-400'
    default:
      return 'text-gray-600 dark:text-gray-400'
  }
}