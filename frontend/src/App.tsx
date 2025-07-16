import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from 'react-query'
import { Toaster } from 'react-hot-toast'
import Dashboard from './pages/Dashboard'
import Optimizer from './pages/Optimizer'
import Lineups from './pages/Lineups'
import EnhancedLoginPage from './pages/auth/EnhancedLoginPage'
import EnhancedSignupPage from './pages/auth/EnhancedSignupPage'
import AuthCallback from './pages/auth/AuthCallback'
import Layout from './components/Layout'
import ProtectedRoute from './components/auth/ProtectedRoute'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 5 * 60 * 1000, // 5 minutes
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Router>
        <Routes>
          {/* Public Routes */}
          <Route path="/auth/login" element={<EnhancedLoginPage />} />
          <Route path="/auth/signup" element={<EnhancedSignupPage />} />
          <Route path="/auth/callback" element={<AuthCallback />} />
          {/* Legacy password reset redirect */}
          <Route path="/reset-password" element={<AuthCallback />} />

          {/* Protected Routes */}
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <Layout>
                  <Dashboard />
                </Layout>
              </ProtectedRoute>
            }
          />
          <Route
            path="/optimizer"
            element={
              <ProtectedRoute>
                <Layout>
                  <Optimizer />
                </Layout>
              </ProtectedRoute>
            }
          />
          <Route
            path="/lineups"
            element={
              <ProtectedRoute>
                <Layout>
                  <Lineups />
                </Layout>
              </ProtectedRoute>
            }
          />

          {/* Catch all route - redirect to login */}
          <Route path="*" element={<Navigate to="/auth/login" replace />} />
        </Routes>
        <Toaster
          position="top-right"
          toastOptions={{
            duration: 4000,
            style: {
              background: '#363636',
              color: '#fff',
            },
            success: {
              duration: 3000,
              iconTheme: {
                primary: '#10b981',
                secondary: '#fff',
              },
            },
            error: {
              duration: 5000,
              iconTheme: {
                primary: '#ef4444',
                secondary: '#fff',
              },
            },
          }}
        />
      </Router>
    </QueryClientProvider>
  )
}

export default App
