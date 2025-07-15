import { AlertTriangleIcon, ZapIcon, BarChart3Icon } from 'lucide-react'
import { useAuth } from '../hooks/useAuth'

interface UsageTrackerProps {
  variant?: 'header' | 'dashboard' | 'modal'
  showUpgradeButton?: boolean
}

export function UsageTracker({ variant = 'header', showUpgradeButton = false }: UsageTrackerProps) {
  const { user } = useAuth()
  
  if (!user) return null
  
  const optimizationLimit = user.subscription_tier === 'free' ? 10 : 
                           user.subscription_tier === 'pro' ? 50 : -1
  const simulationLimit = user.subscription_tier === 'free' ? 5 : 
                         user.subscription_tier === 'pro' ? 25 : -1
  
  const optimizationUsed = 0 // user.monthly_optimizations_used || 0
  const simulationUsed = 0 // user.monthly_simulations_used || 0
  
  const optimizationPercentage = optimizationLimit === -1 ? 0 : (optimizationUsed / optimizationLimit) * 100
  const simulationPercentage = simulationLimit === -1 ? 0 : (simulationUsed / simulationLimit) * 100
  
  const isOptimizationNearLimit = optimizationLimit !== -1 && optimizationPercentage >= 80
  const isSimulationNearLimit = simulationLimit !== -1 && simulationPercentage >= 80
  
  const isOptimizationAtLimit = optimizationLimit !== -1 && optimizationUsed >= optimizationLimit
  const isSimulationAtLimit = simulationLimit !== -1 && simulationUsed >= simulationLimit

  if (variant === 'header') {
    return (
      <div className="flex items-center space-x-4 text-sm">
        {/* Optimization Usage */}
        <div className="flex items-center space-x-2">
          <ZapIcon className={`h-4 w-4 ${isOptimizationAtLimit ? 'text-red-500' : isOptimizationNearLimit ? 'text-yellow-500' : 'text-blue-500'}`} />
          <span className={`${isOptimizationAtLimit ? 'text-red-600' : 'text-gray-600'}`}>
            {optimizationLimit === -1 ? `${optimizationUsed} lineups` : `${optimizationUsed}/${optimizationLimit}`}
          </span>
        </div>
        
        {/* Simulation Usage */}
        <div className="flex items-center space-x-2">
          <BarChart3Icon className={`h-4 w-4 ${isSimulationAtLimit ? 'text-red-500' : isSimulationNearLimit ? 'text-yellow-500' : 'text-green-500'}`} />
          <span className={`${isSimulationAtLimit ? 'text-red-600' : 'text-gray-600'}`}>
            {simulationLimit === -1 ? `${simulationUsed} sims` : `${simulationUsed}/${simulationLimit}`}
          </span>
        </div>
        
        {/* Warning indicator */}
        {(isOptimizationNearLimit || isSimulationNearLimit) && (
          <AlertTriangleIcon className="h-4 w-4 text-yellow-500" />
        )}
      </div>
    )
  }

  if (variant === 'dashboard') {
    return (
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Monthly Usage</h3>
          <span className="text-sm text-gray-500 capitalize">{user.subscription_tier} Plan</span>
        </div>
        
        <div className="space-y-4">
          {/* Optimization Usage */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center space-x-2">
                <ZapIcon className="h-4 w-4 text-blue-500" />
                <span className="text-sm font-medium">Lineup Optimizations</span>
              </div>
              <span className="text-sm text-gray-600">
                {optimizationLimit === -1 ? `${optimizationUsed} used` : `${optimizationUsed} of ${optimizationLimit}`}
              </span>
            </div>
            {optimizationLimit !== -1 && (
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div 
                  className={`h-2 rounded-full transition-all duration-300 ${
                    isOptimizationAtLimit ? 'bg-red-500' : 
                    isOptimizationNearLimit ? 'bg-yellow-500' : 'bg-blue-500'
                  }`}
                  style={{ width: `${Math.min(optimizationPercentage, 100)}%` }}
                />
              </div>
            )}
          </div>
          
          {/* Simulation Usage */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center space-x-2">
                <BarChart3Icon className="h-4 w-4 text-green-500" />
                <span className="text-sm font-medium">Monte Carlo Simulations</span>
              </div>
              <span className="text-sm text-gray-600">
                {simulationLimit === -1 ? `${simulationUsed} used` : `${simulationUsed} of ${simulationLimit}`}
              </span>
            </div>
            {simulationLimit !== -1 && (
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div 
                  className={`h-2 rounded-full transition-all duration-300 ${
                    isSimulationAtLimit ? 'bg-red-500' : 
                    isSimulationNearLimit ? 'bg-yellow-500' : 'bg-green-500'
                  }`}
                  style={{ width: `${Math.min(simulationPercentage, 100)}%` }}
                />
              </div>
            )}
          </div>
        </div>
        
        {/* Warning Messages */}
        {(isOptimizationNearLimit || isSimulationNearLimit) && (
          <div className="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded-md">
            <div className="flex items-center space-x-2">
              <AlertTriangleIcon className="h-4 w-4 text-yellow-600" />
              <span className="text-sm text-yellow-800">
                You're approaching your monthly limits. Consider upgrading for unlimited access.
              </span>
            </div>
          </div>
        )}
        
        {(isOptimizationAtLimit || isSimulationAtLimit) && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-md">
            <div className="flex items-center space-x-2">
              <AlertTriangleIcon className="h-4 w-4 text-red-600" />
              <span className="text-sm text-red-800">
                You've reached your monthly limit. Upgrade to continue using this feature.
              </span>
            </div>
          </div>
        )}
        
        {showUpgradeButton && user.subscription_tier === 'free' && (
          <button className="mt-4 w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 transition-colors">
            Upgrade to Premium
          </button>
        )}
      </div>
    )
  }

  // Modal variant for limit exceeded
  if (variant === 'modal') {
    return (
      <div className="text-center">
        <div className="mx-auto w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mb-4">
          <AlertTriangleIcon className="h-8 w-8 text-red-600" />
        </div>
        <h3 className="text-lg font-semibold text-gray-900 mb-2">Usage Limit Reached</h3>
        <p className="text-gray-600 mb-6">
          You've used all your monthly {isOptimizationAtLimit ? 'optimizations' : 'simulations'} for your {user.subscription_tier} plan.
        </p>
        <div className="bg-gray-50 rounded-lg p-4 mb-6">
          <div className="text-sm text-gray-600">
            <div className="flex justify-between items-center">
              <span>Current Usage:</span>
              <span className="font-medium">
                {isOptimizationAtLimit ? `${optimizationUsed}/${optimizationLimit} optimizations` : `${simulationUsed}/${simulationLimit} simulations`}
              </span>
            </div>
          </div>
        </div>
        <button className="w-full bg-blue-600 text-white py-3 px-4 rounded-md hover:bg-blue-700 transition-colors font-medium">
          Upgrade to Continue
        </button>
        <p className="text-xs text-gray-500 mt-3">
          Your usage will reset on the 1st of next month
        </p>
      </div>
    )
  }

  return null
}

export default UsageTracker