import { Fragment, useState, useEffect } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import { XMarkIcon, SparklesIcon, ArrowPathIcon } from '@heroicons/react/24/outline'
import { useAI, useApplyRecommendations } from '@/hooks/useAI'
import { Player } from '@/types/player'
import { Contest } from '@/types/contest'
import { formatCurrency, formatNumber, getPositionColor, cn } from '@/lib/utils'

interface AIAssistantProps {
  contest: Contest | null
  currentLineup: Player[]
  availablePlayers: Player[]
  onAddPlayers: (playerIds: number[]) => void
  remainingBudget: number
  positionsNeeded: string[]
}

export default function AIAssistant({
  contest,
  currentLineup,
  availablePlayers,
  onAddPlayers,
  remainingBudget,
  positionsNeeded,
}: AIAssistantProps) {
  const {
    isAIOpen,
    recommendations,
    isLoadingRecommendations,
    toggleAI,
    getRecommendations,
    resetRecommendations,
  } = useAI()

  const { applyRecommendations } = useApplyRecommendations()
  const [selectedRecommendations, setSelectedRecommendations] = useState<Set<number>>(new Set())
  const [typingIndex, setTypingIndex] = useState(0)

  // Reset typing animation when new recommendations arrive
  useEffect(() => {
    if (recommendations && recommendations.length > 0) {
      setTypingIndex(0)
      const timer = setInterval(() => {
        setTypingIndex(prev => {
          if (prev >= recommendations.length - 1) {
            clearInterval(timer)
            return prev
          }
          return prev + 1
        })
      }, 150)
      return () => clearInterval(timer)
    }
  }, [recommendations])

  const handleGetRecommendations = async () => {
    if (!contest) return

    await getRecommendations(
      {
        contest_id: contest.id,
        contest_type: contest.contest_type === 'gpp' ? 'GPP' : 'Cash',
        sport: contest.sport,
        remaining_budget: remainingBudget,
        current_lineup: currentLineup.map(p => p.id),
        positions_needed: positionsNeeded,
        optimize_for: contest.contest_type === 'gpp' ? 'ceiling' : 'floor',
      },
      contest,
      availablePlayers,
      currentLineup
    )
  }

  const handleApplySelected = () => {
    const selectedRecs = recommendations?.filter(rec => 
      selectedRecommendations.has(rec.player_id)
    ) || []
    
    if (selectedRecs.length > 0) {
      applyRecommendations(selectedRecs, onAddPlayers)
      setSelectedRecommendations(new Set())
      toggleAI()
    }
  }

  const toggleRecommendation = (playerId: number) => {
    const newSelected = new Set(selectedRecommendations)
    if (newSelected.has(playerId)) {
      newSelected.delete(playerId)
    } else {
      newSelected.add(playerId)
    }
    setSelectedRecommendations(newSelected)
  }

  return (
    <>
      {/* Floating AI Button */}
      <button
        onClick={toggleAI}
        className="fixed bottom-6 right-6 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-gradient-to-r from-purple-600 to-blue-600 text-white shadow-lg hover:shadow-xl transform hover:scale-110 transition-all duration-200 animate-float"
      >
        <SparklesIcon className="h-6 w-6" />
        <div className="absolute -top-1 -right-1">
          <span className="flex h-3 w-3">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-purple-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-3 w-3 bg-purple-500"></span>
          </span>
        </div>
      </button>

      {/* AI Assistant Panel */}
      <Transition.Root show={isAIOpen} as={Fragment}>
        <Dialog as="div" className="relative z-50" onClose={toggleAI}>
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" />
          </Transition.Child>

          <div className="fixed inset-0 z-10 overflow-y-auto">
            <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
              <Transition.Child
                as={Fragment}
                enter="ease-out duration-300"
                enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
                enterTo="opacity-100 translate-y-0 sm:scale-100"
                leave="ease-in duration-200"
                leaveFrom="opacity-100 translate-y-0 sm:scale-100"
                leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
              >
                <Dialog.Panel className="relative transform overflow-hidden rounded-2xl bg-white dark:bg-gray-800 px-4 pb-4 pt-5 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-2xl sm:p-6">
                  {/* Header */}
                  <div className="flex items-center justify-between mb-6">
                    <div className="flex items-center gap-3">
                      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-r from-purple-600 to-blue-600">
                        <SparklesIcon className="h-6 w-6 text-white" />
                      </div>
                      <div>
                        <Dialog.Title className="text-xl font-semibold text-gray-900 dark:text-white">
                          AI Assistant
                        </Dialog.Title>
                        <p className="text-sm text-gray-500 dark:text-gray-400">
                          Smart player recommendations powered by AI
                        </p>
                      </div>
                    </div>
                    <button
                      onClick={toggleAI}
                      className="rounded-lg p-2 text-gray-400 hover:text-gray-500 dark:hover:text-gray-300"
                    >
                      <XMarkIcon className="h-5 w-5" />
                    </button>
                  </div>

                  {/* Content */}
                  <div className="space-y-4">
                    {!recommendations && !isLoadingRecommendations && (
                      <div className="text-center py-8">
                        <div className="mx-auto h-24 w-24 rounded-full bg-gradient-to-r from-purple-100 to-blue-100 dark:from-purple-900/20 dark:to-blue-900/20 flex items-center justify-center mb-4">
                          <SparklesIcon className="h-12 w-12 text-purple-600 dark:text-purple-400" />
                        </div>
                        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                          Get AI Recommendations
                        </h3>
                        <p className="text-gray-500 dark:text-gray-400 mb-6 max-w-md mx-auto">
                          Let our AI analyze your lineup and suggest the best players based on your budget and positions needed.
                        </p>
                        <button
                          onClick={handleGetRecommendations}
                          disabled={!contest || positionsNeeded.length === 0}
                          className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-gradient-to-r from-purple-600 to-blue-600 text-white font-medium hover:shadow-lg transform hover:scale-105 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          <SparklesIcon className="h-5 w-5" />
                          Generate Recommendations
                        </button>
                      </div>
                    )}

                    {/* Loading State */}
                    {isLoadingRecommendations && (
                      <div className="text-center py-8">
                        <div className="mx-auto h-24 w-24 rounded-full bg-gradient-to-r from-purple-100 to-blue-100 dark:from-purple-900/20 dark:to-blue-900/20 flex items-center justify-center mb-4 animate-pulse">
                          <ArrowPathIcon className="h-12 w-12 text-purple-600 dark:text-purple-400 animate-spin" />
                        </div>
                        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                          Analyzing Players...
                        </h3>
                        <div className="flex items-center justify-center gap-1">
                          <div className="h-2 w-2 rounded-full bg-purple-600 animate-bounce" style={{ animationDelay: '0ms' }}></div>
                          <div className="h-2 w-2 rounded-full bg-purple-600 animate-bounce" style={{ animationDelay: '150ms' }}></div>
                          <div className="h-2 w-2 rounded-full bg-purple-600 animate-bounce" style={{ animationDelay: '300ms' }}></div>
                        </div>
                      </div>
                    )}

                    {/* Recommendations */}
                    {recommendations && recommendations.length > 0 && (
                      <div className="space-y-4">
                        <div className="flex items-center justify-between">
                          <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                            Recommended Players
                          </h3>
                          <button
                            onClick={() => {
                              resetRecommendations()
                              setSelectedRecommendations(new Set())
                            }}
                            className="text-sm text-purple-600 hover:text-purple-700 dark:text-purple-400 dark:hover:text-purple-300"
                          >
                            Get New Recommendations
                          </button>
                        </div>

                        <div className="space-y-3 max-h-96 overflow-y-auto">
                          {recommendations.slice(0, typingIndex + 1).map((rec, index) => (
                            <div
                              key={rec.player_id}
                              className={cn(
                                "rounded-xl border transition-all duration-300 cursor-pointer",
                                selectedRecommendations.has(rec.player_id)
                                  ? "border-purple-500 bg-purple-50 dark:bg-purple-900/20"
                                  : "border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800/50 hover:border-purple-300 dark:hover:border-purple-600",
                                index === typingIndex && "animate-slideIn"
                              )}
                              onClick={() => toggleRecommendation(rec.player_id)}
                            >
                              <div className="p-4">
                                <div className="flex items-start justify-between">
                                  <div className="flex items-start gap-3">
                                    <div className={cn(
                                      'flex h-10 w-10 items-center justify-center rounded-lg text-sm font-bold text-white',
                                      getPositionColor(rec.position, contest?.sport)
                                    )}>
                                      {rec.position}
                                    </div>
                                    <div>
                                      <h4 className="font-medium text-gray-900 dark:text-white">
                                        {rec.player_name}
                                      </h4>
                                      <p className="text-sm text-gray-500 dark:text-gray-400">
                                        {rec.team} • {formatCurrency(rec.salary)}
                                      </p>
                                      <p className="text-sm font-medium text-blue-600 dark:text-blue-400 mt-1">
                                        {formatNumber(rec.projected_points)} pts projected
                                      </p>
                                    </div>
                                  </div>
                                  <div className="text-right">
                                    <div className="flex items-center gap-1">
                                      <span className="text-sm font-medium text-gray-500 dark:text-gray-400">
                                        Confidence
                                      </span>
                                      <span className={cn(
                                        "text-sm font-bold",
                                        rec.confidence >= 0.8 ? "text-green-600 dark:text-green-400" :
                                        rec.confidence >= 0.6 ? "text-yellow-600 dark:text-yellow-400" :
                                        "text-gray-600 dark:text-gray-400"
                                      )}>
                                        {Math.round(rec.confidence * 100)}%
                                      </span>
                                    </div>
                                    <input
                                      type="checkbox"
                                      checked={selectedRecommendations.has(rec.player_id)}
                                      onChange={() => {}}
                                      className="mt-2 h-4 w-4 text-purple-600 focus:ring-purple-500 border-gray-300 rounded"
                                    />
                                  </div>
                                </div>
                                
                                <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
                                  {rec.reasoning}
                                </p>

                                {rec.beginner_tip && (
                                  <div className="mt-3 p-2 rounded-lg bg-blue-50 dark:bg-blue-900/20">
                                    <p className="text-xs text-blue-700 dark:text-blue-300">
                                      💡 Tip: {rec.beginner_tip}
                                    </p>
                                  </div>
                                )}

                                {(rec.stack_with?.length || rec.avoid_with?.length) && (
                                  <div className="mt-3 flex gap-4 text-xs">
                                    {rec.stack_with && rec.stack_with.length > 0 && (
                                      <div>
                                        <span className="text-green-600 dark:text-green-400 font-medium">
                                          Stack with:
                                        </span>{' '}
                                        {rec.stack_with.join(', ')}
                                      </div>
                                    )}
                                    {rec.avoid_with && rec.avoid_with.length > 0 && (
                                      <div>
                                        <span className="text-red-600 dark:text-red-400 font-medium">
                                          Avoid with:
                                        </span>{' '}
                                        {rec.avoid_with.join(', ')}
                                      </div>
                                    )}
                                  </div>
                                )}
                              </div>
                            </div>
                          ))}
                        </div>

                        {/* Actions */}
                        <div className="flex items-center justify-between pt-4 border-t border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-500 dark:text-gray-400">
                            {selectedRecommendations.size} players selected
                          </p>
                          <button
                            onClick={handleApplySelected}
                            disabled={selectedRecommendations.size === 0}
                            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-purple-600 text-white font-medium hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            Apply Selected
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                </Dialog.Panel>
              </Transition.Child>
            </div>
          </div>
        </Dialog>
      </Transition.Root>
    </>
  )
}