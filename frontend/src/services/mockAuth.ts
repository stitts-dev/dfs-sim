// Mock authentication service for development
export const mockAuth = {
  // Mock JWT token
  getToken: () => {
    // In a real app, this would come from login
    return 'mock-jwt-token-for-development'
  },

  // Set mock token in localStorage
  setMockAuth: () => {
    localStorage.setItem('auth_token', 'mock-jwt-token-for-development')
    localStorage.setItem('authToken', 'mock-jwt-token-for-development')
  },

  // Clear auth
  clearAuth: () => {
    localStorage.removeItem('auth_token')
    localStorage.removeItem('authToken')
  },

  // Check if authenticated
  isAuthenticated: () => {
    return !!localStorage.getItem('auth_token')
  }
}

// Auto-set mock auth on load for development
if (import.meta.env.DEV) {
  mockAuth.setMockAuth()
}