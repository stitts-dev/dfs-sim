import { memo } from 'react'
import { useDraggable } from '@dnd-kit/core'
import { CSS } from '@dnd-kit/utilities'
import { formatCurrency, formatNumber, getPositionColor, cn } from '@/lib/utils'
import { Player } from '@/types/player'
import Tooltip from '@/components/ui/Tooltip'
import PlayerDetailTooltip from '@/components/player/PlayerDetailTooltip'
import PositionTooltip from '@/components/ui/PositionTooltip'
import DFSTermTooltip from '@/components/ui/DFSTermTooltip'
import { usePreferencesStore } from '@/store/preferences'

export interface PlayerCardProps {
  player: Player
  isSelected: boolean
  isLocked: boolean
  isExcluded: boolean
  onToggle: () => void
  onLock: () => void
  onExclude: () => void
  variant?: 'default' | 'compact' | 'minimal'
  showOwnership?: boolean
  showValue?: boolean
  className?: string
}

const PlayerCard = memo<PlayerCardProps>(function PlayerCard({
  player,
  isSelected,
  isLocked,
  isExcluded,
  onToggle,
  onLock,
  onExclude,
  variant = 'default',
  showOwnership = true,
  showValue = true,
  className,
}) {
  const value = player.projected_points / (player.salary / 1000)
  const { beginnerMode, tooltipSettings } = usePreferencesStore()
  
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    isDragging,
  } = useDraggable({
    id: player.id,
    data: {
      player,
    },
    disabled: isExcluded,
  })

  const style = {
    transform: CSS.Translate.toString(transform),
    opacity: isDragging ? 0.5 : 1,
    cursor: isDragging ? 'grabbing' : isExcluded ? 'not-allowed' : 'grab',
  }

  // Determine value class based on points per $1000
  const getValueClass = () => {
    if (value >= 5) return 'value-high'
    if (value >= 4) return 'value-medium'
    return 'value-low'
  }

  const renderCompactCard = () => (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        'border border-gray-200 dark:border-gray-700 p-2 rounded-lg transition-colors duration-150',
        'hover:bg-gray-50 dark:hover:bg-gray-800',
        isSelected && 'bg-blue-50 dark:bg-blue-900/50 border-blue-500',
        isExcluded && 'opacity-50 grayscale',
        isDragging && 'z-50 shadow-xl opacity-90',
        !isExcluded && showValue && getValueClass(),
        className
      )}
      {...attributes}
      {...listeners}
      onClick={onToggle}
      role="button"
      tabIndex={0}
      aria-label={`${player.name} - ${player.position} - ${formatCurrency(player.salary)}`}
      aria-pressed={isSelected}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <PositionTooltip position={player.position} sport={player.sport}>
            <div
              className={cn(
                'flex h-6 w-6 items-center justify-center rounded text-xs font-bold text-white cursor-help',
                getPositionColor(player.position, player.sport)
              )}
            >
              {player.position}
            </div>
          </PositionTooltip>
          <div>
            <div className="text-sm font-medium text-gray-900 dark:text-white truncate max-w-24">
              {player.name}
              {player.is_injured && (
                <span className="ml-1 text-red-500" title={player.injury_status}>
                  🤕
                </span>
              )}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {formatCurrency(player.salary)}
            </div>
          </div>
        </div>
        <div className="text-right">
          <div className="text-sm font-medium text-gray-900 dark:text-white">
            {formatNumber(player.projected_points)}
          </div>
          {!beginnerMode && showValue && (
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {formatNumber(value, 2)}x
            </div>
          )}
        </div>
      </div>
    </div>
  )

  const renderMinimalCard = () => (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        'border-l-4 border-gray-200 dark:border-gray-700 p-2 transition-colors duration-150',
        'hover:bg-gray-50 dark:hover:bg-gray-800',
        isSelected && 'bg-blue-50 dark:bg-blue-900/50 border-blue-500',
        isExcluded && 'opacity-50 grayscale',
        isDragging && 'z-50 shadow-xl opacity-90',
        !isExcluded && showValue && getValueClass(),
        className
      )}
      {...attributes}
      {...listeners}
      onClick={onToggle}
      role="button"
      tabIndex={0}
      aria-label={`${player.name} - ${player.position} - ${formatCurrency(player.salary)}`}
      aria-pressed={isSelected}
    >
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-gray-900 dark:text-white truncate">
          {player.name}
          {player.is_injured && (
            <span className="ml-1 text-red-500" title={player.injury_status}>
              🤕
            </span>
          )}
        </span>
        <span className="text-sm text-gray-500 dark:text-gray-400">
          {formatNumber(player.projected_points)}
        </span>
      </div>
    </div>
  )

  const renderDefaultCard = () => (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        'border-b border-gray-200 dark:border-gray-700 p-3 transition-colors duration-150',
        'hover:bg-gray-50 dark:hover:bg-gray-800',
        isSelected && 'bg-blue-50 dark:bg-blue-900/50 border-blue-500',
        isExcluded && 'opacity-50 grayscale',
        isDragging && 'z-50 shadow-xl opacity-90',
        !isExcluded && showValue && getValueClass(),
        className
      )}
      {...attributes}
      {...listeners}
      role="button"
      tabIndex={0}
      aria-label={`${player.name} - ${player.position} - ${formatCurrency(player.salary)}`}
      aria-pressed={isSelected}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          {/* Position Badge */}
          <PositionTooltip position={player.position} sport={player.sport}>
            <div
              className={cn(
                'flex h-8 w-8 items-center justify-center rounded text-xs font-bold text-white cursor-help',
                getPositionColor(player.position, player.sport)
              )}
            >
              {player.position}
            </div>
          </PositionTooltip>

          {/* Player Info */}
          <div>
            <div className="font-medium text-gray-900 dark:text-white">
              {player.name}
              {player.is_injured && (
                <span className="ml-1 text-red-500" title={player.injury_status}>
                  🤕
                </span>
              )}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {player.team} vs {player.opponent}
            </div>
          </div>
        </div>

        <div className="flex items-center space-x-3">
          {/* Stats */}
          <div className="text-right">
            <DFSTermTooltip term="Salary">
              <div className="text-sm font-medium text-gray-900 dark:text-white cursor-help">
                {formatCurrency(player.salary)}
              </div>
            </DFSTermTooltip>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              <DFSTermTooltip term="Proj Pts">
                <span className="cursor-help">{formatNumber(player.projected_points)} pts</span>
              </DFSTermTooltip>
              {!beginnerMode && showValue && (
                <>
                  {' • '}
                  <DFSTermTooltip term="Value">
                    <span className="cursor-help">{formatNumber(value, 2)}x</span>
                  </DFSTermTooltip>
                </>
              )}
            </div>
          </div>

          {/* Actions */}
          <div className="flex space-x-1">
            <DFSTermTooltip term="Lock">
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  onLock()
                }}
                className={cn(
                  'rounded p-1 text-xs transition-colors',
                  isLocked
                    ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
                    : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                )}
                title="Lock player"
                aria-label={`${isLocked ? 'Unlock' : 'Lock'} ${player.name}`}
              >
                🔒
              </button>
            </DFSTermTooltip>
            <DFSTermTooltip term="Exclude">
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  onExclude()
                }}
                className={cn(
                  'rounded p-1 text-xs transition-colors',
                  isExcluded
                    ? 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
                    : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                )}
                title="Exclude player"
                aria-label={`${isExcluded ? 'Include' : 'Exclude'} ${player.name}`}
              >
                ❌
              </button>
            </DFSTermTooltip>
          </div>
        </div>
      </div>

      {/* Ownership bar - only show in expert mode */}
      {!beginnerMode && showOwnership && player.ownership > 0 && (
        <div className="mt-2 px-3 pb-2">
          <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
            <DFSTermTooltip term="Own%">
              <span className="flex items-center gap-1 cursor-help">
                <span className="text-xs">👥</span>
                Ownership
              </span>
            </DFSTermTooltip>
            <span className="font-medium">{formatNumber(player.ownership)}%</span>
          </div>
          <div className="mt-1 h-1.5 w-full rounded-full bg-gray-200/50 dark:bg-gray-700/50 overflow-hidden">
            <div
              className={cn(
                "h-full rounded-full transition-all duration-500",
                player.ownership > 50 ? 'gradient-danger' :
                player.ownership > 30 ? 'gradient-primary' :
                'gradient-success'
              )}
              style={{ width: `${player.ownership}%` }}
            />
          </div>
        </div>
      )}
    </div>
  )

  const renderCard = () => {
    switch (variant) {
      case 'compact':
        return renderCompactCard()
      case 'minimal':
        return renderMinimalCard()
      default:
        return renderDefaultCard()
    }
  }

  const playerCard = renderCard()

  return tooltipSettings.enabled ? (
    <Tooltip
      content={<PlayerDetailTooltip player={player} showAdvanced={!beginnerMode} sport={player.sport} />}
      placement="right"
      delay={tooltipSettings.delay}
      interactive
      maxWidth={350}
      disabled={isDragging}
    >
      {playerCard}
    </Tooltip>
  ) : (
    playerCard
  )
})

export default PlayerCard