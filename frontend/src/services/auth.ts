// Real authentication service integrated with phone/OTP system
// This service is now managed by the Zustand auth store

export function getAuthToken(): string | null {
  return localStorage.getItem('auth_token')
}

export function setAuthToken(token: string): void {
  localStorage.setItem('auth_token', token)
}

export function clearAuthToken(): void {
  localStorage.removeItem('auth_token')
}

// Check if user is authenticated
export function isAuthenticated(): boolean {
  return !!getAuthToken()
}

console.log('Authentication service initialized with phone/OTP system')