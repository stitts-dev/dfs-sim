import axios from 'axios'
import { LineupValidation } from '@/types/lineup'
import { OptimizeConfig, OptimizerResult, LineupConstraints } from '@/types/optimizer'
import { SimulationConfig, SimulationResult } from '@/types/simulation'

// Use relative URL so Vite proxy can handle it
const API_BASE = '/api/v1'

const api = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth token to requests if available
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Contest endpoints
export const getContests = async (params?: {
  sport?: string
  platform?: string
  contest_type?: string
  active?: string
}) => {
  const { data } = await api.get('/contests', { params })
  return data.data
}

export const getContest = async (id: number) => {
  const { data } = await api.get(`/contests/${id}`)
  return data.data.contest
}

// Player endpoints
export const getPlayers = async (contestId: number, params?: {
  position?: string
  team?: string
  minSalary?: number
  maxSalary?: number
  search?: string
  sortBy?: string
  sortOrder?: string
}) => {
  const { data } = await api.get(`/contests/${contestId}/players`, { params })
  return data.data
}

export const getPlayer = async (id: number) => {
  const { data } = await api.get(`/players/${id}`)
  return data.data
}

// Lineup endpoints
export const getLineups = async (params?: {
  contest_id?: number
  submitted?: boolean
  page?: number
  perPage?: number
}) => {
  const { data } = await api.get('/lineups', { params })
  return data
}

export const getLineup = async (id: number) => {
  const { data } = await api.get(`/lineups/${id}`)
  return data.data
}

export const createLineup = async (lineup: {
  contest_id: number
  name: string
  player_ids: number[]
}) => {
  const { data } = await api.post('/lineups', lineup)
  return data.data
}

export const updateLineup = async (id: number, updates: {
  name?: string
  player_ids?: number[]
}) => {
  const { data } = await api.put(`/lineups/${id}`, updates)
  return data.data
}

export const deleteLineup = async (id: number) => {
  const { data } = await api.delete(`/lineups/${id}`)
  return data.data
}

export const submitLineup = async (id: number) => {
  const { data } = await api.post(`/lineups/${id}/submit`)
  return data.data
}

// Optimizer endpoints
export const optimizeLineups = async (config: OptimizeConfig) => {
  const { data } = await api.post('/optimize', config)
  return data.data as OptimizerResult
}

export const validateLineup = async (contestId: number, playerIds: number[]) => {
  const { data } = await api.post('/optimize/validate', {
    contest_id: contestId,
    player_ids: playerIds,
  })
  return data.data as LineupValidation
}

export const getConstraints = async (contestId: number) => {
  const { data } = await api.get(`/optimize/constraints/${contestId}`)
  return data.data as LineupConstraints
}

// Simulation endpoints
export const runSimulation = async (config: SimulationConfig) => {
  const { data } = await api.post('/simulate', config)
  return data.data as SimulationResult
}

export const getSimulationResult = async (lineupId: number) => {
  const { data } = await api.get(`/simulations/${lineupId}`)
  return data.data as SimulationResult
}

export const batchSimulate = async (config: {
  lineup_ids: number[]
  num_simulations: number
  use_correlations: boolean
  contest_size: number
  entry_fee: number
}) => {
  const { data } = await api.post('/simulate/batch', config)
  return data.data
}

// Export endpoints
export const exportLineups = async (lineupIds: number[], format: string) => {
  const response = await api.post('/export', {
    lineup_ids: lineupIds,
    format,
  }, {
    responseType: 'blob',
  })
  
  // Create download link
  const url = window.URL.createObjectURL(new Blob([response.data]))
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', `lineups_${format}_${Date.now()}.csv`)
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.URL.revokeObjectURL(url)
}

export const getExportFormats = async (sport?: string, platform?: string) => {
  const { data } = await api.get('/export/formats', {
    params: { sport, platform },
  })
  return data.data
}

// WebSocket connection
export const connectWebSocket = () => {
  const wsUrl = API_BASE.replace('http', 'ws').replace('/api/v1', '/ws')
  return new WebSocket(wsUrl)
}