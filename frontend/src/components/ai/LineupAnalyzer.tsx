import { useEffect, useState } from 'react'
import { 
  ChartBarIcon, 
  ExclamationTriangleIcon,
  LightBulbIcon,
  ArrowTrendingUpIcon,
  SparklesIcon,
  InformationCircleIcon,
  ArrowPathIcon
} from '@heroicons/react/24/outline'
import { useLineupAnalysis } from '@/hooks/useAI'
import { Contest } from '@/types/contest'
import { Lineup } from '@/types/lineup'
import { cn } from '@/lib/utils'
import { getPositionRequirements } from '@/lib/lineup-utils'

interface LineupAnalyzerProps {
  lineup: Lineup | null
  contest: Contest | null
  className?: string
}

export default function LineupAnalyzer({ lineup, contest, className }: LineupAnalyzerProps) {
  const { analysis, isAnalyzing, analyzeLineup } = useLineupAnalysis()
  const [hasAnalyzed, setHasAnalyzed] = useState(false)

  // Check if lineup is complete (all positions filled)
  const isLineupComplete = lineup && contest && lineup.players.length === getPositionRequirements(contest).length
  
  // Check if lineup is saved (has a real ID > 0)
  const isLineupSaved = lineup && lineup.id && lineup.id > 0 && lineup.is_submitted !== undefined

  const handleAnalyze = async () => {
    if (lineup && contest && isLineupSaved) {
      await analyzeLineup(lineup.players, contest, lineup.id)
      setHasAnalyzed(true)
    }
  }

  // Reset analyzed state when lineup changes
  useEffect(() => {
    setHasAnalyzed(false)
  }, [lineup?.id])

  if (!lineup || !contest) {
    return null
  }

  return (
    <div className={cn("rounded-xl glass p-6 space-y-6", className)}>
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
          <ChartBarIcon className="h-5 w-5 text-purple-600 dark:text-purple-400" />
          Lineup Analysis
        </h3>
        <div className="flex items-center gap-2">
          <SparklesIcon className="h-4 w-4 text-purple-600 dark:text-purple-400" />
          <span className="text-xs text-gray-500 dark:text-gray-400">AI Powered</span>
        </div>
      </div>

      {/* Show requirements if lineup is not ready for analysis */}
      {!isLineupSaved && (
        <div className="p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700">
          <div className="flex items-start gap-3">
            <InformationCircleIcon className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5" />
            <div className="space-y-2">
              <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                Lineup Analysis Requirements
              </p>
              <ul className="text-sm text-blue-800 dark:text-blue-200 space-y-1">
                {!isLineupComplete && (
                  <li className="flex items-center gap-2">
                    <span className="text-blue-500">•</span>
                    Complete your lineup by filling all positions
                  </li>
                )}
                <li className="flex items-center gap-2">
                  <span className="text-blue-500">•</span>
                  Save your lineup to enable AI analysis
                </li>
              </ul>
            </div>
          </div>
        </div>
      )}

      {/* Show analyze button for saved lineups that haven't been analyzed */}
      {isLineupSaved && !hasAnalyzed && !analysis && (
        <div className="text-center py-8">
          <button
            onClick={handleAnalyze}
            disabled={isAnalyzing}
            className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-gradient-to-r from-purple-600 to-blue-600 text-white font-medium hover:shadow-lg transform hover:scale-105 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isAnalyzing ? (
              <>
                <ArrowTrendingUpIcon className="h-5 w-5 animate-spin" />
                Analyzing...
              </>
            ) : (
              <>
                <ChartBarIcon className="h-5 w-5" />
                Analyze This Lineup
              </>
            )}
          </button>
        </div>
      )}

      {isAnalyzing && (
        <div className="space-y-4">
          <div className="animate-pulse">
            <div className="h-20 bg-gray-200 dark:bg-gray-700 rounded-lg mb-4"></div>
            <div className="space-y-2">
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/2"></div>
            </div>
          </div>
        </div>
      )}

      {analysis && !isAnalyzing && (
        <div className="space-y-6">
          {/* Re-analyze button */}
          {isLineupSaved && (
            <div className="flex justify-end">
              <button
                onClick={handleAnalyze}
                className="text-sm text-purple-600 hover:text-purple-700 dark:text-purple-400 dark:hover:text-purple-300 flex items-center gap-1"
              >
                <ArrowPathIcon className="h-4 w-4" />
                Re-analyze
              </button>
            </div>
          )}
          
          {/* Overall Score */}
          <div className="text-center">
            <div className="relative mx-auto h-32 w-32">
              <svg className="transform -rotate-90 h-32 w-32">
                <circle
                  cx="64"
                  cy="64"
                  r="56"
                  stroke="currentColor"
                  strokeWidth="8"
                  fill="none"
                  className="text-gray-200 dark:text-gray-700"
                />
                <circle
                  cx="64"
                  cy="64"
                  r="56"
                  stroke="currentColor"
                  strokeWidth="8"
                  fill="none"
                  strokeDasharray={351.86}
                  strokeDashoffset={351.86 * (1 - analysis.score / 10)}
                  className={cn(
                    "transition-all duration-1000 ease-out",
                    analysis.score >= 8 ? "text-green-500" :
                    analysis.score >= 6 ? "text-yellow-500" :
                    "text-red-500"
                  )}
                />
              </svg>
              <div className="absolute inset-0 flex items-center justify-center">
                <div>
                  <p className="text-3xl font-bold text-gray-900 dark:text-white">
                    {analysis.score}/10
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">Score</p>
                </div>
              </div>
            </div>
          </div>

          {/* Strengths */}
          {analysis.strengths.length > 0 && (
            <div>
              <h4 className="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white mb-3">
                <ArrowTrendingUpIcon className="h-4 w-4 text-green-600 dark:text-green-400" />
                Strengths
              </h4>
              <ul className="space-y-2">
                {analysis.strengths.map((strength, index) => (
                  <li
                    key={index}
                    className="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-300"
                  >
                    <span className="text-green-500 mt-0.5">✓</span>
                    {strength}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Weaknesses */}
          {analysis.weaknesses.length > 0 && (
            <div>
              <h4 className="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white mb-3">
                <ExclamationTriangleIcon className="h-4 w-4 text-yellow-600 dark:text-yellow-400" />
                Areas for Improvement
              </h4>
              <ul className="space-y-2">
                {analysis.weaknesses.map((weakness, index) => (
                  <li
                    key={index}
                    className="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-300"
                  >
                    <span className="text-yellow-500 mt-0.5">!</span>
                    {weakness}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Suggestions */}
          {analysis.suggestions.length > 0 && (
            <div>
              <h4 className="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white mb-3">
                <LightBulbIcon className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                Suggestions
              </h4>
              <ul className="space-y-2">
                {analysis.suggestions.map((suggestion, index) => (
                  <li
                    key={index}
                    className="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-300"
                  >
                    <span className="text-blue-500 mt-0.5">→</span>
                    {suggestion}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  )
}