import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery } from 'react-query'
import toast from 'react-hot-toast'
import {
  DndContext,
  DragEndEvent,
  DragOverlay,
  MouseSensor,
  TouchSensor,
  useSensor,
  useSensors,
  closestCenter,
} from '@dnd-kit/core'
import PlayerPool from '@/components/PlayerPool'
import LineupBuilder from '@/components/LineupBuilder'
import OptimizerControls from '@/components/OptimizerControls'
import AIAssistant from '@/components/ai/AIAssistant'
import LineupAnalyzer from '@/components/ai/LineupAnalyzer'
import { getContest, getPlayers, optimizeLineups, OptimizeConfigWithContext } from '@/services/api'
import { Player } from '@/types/player'
import { Lineup } from '@/types/lineup'
import { OptimizeConfig } from '@/types/optimizer'
import { getPositionRequirements } from '@/lib/lineup-utils'
import { formatCurrency, formatNumber, cn, getPositionColor } from '@/lib/utils'

export default function Optimizer() {
  const [searchParams] = useSearchParams()
  const contestId = parseInt(searchParams.get('contest') || '0')
  
  const [currentLineup, setCurrentLineup] = useState<Player[]>([])
  const [optimizedLineups, setOptimizedLineups] = useState<Lineup[]>([])
  const [isOptimizing, setIsOptimizing] = useState(false)
  const [selectedPlayers, setSelectedPlayers] = useState<Set<number>>(new Set())
  const [lockedPlayers, setLockedPlayers] = useState<Set<number>>(new Set())
  const [excludedPlayers, setExcludedPlayers] = useState<Set<number>>(new Set())
  const [activeId, setActiveId] = useState<number | null>(null)

  // Configure drag sensors
  const sensors = useSensors(
    useSensor(MouseSensor, {
      activationConstraint: {
        distance: 5,
      },
    }),
    useSensor(TouchSensor, {
      activationConstraint: {
        delay: 100,
        tolerance: 5,
      },
    })
  )

  const { data: contest } = useQuery(
    ['contest', contestId],
    () => getContest(contestId),
    { enabled: !!contestId }
  )

  const { data: players, isLoading: playersLoading } = useQuery(
    ['players', contestId],
    () => getPlayers(contestId),
    { enabled: !!contestId }
  )

  const handleOptimize = async (config: Partial<OptimizeConfig>) => {
    if (!contestId || !contest) {
      toast.error('Please select a contest first')
      return
    }

    if (!players || players.length === 0) {
      toast.error('No players available for optimization')
      return
    }

    setIsOptimizing(true)
    
    try {
      const optimizeConfig: OptimizeConfigWithContext = {
        contest_id: contestId,
        sport: contest.sport,        // Add sport from contest
        platform: contest.platform,  // Add platform from contest
        num_lineups: config.num_lineups || 20,
        min_different_players: config.min_different_players || 3,
        use_correlations: config.use_correlations ?? true,
        correlation_weight: config.correlation_weight || 0.3,
        stacking_rules: config.stacking_rules || [],
        locked_players: Array.from(lockedPlayers),
        excluded_players: Array.from(excludedPlayers),
        min_exposure: {},
        max_exposure: {},
      }

      console.log('Sending optimization request:', optimizeConfig)
      
      const result = await optimizeLineups(optimizeConfig)
      
      console.log('Optimization result:', result)
      
      if (!result?.lineups?.length) {
        console.error('No lineups returned:', result)
        toast.error('No valid lineups generated. Check console for details.')
        return
      }
      
      setOptimizedLineups(result.lineups)
      setCurrentLineup(result.lineups[0].players)
      toast.success(`Generated ${result.lineups.length} optimized lineups!`)
    } catch (error: any) {
      console.error('Optimization failed:', error)
      
      // Provide more specific error messages
      if (error.response?.status === 404) {
        toast.error('Optimization endpoint not found. Please ensure backend is running.')
      } else if (error.response?.status === 400) {
        toast.error(error.response?.data?.message || 'Invalid optimization parameters')
      } else if (error.response?.status === 500) {
        toast.error('Server error during optimization. Please try again.')
      } else if (error.code === 'ECONNREFUSED') {
        toast.error('Cannot connect to backend. Please ensure the server is running.')
      } else {
        toast.error('Optimization failed. Please check your constraints and try again.')
      }
    } finally {
      setIsOptimizing(false)
    }
  }

  const handlePlayerToggle = (player: Player) => {
    if (excludedPlayers.has(player.id)) {
      toast.error('Cannot select excluded player')
      return
    }

    const newSelected = new Set(selectedPlayers)
    if (newSelected.has(player.id)) {
      newSelected.delete(player.id)
    } else {
      newSelected.add(player.id)
    }
    setSelectedPlayers(newSelected)
  }

  const handleLockPlayer = (playerId: number) => {
    const newLocked = new Set(lockedPlayers)
    if (newLocked.has(playerId)) {
      newLocked.delete(playerId)
    } else {
      newLocked.add(playerId)
      // Remove from excluded if locked
      const newExcluded = new Set(excludedPlayers)
      newExcluded.delete(playerId)
      setExcludedPlayers(newExcluded)
    }
    setLockedPlayers(newLocked)
  }

  const handleExcludePlayer = (playerId: number) => {
    const newExcluded = new Set(excludedPlayers)
    if (newExcluded.has(playerId)) {
      newExcluded.delete(playerId)
    } else {
      newExcluded.add(playerId)
      // Remove from locked if excluded
      const newLocked = new Set(lockedPlayers)
      newLocked.delete(playerId)
      setLockedPlayers(newLocked)
      // Remove from current lineup
      setCurrentLineup(currentLineup.filter(p => p.id !== playerId))
    }
    setExcludedPlayers(newExcluded)
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    setActiveId(null)

    if (!over || !contest || !players) return

    const draggedPlayerId = Number(active.id)
    const draggedPlayer = players.find((p: Player) => p.id === draggedPlayerId)
    
    if (!draggedPlayer) return
    
    if (excludedPlayers.has(draggedPlayerId)) {
      toast.error('Cannot add excluded player')
      return
    }

    const droppedSlotIndex = Number(over.id)
    const positionRequirements = getPositionRequirements(contest)

    // Check if this is a valid drop target
    if (isNaN(droppedSlotIndex) || droppedSlotIndex < 0 || droppedSlotIndex >= positionRequirements.length) {
      return
    }

    // Check if player can fill this position
    const requiredPosition = positionRequirements[droppedSlotIndex]
    if (!canPlayerFillPosition(draggedPlayer, requiredPosition, contest.sport)) {
      toast.error(`${draggedPlayer.name} cannot play ${requiredPosition}`)
      return
    }

    // Create a new lineup array with fixed positions
    const newLineup: (Player | null)[] = Array(positionRequirements.length).fill(null)
    
    // Copy existing players to new array, but skip the dragged player if already in lineup
    currentLineup.forEach((player, idx) => {
      if (player.id !== draggedPlayerId && idx < positionRequirements.length) {
        newLineup[idx] = player
      }
    })

    // Get the player being replaced (if any)
    const replacedPlayer = newLineup[droppedSlotIndex]

    // Place the dragged player in the new position
    newLineup[droppedSlotIndex] = draggedPlayer

    // Calculate new salary
    const filledLineup = newLineup.filter((p): p is Player => p !== null)
    const totalSalary = filledLineup.reduce((sum, p) => sum + p.salary, 0)

    if (totalSalary > contest.salary_cap) {
      toast.error(`Over salary cap by ${formatCurrency(totalSalary - contest.salary_cap)}`)
      return
    }

    // Update the lineup
    setCurrentLineup(filledLineup)
    
    if (replacedPlayer) {
      toast.success(`Replaced ${replacedPlayer.name} with ${draggedPlayer.name}`)
    } else {
      toast.success(`Added ${draggedPlayer.name} to ${requiredPosition}`)
    }
  }

  const handleDragStart = (event: any) => {
    setActiveId(Number(event.active.id))
  }

  // Helper to import position compatibility check
  const canPlayerFillPosition = (player: Player, position: string, sport: string): boolean => {
    const positionMap: Record<string, Record<string, string[]>> = {
      nfl: {
        QB: ['QB'],
        RB: ['RB'],
        WR: ['WR'],
        TE: ['TE'],
        FLEX: ['RB', 'WR', 'TE'],
        DST: ['DST', 'D/ST'],
      },
      nba: {
        PG: ['PG'],
        SG: ['SG'],
        SF: ['SF'],
        PF: ['PF'],
        C: ['C'],
        G: ['PG', 'SG'],
        F: ['SF', 'PF'],
        UTIL: ['PG', 'SG', 'SF', 'PF', 'C'],
      },
      mlb: {
        P: ['P', 'SP', 'RP'],
        C: ['C'],
        '1B': ['1B'],
        '2B': ['2B'],
        '3B': ['3B'],
        SS: ['SS'],
        OF: ['OF', 'LF', 'CF', 'RF'],
      },
      nhl: {
        C: ['C'],
        W: ['W', 'LW', 'RW'],
        D: ['D'],
        G: ['G'],
        UTIL: ['C', 'W', 'LW', 'RW', 'D'],
      },
    }

    const sportMap = positionMap[sport]
    if (!sportMap) return false
    const allowedPositions = sportMap[position]
    if (!allowedPositions) return false
    return allowedPositions.includes(player.position)
  }

  if (!contestId) {
    return (
      <div className="flex h-96 items-center justify-center">
        <p className="text-gray-500 dark:text-gray-400">
          Please select a contest from the dashboard
        </p>
      </div>
    )
  }

  return (
    <DndContext
      sensors={sensors}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      collisionDetection={closestCenter}
    >
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        {/* Fixed Header */}
        <div className="sticky top-0 z-10 p-6 pb-4 bg-white/80 dark:bg-gray-900/80 backdrop-blur-sm border-b border-gray-200 dark:border-gray-800">
          <h2 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center gap-3">
            <span className="text-4xl">🏆</span>
            {contest?.name || 'Lineup Optimizer'}
          </h2>
          <div className="mt-2 flex items-center gap-4 text-sm">
            {contest && (
              <>
                <span className="px-3 py-1 rounded-full glass text-gray-700 dark:text-gray-300">
                  {contest.sport.toUpperCase()}
                </span>
                <span className="px-3 py-1 rounded-full glass text-gray-700 dark:text-gray-300">
                  {contest.platform}
                </span>
                <span className="px-3 py-1 rounded-full glass font-semibold text-green-600 dark:text-green-400">
                  ${contest.salary_cap.toLocaleString()} Cap
                </span>
              </>
            )}
          </div>
        </div>

        {/* Main Content */}
        <div className="p-4 sm:p-6">
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
            {/* Player Pool - Sticky sidebar on desktop */}
            <div className="lg:col-span-4 order-2 lg:order-1">
              <div className="lg:sticky lg:top-[120px] lg:h-[calc(100vh-140px)]">
                <PlayerPool
                  players={players || []}
                  loading={playersLoading}
                  selectedPlayers={selectedPlayers}
                  lockedPlayers={lockedPlayers}
                  excludedPlayers={excludedPlayers}
                  onPlayerToggle={handlePlayerToggle}
                  onLockPlayer={handleLockPlayer}
                  onExcludePlayer={handleExcludePlayer}
                />
              </div>
            </div>

            {/* Main Content Area */}
            <div className="lg:col-span-8 space-y-6 order-1 lg:order-2">
              {/* Optimizer Controls */}
              <OptimizerControls
                contest={contest}
                onOptimize={handleOptimize}
                isOptimizing={isOptimizing}
                lockedCount={lockedPlayers.size}
                excludedCount={excludedPlayers.size}
              />

              {/* Lineup Builder */}
              <LineupBuilder
                contest={contest}
                lineup={currentLineup}
                allPlayers={players || []}
                optimizedLineups={optimizedLineups}
                onLineupChange={setCurrentLineup}
                onSelectLineup={(lineup: Lineup) => setCurrentLineup(lineup.players)}
              />

              {/* AI Analysis - Only show for saved lineups from optimizedLineups */}
              <LineupAnalyzer
                lineup={optimizedLineups.length > 0 && currentLineup.length > 0 ? 
                  optimizedLineups.find(l => 
                    l.players.length === currentLineup.length &&
                    l.players.every(p => currentLineup.some(cp => cp.id === p.id))
                  ) || null
                  : null
                }
                contest={contest}
              />
            </div>
          </div>
        </div>
      </div>
    
    {/* Drag Overlay */}
    <DragOverlay dropAnimation={null}>
      {activeId ? (
        <div className="glass rounded-xl p-4 shadow-2xl cursor-grabbing">
          {(() => {
            const player = players?.find((p: Player) => p.id === activeId)
            if (!player) return null
            
            return (
              <div className="flex items-center space-x-3">
                <div className={cn(
                  'flex h-10 w-10 items-center justify-center rounded-lg text-sm font-bold text-white shadow-lg',
                  getPositionColor(player.position)
                )}>
                  {player.position}
                </div>
                <div>
                  <p className="font-semibold text-gray-900 dark:text-white">
                    {player.name}
                  </p>
                  <p className="text-sm text-gray-600 dark:text-gray-300">
                    {player.team} • {formatCurrency(player.salary)}
                  </p>
                  <p className="text-xs text-blue-600 dark:text-blue-400 font-semibold">
                    {formatNumber(player.projected_points)} pts
                  </p>
                </div>
              </div>
            )
          })()}
        </div>
      ) : null}
    </DragOverlay>
    
    {/* AI Assistant - Fixed position */}
    <AIAssistant
      contest={contest}
      currentLineup={currentLineup}
      availablePlayers={players || []}
      onAddPlayers={(playerIds) => {
        const playersToAdd = players?.filter((p: Player) => playerIds.includes(p.id)) || []
        const newLineup = [...currentLineup]
        const positionRequirements = getPositionRequirements(contest)
        
        // Try to add players to appropriate empty slots
        playersToAdd.forEach((player: Player) => {
          for (let i = 0; i < positionRequirements.length; i++) {
            if (!newLineup[i] && canPlayerFillPosition(player, positionRequirements[i], contest?.sport || '')) {
              newLineup[i] = player
              break
            }
          }
        })
        
        setCurrentLineup(newLineup.filter((p): p is Player => p !== null))
      }}
      remainingBudget={(contest?.salary_cap || 50000) - currentLineup.reduce((sum, p) => sum + p.salary, 0)}
      positionsNeeded={(() => {
        const positions = getPositionRequirements(contest)
        const filled = currentLineup.map((_, i) => i)
        return positions.filter((_pos, idx) => !filled.includes(idx))
      })()}
    />
    </DndContext>
  )
}