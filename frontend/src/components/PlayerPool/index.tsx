import { useState, useMemo, memo } from 'react'
import { useDraggable } from '@dnd-kit/core'
import { CSS } from '@dnd-kit/utilities'
import { formatCurrency, formatNumber, getPositionColor, cn } from '@/lib/utils'
import { Player } from '@/types/player'
import Tooltip from '@/components/ui/Tooltip'
import PlayerDetailTooltip from '@/components/player/PlayerDetailTooltip'
import PositionTooltip from '@/components/ui/PositionTooltip'
import DFSTermTooltip from '@/components/ui/DFSTermTooltip'
import HelpIcon from '@/components/ui/HelpIcon'
import { usePreferencesStore } from '@/store/preferences'

interface PlayerPoolProps {
  players: Player[]
  loading: boolean
  selectedPlayers: Set<number>
  lockedPlayers: Set<number>
  excludedPlayers: Set<number>
  onPlayerToggle: (player: Player) => void
  onLockPlayer: (playerId: number) => void
  onExcludePlayer: (playerId: number) => void
}

export default function PlayerPool({
  players,
  loading,
  selectedPlayers,
  lockedPlayers,
  excludedPlayers,
  onPlayerToggle,
  onLockPlayer,
  onExcludePlayer,
}: PlayerPoolProps) {
  const { beginnerMode } = usePreferencesStore()
  const [search, setSearch] = useState('')
  const [positionFilter, setPositionFilter] = useState<string>('all')
  const [teamFilter, setTeamFilter] = useState<string>('all')
  const [sortBy, setSortBy] = useState<'salary' | 'projected' | 'value'>('projected')

  // Get unique positions and teams
  const positions = useMemo(() => {
    const uniquePositions = new Set(players.map(p => p.position))
    return ['all', ...Array.from(uniquePositions).sort()]
  }, [players])

  const teams = useMemo(() => {
    const uniqueTeams = new Set(players.map(p => p.team))
    return ['all', ...Array.from(uniqueTeams).sort()]
  }, [players])

  // Filter and sort players
  const filteredPlayers = useMemo(() => {
    let filtered = players.filter(player => {
      if (search && !player.name.toLowerCase().includes(search.toLowerCase())) {
        return false
      }
      if (positionFilter !== 'all' && player.position !== positionFilter) {
        return false
      }
      if (teamFilter !== 'all' && player.team !== teamFilter) {
        return false
      }
      return true
    })

    // Sort players
    filtered.sort((a, b) => {
      switch (sortBy) {
        case 'salary':
          return b.salary - a.salary
        case 'projected':
          return b.projected_points - a.projected_points
        case 'value':
          return (b.projected_points / (b.salary / 1000)) - (a.projected_points / (a.salary / 1000))
        default:
          return 0
      }
    })

    return filtered
  }, [players, search, positionFilter, teamFilter, sortBy])

  if (loading) {
    return (
      <div className="glass rounded-xl p-6 shadow-glow-lg animate-fade-in">
        <div className="space-y-4">
          <div className="h-10 skeleton rounded-lg"></div>
          <div className="h-10 skeleton rounded-lg w-3/4"></div>
          <div className="flex gap-2">
            <div className="h-8 w-24 skeleton rounded-lg"></div>
            <div className="h-8 w-24 skeleton rounded-lg"></div>
            <div className="h-8 w-24 skeleton rounded-lg"></div>
          </div>
          <div className="space-y-2 mt-4">
            {[...Array(8)].map((_, i) => (
              <div key={i} className="h-20 skeleton rounded-lg" style={{ animationDelay: `${i * 0.1}s` }}></div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="glass rounded-xl shadow-lg h-full flex flex-col">
      <div className="flex-shrink-0 border-b border-gray-200/20 p-4 dark:border-gray-700/20">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <span className="text-2xl">🏀</span>
            Player Pool
          </h3>
          <DFSTermTooltip term="GPP">
            <button className="text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 flex items-center gap-1">
              <HelpIcon size="sm" />
              <span>DFS Help</span>
            </button>
          </DFSTermTooltip>
        </div>
        
        {/* Search */}
        <div className="mt-4 relative">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <svg className="h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
          <input
            type="text"
            placeholder="Search players..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-3 py-2 rounded-lg glass border-0 text-sm focus:ring-2 focus:ring-blue-500/50 transition-all duration-200 placeholder-gray-400"
          />
        </div>

        {/* Filters */}
        <div className="mt-4 flex gap-2 flex-wrap">
          <select
            value={positionFilter}
            onChange={(e) => setPositionFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg text-sm glass hover:bg-white/90 dark:hover:bg-gray-800/90 transition-all duration-200 focus:ring-2 focus:ring-blue-500/50 border-0"
          >
            {positions.map(pos => (
              <option key={pos} value={pos}>
                {pos === 'all' ? '📍 All Positions' : pos}
              </option>
            ))}
          </select>

          <select
            value={teamFilter}
            onChange={(e) => setTeamFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg text-sm glass hover:bg-white/90 dark:hover:bg-gray-800/90 transition-all duration-200 focus:ring-2 focus:ring-blue-500/50 border-0"
          >
            {teams.map(team => (
              <option key={team} value={team}>
                {team === 'all' ? '🏀 All Teams' : team}
              </option>
            ))}
          </select>

          <DFSTermTooltip term="Value">
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as any)}
              className="px-3 py-1.5 rounded-lg text-sm glass hover:bg-white/90 dark:hover:bg-gray-800/90 transition-all duration-200 focus:ring-2 focus:ring-blue-500/50 border-0"
            >
              <option value="projected">⚡ Projected</option>
              <option value="salary">💰 Salary</option>
              {!beginnerMode && <option value="value">💎 Value</option>}
            </select>
          </DFSTermTooltip>
        </div>

        {/* Player count and column headers */}
        <div className="mt-4 flex items-center justify-between text-sm text-gray-500 dark:text-gray-400">
          <div>{filteredPlayers.length} players • {selectedPlayers.size} selected</div>
          <div className="flex items-center gap-4 text-xs">
            <DFSTermTooltip term="Proj Pts">
              <span className="flex items-center gap-1 cursor-help">
                Proj Pts <HelpIcon size="sm" />
              </span>
            </DFSTermTooltip>
            {!beginnerMode && (
              <>
                <DFSTermTooltip term="$/Pt">
                  <span className="flex items-center gap-1 cursor-help">
                    $/Pt <HelpIcon size="sm" />
                  </span>
                </DFSTermTooltip>
                <DFSTermTooltip term="Own%">
                  <span className="flex items-center gap-1 cursor-help">
                    Own% <HelpIcon size="sm" />
                  </span>
                </DFSTermTooltip>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Player List */}
      <div className="flex-1 overflow-y-auto scrollbar-thin min-h-0">
        {filteredPlayers.length === 0 ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400">
            <p className="text-lg">No players found</p>
            <p className="text-sm mt-2">Try adjusting your filters</p>
          </div>
        ) : (
          filteredPlayers.map(player => (
            <PlayerCard
              key={player.id}
              player={player}
              isSelected={selectedPlayers.has(player.id)}
              isLocked={lockedPlayers.has(player.id)}
              isExcluded={excludedPlayers.has(player.id)}
              onToggle={() => onPlayerToggle(player)}
              onLock={() => onLockPlayer(player.id)}
              onExclude={() => onExcludePlayer(player.id)}
            />
          ))
        )}
      </div>
    </div>
  )
}

interface PlayerCardProps {
  player: Player
  isSelected: boolean
  isLocked: boolean
  isExcluded: boolean
  onToggle: () => void
  onLock: () => void
  onExclude: () => void
}

const PlayerCard = memo(function PlayerCard({
  player,
  isSelected,
  isLocked,
  isExcluded,
  onToggle: _onToggle, // Prefix with underscore to indicate it's intentionally unused
  onLock,
  onExclude,
}: PlayerCardProps) {
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

  const playerCard = (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        'border-b border-gray-200 dark:border-gray-700 p-3 transition-colors duration-150',
        'hover:bg-gray-50 dark:hover:bg-gray-800',
        isSelected && 'bg-blue-50 dark:bg-blue-900/50 border-blue-500',
        isExcluded && 'opacity-50 grayscale',
        isDragging && 'z-50 shadow-xl opacity-90',
        !isExcluded && getValueClass()
      )}
      {...attributes}
      {...listeners}
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
              {!beginnerMode && (
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
                  'rounded p-1 text-xs',
                  isLocked
                    ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
                    : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                )}
                title="Lock player"
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
                  'rounded p-1 text-xs',
                  isExcluded
                    ? 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
                    : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                )}
                title="Exclude player"
              >
                ❌
              </button>
            </DFSTermTooltip>
          </div>
        </div>
      </div>

      {/* Ownership bar - only show in expert mode */}
      {!beginnerMode && player.ownership > 0 && (
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