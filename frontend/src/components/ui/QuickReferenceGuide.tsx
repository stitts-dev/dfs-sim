import { useState } from 'react'
import { cn } from '@/lib/utils'
import { POSITION_INFO } from './PositionTooltip'
import { DFS_TERMS } from './DFSTermTooltip'

interface QuickReferenceGuideProps {
  isOpen: boolean
  onClose: () => void
}

export default function QuickReferenceGuide({ isOpen, onClose }: QuickReferenceGuideProps) {
  const [activeTab, setActiveTab] = useState<'positions' | 'terms' | 'strategies'>('terms')

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />
      
      {/* Modal */}
      <div className="relative glass rounded-2xl shadow-glow-xl w-full max-w-4xl max-h-[80vh] overflow-hidden animate-scale-in">
        {/* Header */}
        <div className="border-b border-gray-200/20 p-6 dark:border-gray-700/20">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
              <span className="text-3xl">📚</span>
              DFS Quick Reference Guide
            </h2>
            <button
              onClick={onClose}
              className="rounded-lg p-2 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          
          {/* Tabs */}
          <div className="mt-4 flex space-x-1">
            {(['terms', 'positions', 'strategies'] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={cn(
                  'px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200',
                  activeTab === tab
                    ? 'bg-blue-500 text-white shadow-lg'
                    : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
                )}
              >
                {tab === 'terms' && '📝 Terms'}
                {tab === 'positions' && '🏃 Positions'}
                {tab === 'strategies' && '🎯 Strategies'}
              </button>
            ))}
          </div>
        </div>
        
        {/* Content */}
        <div className="overflow-y-auto max-h-[calc(80vh-200px)] p-6">
          {activeTab === 'terms' && (
            <div className="grid gap-4 md:grid-cols-2">
              {Object.entries(DFS_TERMS).map(([key, info]) => (
                <div key={key} className="glass rounded-lg p-4 hover:shadow-lg transition-all duration-200">
                  <div className="flex items-start gap-3">
                    <span className="text-2xl">{info.emoji}</span>
                    <div className="flex-1">
                      <h3 className="font-semibold text-gray-900 dark:text-white">
                        {info.term}
                      </h3>
                      <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                        {info.description}
                      </p>
                      {info.example && (
                        <p className="mt-2 text-xs text-gray-500 dark:text-gray-500 italic">
                          Example: {info.example}
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
          
          {activeTab === 'positions' && (
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                  🏀 Basketball Positions
                </h3>
                <div className="grid gap-4 md:grid-cols-2">
                  {Object.entries(POSITION_INFO)
                    .filter(([key]) => ['PG', 'SG', 'SF', 'PF', 'C', 'G', 'F', 'UTIL'].includes(key))
                    .map(([key, info]) => (
                      <PositionCard key={key} position={key} info={info} />
                    ))}
                </div>
              </div>
              
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                  🏈 Football Positions
                </h3>
                <div className="grid gap-4 md:grid-cols-2">
                  {Object.entries(POSITION_INFO)
                    .filter(([key]) => ['QB', 'RB', 'WR', 'TE', 'DST', 'K', 'FLEX'].includes(key))
                    .map(([key, info]) => (
                      <PositionCard key={key} position={key} info={info} />
                    ))}
                </div>
              </div>
              
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                  ⚾ Baseball Positions
                </h3>
                <div className="grid gap-4 md:grid-cols-2">
                  {Object.entries(POSITION_INFO)
                    .filter(([key]) => ['P', 'C (Baseball)', '1B', '2B', '3B', 'SS', 'OF'].includes(key))
                    .map(([key, info]) => (
                      <PositionCard key={key} position={key} info={info} />
                    ))}
                </div>
              </div>
            </div>
          )}
          
          {activeTab === 'strategies' && (
            <div className="space-y-6">
              <div className="glass rounded-lg p-6">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                  🏆 Tournament (GPP) Strategy
                </h3>
                <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <li className="flex items-start gap-2">
                    <span className="text-green-500">✓</span>
                    <span>Target high-ceiling players with boom potential</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-green-500">✓</span>
                    <span>Use game stacks to capture correlated scoring</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-green-500">✓</span>
                    <span>Fade chalk plays to differentiate lineups</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-green-500">✓</span>
                    <span>Consider contrarian plays with low ownership</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-green-500">✓</span>
                    <span>Build multiple diverse lineups for better coverage</span>
                  </li>
                </ul>
              </div>
              
              <div className="glass rounded-lg p-6">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                  💸 Cash Game Strategy
                </h3>
                <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500">✓</span>
                    <span>Prioritize high-floor, consistent players</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500">✓</span>
                    <span>Target players with good matchups and high volume</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500">✓</span>
                    <span>Avoid risky plays and focus on safety</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500">✓</span>
                    <span>Don't worry about ownership - play the best plays</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500">✓</span>
                    <span>Build one optimal lineup focused on median projection</span>
                  </li>
                </ul>
              </div>
              
              <div className="glass rounded-lg p-6">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                  🔗 Stacking Guidelines
                </h3>
                <div className="space-y-4 text-sm text-gray-600 dark:text-gray-400">
                  <div>
                    <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-1">Game Stacks</h4>
                    <p>Combine players from both teams in high-scoring games. Example: QB + WR + opposing WR</p>
                  </div>
                  <div>
                    <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-1">Team Stacks</h4>
                    <p>Stack players from the same team. Example: QB + 2 pass catchers from same team</p>
                  </div>
                  <div>
                    <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-1">Mini Stacks</h4>
                    <p>Small correlations like RB + Defense (game script correlation)</p>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

interface PositionInfo {
  name: string
  description: string
  keyStats: string[]
  emoji: string
}

function PositionCard({ position, info }: { position: string; info: PositionInfo }) {
  return (
    <div className="glass rounded-lg p-4 hover:shadow-lg transition-all duration-200">
      <div className="flex items-start gap-3">
        <span className="text-2xl">{info.emoji}</span>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h4 className="font-semibold text-gray-900 dark:text-white">
              {info.name}
            </h4>
            <span className="text-xs px-2 py-0.5 rounded-full bg-blue-500/20 text-blue-600 dark:text-blue-400">
              {position}
            </span>
          </div>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            {info.description}
          </p>
          <div className="mt-2 flex flex-wrap gap-1">
            {info.keyStats.map((stat: string, index: number) => (
              <span
                key={index}
                className="px-2 py-0.5 text-xs rounded-full bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
              >
                {stat}
              </span>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}