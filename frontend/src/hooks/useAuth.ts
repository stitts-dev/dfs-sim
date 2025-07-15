import { useAuthStore } from '@/store/auth'
import { usePhoneAuth } from './usePhoneAuth'

/**
 * Unified authentication hook that provides a clean interface for components
 * Wraps the underlying phone auth implementation
 */
export const useAuth = () => {
  const {
    user,
    token,
    isAuthenticated,
    isLoading,
    error
  } = useAuthStore()

  const { signOut } = usePhoneAuth()

  return {
    user,
    token,
    isAuthenticated,
    isLoading,
    error,
    logout: signOut,
    signOut
  }
}