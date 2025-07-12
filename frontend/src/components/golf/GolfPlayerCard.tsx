import React from 'react'
import { GolfPlayer } from '@/types/golf'
import { formatCurrency, formatNumber, cn } from '@/lib/utils'

interface GolfPlayerCardProps {
  player: GolfPlayer
  onAdd: (player: GolfPlayer) => void
  isSelected: boolean
  isLocked?: boolean
  isExcluded?: boolean
  showDetails?: boolean
}

export const GolfPlayerCard: React.FC<GolfPlayerCardProps> = ({
  player,
  onAdd,
  isSelected,
  isLocked = false,
  isExcluded = false,
  showDetails = true,
}) => {
  const getCutProbabilityColor = (probability?: number) => {
    if (!probability) return 'text-gray-600'
    if (probability >= 0.8) return 'text-green-600 dark:text-green-400'
    if (probability >= 0.6) return 'text-yellow-600 dark:text-yellow-400'
    return 'text-red-600 dark:text-red-400'
  }

  const getValueColor = (value: number) => {
    if (value >= 6) return 'text-green-600 dark:text-green-400'
    if (value >= 4.5) return 'text-yellow-600 dark:text-yellow-400'
    return 'text-red-600 dark:text-red-400'
  }

  const value = player.projected_points / (player.salary / 1000)

  return (
    <div
      className={cn(
        'border rounded-lg p-4 transition-all duration-200',
        isSelected && 'bg-green-50 dark:bg-green-900/20 border-green-500',
        isLocked && 'bg-blue-50 dark:bg-blue-900/20 border-blue-500',
        isExcluded && 'bg-red-50 dark:bg-red-900/20 border-red-500 opacity-60',
        !isSelected && !isLocked && !isExcluded && 'bg-white dark:bg-gray-800 hover:shadow-md'
      )}
    >
      <div className="flex justify-between items-start mb-3">
        <div className="flex-1">
          <h3 className="font-semibold text-gray-900 dark:text-white">
            {player.name}
          </h3>
          <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
            <span className="flag-icon">{player.team}</span>
            {player.world_rank && (
              <span className="text-xs">WR: #{player.world_rank}</span>
            )}
          </div>
        </div>
        <div className="text-right">
          <p className="font-bold text-gray-900 dark:text-white">
            {formatCurrency(player.salary)}
          </p>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {formatNumber(player.projected_points, 1)} pts
          </p>
        </div>
      </div>

      {showDetails && (
        <>
          <div className="grid grid-cols-3 gap-2 text-xs mb-3">
            <div className="text-center">
              <span className="text-gray-600 dark:text-gray-400 block">Cut %</span>
              <span className={cn('font-semibold', getCutProbabilityColor(player.cut_probability))}>
                {player.cut_probability ? `${(player.cut_probability * 100).toFixed(0)}%` : 'N/A'}
              </span>
            </div>
            <div className="text-center">
              <span className="text-gray-600 dark:text-gray-400 block">Top 10</span>
              <span className="font-semibold">
                {player.top10_probability ? `${(player.top10_probability * 100).toFixed(0)}%` : 'N/A'}
              </span>
            </div>
            <div className="text-center">
              <span className="text-gray-600 dark:text-gray-400 block">Own %</span>
              <span className="font-semibold">
                {formatNumber(player.ownership, 1)}%
              </span>
            </div>
          </div>

          <div className="flex justify-between items-center text-xs mb-3">
            <div>
              <span className="text-gray-600 dark:text-gray-400">Floor: </span>
              <span className="font-semibold">{formatNumber(player.floor_points, 1)}</span>
            </div>
            <div>
              <span className="text-gray-600 dark:text-gray-400">Ceiling: </span>
              <span className="font-semibold">{formatNumber(player.ceiling_points, 1)}</span>
            </div>
            <div>
              <span className="text-gray-600 dark:text-gray-400">Value: </span>
              <span className={cn('font-semibold', getValueColor(value))}>
                {formatNumber(value, 2)}x
              </span>
            </div>
          </div>

          {player.recent_form && (
            <div className="text-xs text-gray-600 dark:text-gray-400 mb-3">
              <span>Recent: </span>
              <span className="font-medium">{player.recent_form}</span>
            </div>
          )}
        </>
      )}

      <div className="flex gap-2">
        <button
          onClick={() => onAdd(player)}
          disabled={isSelected || isExcluded}
          className={cn(
            'flex-1 py-2 px-3 rounded text-sm font-medium transition-colors',
            isSelected
              ? 'bg-gray-100 dark:bg-gray-700 text-gray-500 cursor-not-allowed'
              : isExcluded
              ? 'bg-gray-100 dark:bg-gray-700 text-gray-500 cursor-not-allowed'
              : 'bg-green-600 hover:bg-green-700 text-white'
          )}
        >
          {isSelected ? 'Selected' : isExcluded ? 'Excluded' : 'Add to Lineup'}
        </button>
      </div>
    </div>
  )
}

export default GolfPlayerCard