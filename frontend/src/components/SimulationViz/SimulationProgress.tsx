import { memo } from 'react'
import { formatNumber, cn } from '@/lib/utils'

interface SimulationProgressProps {
  progress: {
    completed: number
    total: number
    percentage: number
    eta_seconds: number
  }
  className?: string
}

const SimulationProgress = memo<SimulationProgressProps>(function SimulationProgress({
  progress,
  className
}) {
  const formatETA = (seconds: number) => {
    if (seconds < 60) {
      return `${Math.round(seconds)}s`
    } else if (seconds < 3600) {
      const minutes = Math.floor(seconds / 60)
      const remainingSeconds = Math.round(seconds % 60)
      return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`
    } else {
      const hours = Math.floor(seconds / 3600)
      const minutes = Math.floor((seconds % 3600) / 60)
      return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`
    }
  }

  const getProgressColor = () => {
    if (progress.percentage < 25) return 'from-red-500 to-orange-500'
    if (progress.percentage < 50) return 'from-orange-500 to-yellow-500'
    if (progress.percentage < 75) return 'from-yellow-500 to-blue-500'
    return 'from-blue-500 to-green-500'
  }

  return (
    <div className={cn('space-y-6', className)}>
      <div className="text-center">
        <div className="text-6xl mb-4 animate-pulse">⚡</div>
        <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
          Running Monte Carlo Simulation
        </h3>
        <p className="text-gray-500 dark:text-gray-400">
          Analyzing {formatNumber(progress.total)} possible outcomes...
        </p>
      </div>

      {/* Progress Circle */}
      <div className="flex justify-center">
        <div className="relative w-32 h-32">
          <svg
            className="w-32 h-32 transform -rotate-90"
            viewBox="0 0 100 100"
          >
            {/* Background circle */}
            <circle
              cx="50"
              cy="50"
              r="45"
              stroke="currentColor"
              strokeWidth="8"
              fill="transparent"
              className="text-gray-200 dark:text-gray-700"
            />
            {/* Progress circle */}
            <circle
              cx="50"
              cy="50"
              r="45"
              stroke="currentColor"
              strokeWidth="8"
              fill="transparent"
              strokeDasharray={`${2 * Math.PI * 45}`}
              strokeDashoffset={`${2 * Math.PI * 45 * (1 - progress.percentage / 100)}`}
              className="text-blue-500 transition-all duration-500 ease-out"
              strokeLinecap="round"
            />
          </svg>
          {/* Percentage text */}
          <div className="absolute inset-0 flex items-center justify-center">
            <span className="text-2xl font-bold text-gray-900 dark:text-white">
              {Math.round(progress.percentage)}%
            </span>
          </div>
        </div>
      </div>

      {/* Progress Bar */}
      <div className="space-y-2">
        <div className="flex justify-between text-sm text-gray-500 dark:text-gray-400">
          <span>Progress</span>
          <span>
            {formatNumber(progress.completed)} / {formatNumber(progress.total)}
          </span>
        </div>
        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3 overflow-hidden">
          <div
            className={cn(
              'h-full bg-gradient-to-r transition-all duration-500 ease-out',
              getProgressColor()
            )}
            style={{ width: `${progress.percentage}%` }}
          />
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-6">
        <div className="text-center">
          <div className="text-2xl font-bold text-gray-900 dark:text-white">
            {formatNumber(progress.completed)}
          </div>
          <div className="text-sm text-gray-500 dark:text-gray-400">
            Simulations Complete
          </div>
        </div>
        <div className="text-center">
          <div className="text-2xl font-bold text-gray-900 dark:text-white">
            {formatETA(progress.eta_seconds)}
          </div>
          <div className="text-sm text-gray-500 dark:text-gray-400">
            Estimated Time Remaining
          </div>
        </div>
      </div>

      {/* Animation indicators */}
      <div className="flex justify-center space-x-2">
        {[...Array(3)].map((_, i) => (
          <div
            key={i}
            className="w-2 h-2 bg-blue-500 rounded-full animate-pulse"
            style={{
              animationDelay: `${i * 0.2}s`,
              animationDuration: '1s'
            }}
          />
        ))}
      </div>

      {/* Tips */}
      <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
        <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-2">
          💡 While you wait...
        </h4>
        <ul className="text-sm text-blue-700 dark:text-blue-300 space-y-1">
          <li>• More simulations = more accurate results</li>
          <li>• This analysis considers player correlations</li>
          <li>• Results include tournament payout structure</li>
        </ul>
      </div>
    </div>
  )
})

export default SimulationProgress