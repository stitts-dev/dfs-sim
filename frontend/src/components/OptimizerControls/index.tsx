import { useState } from 'react'
import { cn } from '@/lib/utils'
import { Contest } from '@/types/contest'
import { OptimizeConfig, StackingRule } from '@/types/optimizer'
import DFSTermTooltip from '@/components/ui/DFSTermTooltip'
import HelpIcon from '@/components/ui/HelpIcon'
import { usePreferencesStore } from '@/store/preferences'

interface OptimizerControlsProps {
  contest?: Contest
  onOptimize: (config: Partial<OptimizeConfig>) => void
  isOptimizing: boolean
  lockedCount: number
  excludedCount: number
}

export default function OptimizerControls({
  contest,
  onOptimize,
  isOptimizing,
  lockedCount,
  excludedCount,
}: OptimizerControlsProps) {
  const { beginnerMode } = usePreferencesStore()
  const [numLineups, setNumLineups] = useState(beginnerMode ? 5 : 20)
  const [minDifferentPlayers, setMinDifferentPlayers] = useState(3)
  const [useCorrelations, setUseCorrelations] = useState(true)
  const [correlationWeight, setCorrelationWeight] = useState(0.3)
  const [stackingRules, setStackingRules] = useState<StackingRule[]>([])
  const [showAdvanced, setShowAdvanced] = useState(false)

  const handleOptimize = () => {
    onOptimize({
      num_lineups: numLineups,
      min_different_players: minDifferentPlayers,
      use_correlations: useCorrelations,
      correlation_weight: correlationWeight,
      stacking_rules: stackingRules,
    })
  }

  const addStackingRule = () => {
    setStackingRules([
      ...stackingRules,
      {
        type: 'game',
        min_players: 2,
        max_players: 4,
      },
    ])
  }

  const updateStackingRule = (index: number, updates: Partial<StackingRule>) => {
    const newRules = [...stackingRules]
    newRules[index] = { ...newRules[index], ...updates }
    setStackingRules(newRules)
  }

  const removeStackingRule = (index: number) => {
    setStackingRules(stackingRules.filter((_, i) => i !== index))
  }

  return (
    <div className="glass rounded-xl p-6 shadow-glow-lg animate-fade-in">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
        <span className="text-2xl">⚙️</span>
        Optimizer Settings
      </h3>

      {beginnerMode && (
        <div className="mt-4 p-4 rounded-lg bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-700">
          <p className="text-sm text-blue-800 dark:text-blue-200">
            💡 <strong>Beginner Tip:</strong> Start with 5 lineups and let the optimizer find the best player combinations for you!
          </p>
        </div>
      )}

      <div className="mt-6 space-y-4">
        {/* Basic Settings */}
        <div className={cn(
          "grid gap-4",
          beginnerMode ? "grid-cols-1" : "grid-cols-2"
        )}>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 flex items-center gap-1">
              Number of Lineups
              <DFSTermTooltip term="GPP">
                <span className="cursor-help">
                  <HelpIcon size="sm" />
                </span>
              </DFSTermTooltip>
            </label>
            <input
              type="number"
              min="1"
              max="150"
              value={numLineups}
              onChange={(e) => setNumLineups(parseInt(e.target.value) || 1)}
              className="mt-1 w-full rounded-lg glass px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500/50 transition-all duration-200 border-0"
            />
          </div>

          {!beginnerMode && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                <span className="flex items-center gap-1">
                  Min Different Players
                  <DFSTermTooltip term="GPP">
                    <span className="cursor-help" title="Ensures lineup diversity by requiring a minimum number of different players between lineups">
                      <HelpIcon size="sm" />
                    </span>
                  </DFSTermTooltip>
                </span>
              </label>
              <input
                type="number"
                min="1"
                max="9"
                value={minDifferentPlayers}
                onChange={(e) => setMinDifferentPlayers(parseInt(e.target.value) || 1)}
                className="mt-1 w-full rounded-lg glass px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500/50 transition-all duration-200 border-0"
              />
            </div>
          )}
        </div>

        {/* Correlations - only show in expert mode */}
        {!beginnerMode && (
          <div>
            <div className="flex items-center justify-between">
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={useCorrelations}
                  onChange={(e) => setUseCorrelations(e.target.checked)}
                  className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300 flex items-center gap-1">
                  Use Correlations
                  <DFSTermTooltip term="Correlation">
                    <span className="cursor-help">
                      <HelpIcon size="sm" />
                    </span>
                  </DFSTermTooltip>
                </span>
              </label>
              
              {useCorrelations && (
                <div className="flex items-center space-x-2">
                  <label className="text-sm text-gray-500 dark:text-gray-400">
                    Weight:
                  </label>
                  <input
                    type="range"
                    min="0"
                    max="1"
                    step="0.1"
                    value={correlationWeight}
                    onChange={(e) => setCorrelationWeight(parseFloat(e.target.value))}
                    className="w-24"
                  />
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {correlationWeight.toFixed(1)}
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Player Constraints Info */}
        <div className="flex justify-between rounded-lg glass p-3 text-sm">
          <div className="flex items-center gap-2">
            <span className="text-gray-500 dark:text-gray-400">🔒 Locked:</span>
            <span className="font-semibold text-gray-900 dark:text-white bg-green-500/20 px-2 py-0.5 rounded-full">{lockedCount}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-gray-500 dark:text-gray-400">❌ Excluded:</span>
            <span className="font-semibold text-gray-900 dark:text-white bg-red-500/20 px-2 py-0.5 rounded-full">{excludedCount}</span>
          </div>
        </div>

        {/* Advanced Settings - only show in expert mode */}
        {!beginnerMode && (
          <div>
            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="flex items-center text-sm font-medium text-blue-600 hover:text-blue-700"
            >
              {showAdvanced ? '▼' : '▶'} Advanced Settings
            </button>
            
            {showAdvanced && (
            <div className="mt-4 space-y-4">
              {/* Stacking Rules */}
              <div>
                <div className="flex items-center justify-between">
                  <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 flex items-center gap-1">
                    Stacking Rules
                    <DFSTermTooltip term="Stacking">
                      <span className="cursor-help">
                        <HelpIcon size="sm" />
                      </span>
                    </DFSTermTooltip>
                  </h4>
                  <button
                    type="button"
                    onClick={addStackingRule}
                    className="text-sm text-blue-600 hover:text-blue-700"
                  >
                    + Add Rule
                  </button>
                </div>
                
                <div className="mt-2 space-y-2">
                  {stackingRules.map((rule, index) => (
                    <div key={index} className="flex items-center space-x-2 rounded-lg border border-gray-200 p-2 dark:border-gray-600">
                      <select
                        value={rule.type}
                        onChange={(e) => updateStackingRule(index, { type: e.target.value as 'team' | 'game' | 'mini' | 'qb_stack' })}
                        className="rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-700"
                      >
                        <option value="game" title="Stack players from the same game (e.g., QB + pass catchers + opposing pass catchers)">Game Stack</option>
                        <option value="team" title="Stack players from the same team (e.g., QB + WR + TE)">Team Stack</option>
                        <option value="mini" title="Small correlation plays (e.g., RB + DEF)">Mini Stack</option>
                      </select>
                      
                      <input
                        type="number"
                        min="1"
                        max="9"
                        value={rule.min_players}
                        onChange={(e) => updateStackingRule(index, { min_players: parseInt(e.target.value) || 1 })}
                        className="w-16 rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-700"
                        placeholder="Min"
                      />
                      
                      <span className="text-gray-500">-</span>
                      
                      <input
                        type="number"
                        min="1"
                        max="9"
                        value={rule.max_players}
                        onChange={(e) => updateStackingRule(index, { max_players: parseInt(e.target.value) || 1 })}
                        className="w-16 rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-700"
                        placeholder="Max"
                      />
                      
                      
                      <button
                        type="button"
                        onClick={() => removeStackingRule(index)}
                        className="text-red-600 hover:text-red-700"
                      >
                        ✕
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
          </div>
        )}

        {/* Optimize Button */}
        <button
          onClick={handleOptimize}
          disabled={isOptimizing || !contest}
          className={cn(
            "w-full rounded-lg py-3 text-sm font-semibold text-white transition-all duration-300",
            "gradient-primary btn-primary shadow-lg hover:shadow-xl transform hover:-translate-y-0.5",
            "disabled:cursor-not-allowed disabled:opacity-50 disabled:transform-none disabled:hover:shadow-lg",
            isOptimizing && "animate-pulse"
          )}
        >
          {isOptimizing ? (
            <span className="flex items-center justify-center gap-2">
              <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              Optimizing...
            </span>
          ) : (
            beginnerMode ? 'Create My Lineups 🏆' : 'Optimize Lineups ✨'
          )}
        </button>
      </div>
    </div>
  )
}