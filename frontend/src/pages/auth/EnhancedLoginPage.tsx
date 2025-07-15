import { Navigate, useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/auth'
import { AuthWizard } from '@/components/auth/AuthWizard'

export default function EnhancedLoginPage() {
  const navigate = useNavigate()
  const { isAuthenticated } = useAuthStore()

  // Redirect if already authenticated
  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />
  }

  const handleComplete = (user: any) => {
    console.log('Login completed for user:', user)
    navigate('/dashboard')
  }

  return (
    <AuthWizard
      initialMode="login"
      initialStep="welcome"
      onComplete={handleComplete}
      onClose={() => navigate('/')}
    />
  )
}