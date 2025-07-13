import { memo } from 'react'
import { SimulationResult } from '@/types/simulation'
import { formatNumber, cn } from '@/lib/utils'

interface SimulationStatsProps {
  result: SimulationResult
  compact?: boolean
  className?: string
}

const SimulationStats = memo<SimulationStatsProps>(function SimulationStats({
  result,
  compact = false,
  className
}) {
  const getROIColor = (roi: number) => {
    if (roi > 20) return 'text-green-600 dark:text-green-400'
    if (roi > 0) return 'text-green-500 dark:text-green-300'
    if (roi > -20) return 'text-yellow-500 dark:text-yellow-400'
    return 'text-red-500 dark:text-red-400'
  }

  const getWinProbabilityColor = (prob: number) => {
    if (prob > 15) return 'text-green-600 dark:text-green-400'
    if (prob > 10) return 'text-green-500 dark:text-green-300'
    if (prob > 5) return 'text-yellow-500 dark:text-yellow-400'
    return 'text-red-500 dark:text-red-400'
  }

  const StatItem = ({ 
    label, 
    value, 
    className: itemClassName,
    tooltip 
  }: { 
    label: string
    value: string | number
    className?: string
    tooltip?: string
  }) => (
    <div className={cn('text-center', itemClassName)} title={tooltip}>
      <div className="text-2xl font-bold text-gray-900 dark:text-white">
        {value}
      </div>
      <div className="text-sm text-gray-500 dark:text-gray-400">
        {label}
      </div>
    </div>
  )

  if (compact) {
    return (
      <div className={cn('glass rounded-lg p-4 shadow-glow', className)}>
        <div className="grid grid-cols-3 gap-4">
          <StatItem
            label="Avg Score"
            value={formatNumber(result.mean, 1)}
          />
          <StatItem
            label="Win %"
            value={`${formatNumber(result.win_probability, 1)}%`}
            className={getWinProbabilityColor(result.win_probability)}
          />
          <StatItem
            label="ROI"
            value={`${formatNumber(result.roi, 0)}%`}
            className={getROIColor(result.roi)}
          />
        </div>
      </div>
    )
  }

  return (
    <div className={cn('glass rounded-xl p-6 shadow-glow-lg space-y-6', className)}>
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
          Simulation Results
        </h3>
        <div className="text-sm text-gray-500 dark:text-gray-400">
          {formatNumber(result.num_simulations)} simulations
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
        <StatItem
          label="Average Score"
          value={formatNumber(result.mean, 1)}
          tooltip="Expected points based on projections"
        />
        <StatItem
          label="Win Probability"
          value={`${formatNumber(result.win_probability, 1)}%`}
          className={getWinProbabilityColor(result.win_probability)}
          tooltip="Chance of finishing in 1st place"
        />
        <StatItem
          label="Cash Rate"
          value={`${formatNumber(result.cash_probability, 1)}%`}
          className={result.cash_probability > 50 ? 'text-green-500' : 'text-yellow-500'}
          tooltip="Probability of finishing in the money"
        />
        <StatItem
          label="Expected ROI"
          value={`${formatNumber(result.roi, 0)}%`}
          className={getROIColor(result.roi)}
          tooltip="Return on investment based on payout structure"
        />
      </div>

      {/* Score Distribution */}
      <div className="border-t border-gray-200 dark:border-gray-700 pt-6">
        <h4 className="text-md font-medium text-gray-900 dark:text-white mb-4">
          Score Distribution
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="space-y-2">
            <div className="text-sm text-gray-500 dark:text-gray-400">Range</div>
            <div className="text-lg font-medium text-gray-900 dark:text-white">
              {formatNumber(result.min, 1)} - {formatNumber(result.max, 1)}
            </div>
          </div>
          <div className="space-y-2">
            <div className="text-sm text-gray-500 dark:text-gray-400">Median</div>
            <div className="text-lg font-medium text-gray-900 dark:text-white">
              {formatNumber(result.median, 1)}
            </div>
          </div>
          <div className="space-y-2">
            <div className="text-sm text-gray-500 dark:text-gray-400">Std Dev</div>
            <div className="text-lg font-medium text-gray-900 dark:text-white">
              {formatNumber(result.standard_deviation, 1)}
            </div>
          </div>
        </div>
      </div>

      {/* Percentiles */}
      <div className="border-t border-gray-200 dark:border-gray-700 pt-6">
        <h4 className="text-md font-medium text-gray-900 dark:text-white mb-4">
          Score Percentiles
        </h4>
        <div className="grid grid-cols-3 md:grid-cols-6 gap-4">
          <div className="text-center">
            <div className="text-lg font-medium text-gray-900 dark:text-white">
              {formatNumber(result.percentile_25, 1)}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">25th</div>
          </div>
          <div className="text-center">
            <div className="text-lg font-medium text-gray-900 dark:text-white">
              {formatNumber(result.median, 1)}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">50th</div>
          </div>
          <div className="text-center">
            <div className="text-lg font-medium text-gray-900 dark:text-white">
              {formatNumber(result.percentile_75, 1)}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">75th</div>
          </div>
          <div className="text-center">
            <div className="text-lg font-medium text-gray-900 dark:text-white">
              {formatNumber(result.percentile_90, 1)}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">90th</div>
          </div>
          <div className="text-center">
            <div className="text-lg font-medium text-gray-900 dark:text-white">
              {formatNumber(result.percentile_95, 1)}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">95th</div>
          </div>
          <div className="text-center">
            <div className="text-lg font-medium text-gray-900 dark:text-white">
              {formatNumber(result.percentile_99, 1)}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">99th</div>
          </div>
        </div>
      </div>

      {/* Finish Rates */}
      <div className="border-t border-gray-200 dark:border-gray-700 pt-6">
        <h4 className="text-md font-medium text-gray-900 dark:text-white mb-4">
          Finish Rate Analysis
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-gradient-to-r from-purple-500/10 to-pink-500/10 border border-purple-500/20 rounded-lg p-3">
            <div className="text-center">
              <div className="text-lg font-bold text-purple-600 dark:text-purple-400">
                {formatNumber(result.top_percent_finishes.top_1, 1)}%
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400">Top 1%</div>
            </div>
          </div>
          <div className="bg-gradient-to-r from-blue-500/10 to-indigo-500/10 border border-blue-500/20 rounded-lg p-3">
            <div className="text-center">
              <div className="text-lg font-bold text-blue-600 dark:text-blue-400">
                {formatNumber(result.top_percent_finishes.top_10, 1)}%
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400">Top 10%</div>
            </div>
          </div>
          <div className="bg-gradient-to-r from-green-500/10 to-emerald-500/10 border border-green-500/20 rounded-lg p-3">
            <div className="text-center">
              <div className="text-lg font-bold text-green-600 dark:text-green-400">
                {formatNumber(result.top_percent_finishes.top_20, 1)}%
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400">Top 20%</div>
            </div>
          </div>
          <div className="bg-gradient-to-r from-yellow-500/10 to-orange-500/10 border border-yellow-500/20 rounded-lg p-3">
            <div className="text-center">
              <div className="text-lg font-bold text-yellow-600 dark:text-yellow-400">
                {formatNumber(result.top_percent_finishes.top_50, 1)}%
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400">Top 50%</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
})

export default SimulationStats