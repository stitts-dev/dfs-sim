import { useDroppable } from '@dnd-kit/core'
import { formatCurrency, formatNumber, cn, getPositionColor } from '@/lib/utils'
import { Player } from '@/types/player'
import { Contest } from '@/types/contest'
import { Lineup } from '@/types/lineup'
import { getPositionRequirements } from '@/lib/lineup-utils'
import AnimatedNumber from '@/components/AnimatedNumber'
import PositionTooltip from '@/components/ui/PositionTooltip'
import DFSTermTooltip from '@/components/ui/DFSTermTooltip'
import HelpIcon from '@/components/ui/HelpIcon'
import { usePreferencesStore } from '@/store/preferences'

interface DroppableSlotProps {
  index: number
  position: string
  player: Player | null
  sport?: string
  onRemove: (playerId: number) => void
}

function DroppableSlot({ index, position, player, sport, onRemove }: DroppableSlotProps) {
  const {
    setNodeRef,
    isOver,
  } = useDroppable({
    id: index.toString(),
  })

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'flex items-center justify-between p-3 transition-all duration-200 rounded-lg',
        'hover:bg-gray-50 dark:hover:bg-gray-800/50',
        isOver && 'bg-blue-100 dark:bg-blue-900/40 shadow-lg ring-2 ring-blue-500',
        !player && 'border-2 border-dashed border-gray-300 dark:border-gray-600',
        player && 'bg-white dark:bg-gray-800 shadow-sm'
      )}
    >
      <div className="flex items-center space-x-3">
        <PositionTooltip position={position}>
          <div className={cn(
            'flex h-8 w-12 items-center justify-center rounded text-xs font-bold text-white cursor-help',
            getPositionColor(position, sport)
          )}>
            {position}
          </div>
        </PositionTooltip>
        
        {player ? (
          <div className="flex-1">
            <p className="font-medium text-gray-900 dark:text-white">
              {player.name}
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {player.team} • {formatCurrency(player.salary)} • {formatNumber(player.projected_points)} pts
            </p>
          </div>
        ) : (
          <p className="text-sm text-gray-400 dark:text-gray-500">
            {isOver ? '🎯 Drop here' : 'Empty slot'}
          </p>
        )}
      </div>
      
      {player && (
        <button
          onClick={() => onRemove(player.id)}
          className="text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 p-1 rounded-lg transition-all duration-200"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  )
}

type DisplayLineup = Lineup | {
  id?: number
  players: Player[]
  total_salary: number
  projected_points: number
  simulated_ceiling?: number
  simulated_floor?: number
  simulated_mean?: number
}

interface LineupBuilderProps {
  contest?: Contest
  lineup: Player[]
  allPlayers: Player[]
  optimizedLineups: DisplayLineup[]
  onLineupChange: (lineup: Player[]) => void
  onSelectLineup: (lineup: DisplayLineup) => void
}

export default function LineupBuilder({
  contest,
  lineup,
  optimizedLineups,
  onLineupChange,
  onSelectLineup,
}: LineupBuilderProps) {
  const { beginnerMode } = usePreferencesStore()
  const totalSalary = lineup.reduce((sum, player) => sum + player.salary, 0)
  const totalProjected = lineup.reduce((sum, player) => sum + player.projected_points, 0)
  const salaryCap = contest?.salary_cap || 50000
  const remainingSalary = salaryCap - totalSalary

  const removePlayer = (playerId: number) => {
    onLineupChange(lineup.filter(p => p.id !== playerId))
  }

  const positionRequirements = getPositionRequirements(contest)

  return (
    <div className="space-y-4 animate-fade-in">
      {/* Beginner mode tips */}
      {beginnerMode && lineup.length === 0 && (
        <div className="glass rounded-xl p-4 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 border border-blue-200 dark:border-blue-700">
          <div className="flex items-start gap-3">
            <span className="text-2xl flex-shrink-0">🎯</span>
            <div>
              <h4 className="font-semibold text-blue-800 dark:text-blue-200">Getting Started</h4>
              <p className="text-sm text-blue-700 dark:text-blue-300 mt-1">
                Drag players from the Player Pool on the left into the position slots below. 
                Try to use all your salary while maximizing projected points!
              </p>
            </div>
          </div>
        </div>
      )}
      
      {beginnerMode && lineup.length > 0 && lineup.length < positionRequirements.length && (
        <div className="glass rounded-xl p-4 bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-900/20 dark:to-orange-900/20 border border-amber-200 dark:border-amber-700">
          <div className="flex items-start gap-3">
            <span className="text-2xl flex-shrink-0">💡</span>
            <div>
              <h4 className="font-semibold text-amber-800 dark:text-amber-200">Keep Going!</h4>
              <p className="text-sm text-amber-700 dark:text-amber-300 mt-1">
                You need {positionRequirements.length - lineup.length} more player{positionRequirements.length - lineup.length > 1 ? 's' : ''}. 
                Look for players with good value (high points per dollar).
              </p>
            </div>
          </div>
        </div>
      )}
      
      {beginnerMode && lineup.length === positionRequirements.length && totalSalary > salaryCap && (
        <div className="glass rounded-xl p-4 bg-gradient-to-r from-red-50 to-pink-50 dark:from-red-900/20 dark:to-pink-900/20 border border-red-200 dark:border-red-700">
          <div className="flex items-start gap-3">
            <span className="text-2xl flex-shrink-0">⚠️</span>
            <div>
              <h4 className="font-semibold text-red-800 dark:text-red-200">Over Budget!</h4>
              <p className="text-sm text-red-700 dark:text-red-300 mt-1">
                You're ${formatNumber(totalSalary - salaryCap)} over the salary cap. 
                Try swapping some expensive players for cheaper alternatives.
              </p>
            </div>
          </div>
        </div>
      )}
      
      {beginnerMode && lineup.length === positionRequirements.length && totalSalary <= salaryCap && (
        <div className="glass rounded-xl p-4 bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20 border border-green-200 dark:border-green-700">
          <div className="flex items-start gap-3">
            <span className="text-2xl flex-shrink-0">✅</span>
            <div>
              <h4 className="font-semibold text-green-800 dark:text-green-200">Great Job!</h4>
              <p className="text-sm text-green-700 dark:text-green-300 mt-1">
                Your lineup is complete and under budget! 
                {remainingSalary > 1000 && `You still have ${formatCurrency(remainingSalary)} to spend if you want to upgrade.`}
              </p>
            </div>
          </div>
        </div>
      )}
      {/* Lineup Summary */}
      <div className="glass rounded-xl p-4 shadow-glow-lg">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
          <span className="text-2xl">📋</span>
          Current Lineup
        </h3>
        
        <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
          <div className="glass rounded-lg p-3 text-center">
            <DFSTermTooltip term="Salary">
              <p className="text-gray-500 dark:text-gray-400 text-xs flex items-center justify-center gap-1 cursor-help">
                Salary Used <HelpIcon size="sm" />
              </p>
            </DFSTermTooltip>
            <p className="text-xl font-bold text-gray-900 dark:text-white mt-1">
              <AnimatedNumber 
                value={totalSalary} 
                formatter={formatCurrency}
                className="transition-colors duration-300"
              />
            </p>
          </div>
          <div className="glass rounded-lg p-3 text-center">
            <p className="text-gray-500 dark:text-gray-400 text-xs">Remaining</p>
            <p className={cn(
              "text-xl font-bold mt-1 transition-colors duration-300",
              remainingSalary < 0 ? 'text-red-500' : 'text-green-500'
            )}>
              <AnimatedNumber 
                value={remainingSalary} 
                formatter={formatCurrency}
              />
            </p>
          </div>
          <div className="glass rounded-lg p-3 text-center">
            <DFSTermTooltip term="Proj Pts">
              <p className="text-gray-500 dark:text-gray-400 text-xs flex items-center justify-center gap-1 cursor-help">
                Projected <HelpIcon size="sm" />
              </p>
            </DFSTermTooltip>
            <p className="text-xl font-bold text-gray-900 dark:text-white mt-1">
              <AnimatedNumber 
                value={totalProjected} 
                formatter={(val) => formatNumber(val)}
                className="text-blue-600 dark:text-blue-400"
              />
            </p>
          </div>
        </div>
        
        {/* Salary Cap Progress Bar */}
        <div className="mt-4">
          <div className="relative h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
            <div 
              className={cn(
                "absolute h-full transition-all duration-500",
                totalSalary > salaryCap ? 'gradient-danger' : 
                totalSalary > salaryCap * 0.9 ? 'gradient-primary' : 
                'gradient-success'
              )}
              style={{ width: `${Math.min((totalSalary / salaryCap) * 100, 100)}%` }}
            />
          </div>
        </div>
      </div>

      {/* Position Slots */}
      <div className="glass rounded-xl shadow-glow-lg">
        <div className="p-4 border-b border-gray-200/20 dark:border-gray-700/20">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center justify-between">
            <span className="flex items-center gap-2">
              <span className="text-2xl">⭐</span>
              Positions
            </span>
            <span className={cn(
              "text-sm font-normal px-3 py-1 rounded-full",
              lineup.length === positionRequirements.length ? 
                "bg-green-500/20 text-green-600 dark:text-green-400" : 
                "bg-gray-500/20 text-gray-600 dark:text-gray-400"
            )}>
              {lineup.length}/{positionRequirements.length}
            </span>
          </h3>
        </div>
        
        <div className="divide-y divide-gray-200 dark:divide-gray-700">
          {positionRequirements.map((position, index) => {
            // Find player in this position
            const player = lineup.find((_, idx) => idx === index) || null
            
            return (
              <DroppableSlot
                key={`${position}-${index}`}
                index={index}
                position={position}
                player={player}
                sport={contest?.sport}
                onRemove={removePlayer}
              />
            )
          })}
        </div>
      </div>

      {/* Optimized Lineups */}
      {optimizedLineups.length > 0 && (
        <div className="glass rounded-xl shadow-glow-lg">
          <div className="p-4 border-b border-gray-200/20 dark:border-gray-700/20">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
              <span className="text-2xl">🏆</span>
              Optimized Lineups
              <span className="text-sm font-normal text-gray-500 dark:text-gray-400">({optimizedLineups.length})</span>
            </h3>
            {beginnerMode && (
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Click any lineup below to use it as your starting point!
              </p>
            )}
          </div>
          
          <div className="max-h-64 overflow-y-auto">
            {optimizedLineups.map((lineup, index) => (
              <div
                key={lineup.id || index}
                className="cursor-pointer border-b border-gray-200 p-3 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700"
                onClick={() => onSelectLineup(lineup)}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <p className="font-medium text-gray-900 dark:text-white">
                      Lineup #{index + 1}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {formatCurrency(lineup.total_salary)} • {formatNumber(lineup.projected_points)} pts
                    </p>
                  </div>
                  <div className="text-sm text-gray-500">
                    →
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}