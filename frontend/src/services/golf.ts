import {
  GolfTournament,
  GolfLeaderboard,
  GolfPlayer,
  GolfPlayerEntry,
  GolfProjection,
  GolfCourseHistory,
  GolfOptimizationParams,
  GolfLineup,
} from '@/types/golf'

// Use the same pattern as api.ts
const API_BASE_URL = '/api/v1'

const golfAPI = {
  // Tournament endpoints
  async getTournaments(params?: {
    status?: string
    limit?: number
    offset?: number
  }): Promise<{ tournaments: GolfTournament[]; total: number }> {
    const queryParams = new URLSearchParams()
    if (params?.status) queryParams.append('status', params.status)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.offset) queryParams.append('offset', params.offset.toString())

    const response = await fetch(
      `${API_BASE_URL}/golf/tournaments?${queryParams}`,
      {
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      throw new Error('Failed to fetch tournaments')
    }

    return response.json()
  },

  async getTournamentSchedule(year?: number): Promise<{
    tournaments: GolfTournament[]
    total_year: number
    year: number
    source: string
    cached_at: string
    next_update: string
  }> {
    const queryParams = new URLSearchParams()
    if (year) queryParams.append('year', year.toString())

    const response = await fetch(
      `${API_BASE_URL}/golf/tournaments/schedule?${queryParams}`,
      {
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      throw new Error('Failed to fetch tournament schedule')
    }

    return response.json()
  },

  async getTournament(tournamentId: string): Promise<GolfTournament> {
    const response = await fetch(
      `${API_BASE_URL}/golf/tournaments/${tournamentId}`,
      {
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      throw new Error('Failed to fetch tournament')
    }

    return response.json()
  },

  async getTournamentLeaderboard(tournamentId: string): Promise<GolfLeaderboard> {
    const response = await fetch(
      `${API_BASE_URL}/golf/tournaments/${tournamentId}/leaderboard`,
      {
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      throw new Error('Failed to fetch leaderboard')
    }

    return response.json()
  },

  async getTournamentPlayers(
    tournamentId: string,
    platform: 'draftkings' | 'fanduel' = 'draftkings'
  ): Promise<{ tournament: GolfTournament; players: GolfPlayer[]; platform: string }> {
    const response = await fetch(
      `${API_BASE_URL}/golf/tournaments/${tournamentId}/players?platform=${platform}`,
      {
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      throw new Error('Failed to fetch tournament players')
    }

    return response.json()
  },

  async getTournamentProjections(
    tournamentId: string
  ): Promise<{
    tournament: GolfTournament
    projections: Record<string, GolfProjection>
    correlations: Record<string, Record<string, number>>
  }> {
    const response = await fetch(
      `${API_BASE_URL}/golf/tournaments/${tournamentId}/projections`,
      {
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      throw new Error('Failed to fetch projections')
    }

    return response.json()
  },

  // Player endpoints
  async getPlayerHistory(
    playerId: string,
    courseId?: string
  ): Promise<{ player_id: string; histories: GolfCourseHistory[] }> {
    const queryParams = new URLSearchParams()
    if (courseId) queryParams.append('course_id', courseId)

    const response = await fetch(
      `${API_BASE_URL}/golf/players/${playerId}/history?${queryParams}`,
      {
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      throw new Error('Failed to fetch player history')
    }

    return response.json()
  },

  // Admin endpoints
  async syncTournamentData(
    tournamentId: string
  ): Promise<{ message: string; tournament: GolfTournament; player_count: number }> {
    const response = await fetch(
      `${API_BASE_URL}/golf/tournaments/${tournamentId}/sync`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      throw new Error('Failed to sync tournament data')
    }

    return response.json()
  },

  // Optimization endpoints
  async optimizeGolfLineups(
    params: GolfOptimizationParams
  ): Promise<{ lineups: GolfLineup[]; correlations: Record<string, Record<string, number>> }> {
    const response = await fetch(`${API_BASE_URL}/optimize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        contest_type: 'golf',
        sport: 'golf',
        ...params,
      }),
    })

    if (!response.ok) {
      throw new Error('Failed to optimize lineups')
    }

    return response.json()
  },

  // Utility functions
  formatScore(score: number): string {
    if (score === 0) return 'E'
    return score > 0 ? `+${score}` : score.toString()
  },

  formatPosition(position: number, tied: boolean = false): string {
    if (position === 0) return '-'
    const suffix = tied ? 'T' : ''
    return `${suffix}${position}`
  },

  calculateCutLine(entries: GolfPlayerEntry[]): number {
    // Simplified cut calculation - top 70 and ties
    const sortedEntries = [...entries].sort((a, b) => a.total_score - b.total_score)
    if (sortedEntries.length <= 70) return 999 // Everyone makes cut
    
    const cutPosition = Math.min(70, Math.floor(sortedEntries.length * 0.7))
    return sortedEntries[cutPosition - 1]?.total_score || 0
  },
}

export default golfAPI