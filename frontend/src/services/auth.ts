// Mock authentication service for development
// In production, this would integrate with your actual auth system

export function getAuthToken(): string | null {
  return localStorage.getItem('authToken')
}

export function setAuthToken(token: string): void {
  localStorage.setItem('authToken', token)
}

export function clearAuthToken(): void {
  localStorage.removeItem('authToken')
}

// Mock login for development/testing
export function mockLogin(): void {
  // Generate a mock JWT token for testing
  const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOjEsImVtYWlsIjoidGVzdEB0ZXN0LmNvbSJ9.mocktoken'
  setAuthToken(mockToken)
}

// Check if user is authenticated
export function isAuthenticated(): boolean {
  return !!getAuthToken()
}

// Initialize mock auth for development
if (import.meta.env.DEV && !isAuthenticated()) {
  mockLogin()
  console.log('Mock authentication initialized for development')
}