// Enhanced API client with automatic token refresh and error handling
import { useAuthStore } from '@/store/auth'

// API response interceptor for handling expired tokens
export const createApiClient = () => {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
  
  const makeRequest = async (endpoint: string, options: RequestInit = {}): Promise<Response> => {
    const { token, refreshToken, logout } = useAuthStore.getState()
    
    // Add Authorization header if token exists
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> || {}),
    }
    
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
    
    const requestOptions: RequestInit = {
      ...options,
      headers,
    }
    
    let response = await fetch(`${apiUrl}${endpoint}`, requestOptions)
    
    // If token expired (401), try to refresh
    if (response.status === 401 && token) {
      try {
        console.log('Token expired, attempting refresh...')
        await refreshToken()
        
        // Retry request with new token
        const { token: newToken } = useAuthStore.getState()
        if (newToken) {
          headers['Authorization'] = `Bearer ${newToken}`
          response = await fetch(`${apiUrl}${endpoint}`, {
            ...requestOptions,
            headers,
          })
        }
      } catch (refreshError) {
        console.warn('Token refresh failed during API request:', refreshError)
        // Force logout on refresh failure
        logout()
        throw new Error('Session expired. Please log in again.')
      }
    }
    
    return response
  }
  
  return {
    get: (endpoint: string, options?: RequestInit) => 
      makeRequest(endpoint, { ...options, method: 'GET' }),
    
    post: (endpoint: string, data?: any, options?: RequestInit) =>
      makeRequest(endpoint, {
        ...options,
        method: 'POST',
        body: data ? JSON.stringify(data) : undefined,
      }),
    
    put: (endpoint: string, data?: any, options?: RequestInit) =>
      makeRequest(endpoint, {
        ...options,
        method: 'PUT',
        body: data ? JSON.stringify(data) : undefined,
      }),
    
    delete: (endpoint: string, options?: RequestInit) =>
      makeRequest(endpoint, { ...options, method: 'DELETE' }),
  }
}

// Export a singleton instance
export const apiClient = createApiClient()

// Helper function for handling API responses
export const handleApiResponse = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}))
    throw new Error(errorData.error || errorData.message || `HTTP ${response.status}`)
  }
  
  const data = await response.json()
  return data
}

// Enhanced fetch function with error handling
export const apiFetch = async <T>(endpoint: string, options?: RequestInit): Promise<T> => {
  const response = await apiClient.get(endpoint, options)
  return handleApiResponse<T>(response)
}

// POST with automatic error handling
export const apiPost = async <T>(endpoint: string, data?: any, options?: RequestInit): Promise<T> => {
  const response = await apiClient.post(endpoint, data, options)
  return handleApiResponse<T>(response)
}

// PUT with automatic error handling
export const apiPut = async <T>(endpoint: string, data?: any, options?: RequestInit): Promise<T> => {
  const response = await apiClient.put(endpoint, data, options)
  return handleApiResponse<T>(response)
}

// DELETE with automatic error handling
export const apiDelete = async <T>(endpoint: string, options?: RequestInit): Promise<T> => {
  const response = await apiClient.delete(endpoint, options)
  return handleApiResponse<T>(response)
}