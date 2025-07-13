import { SimulationResult } from '@/types/simulation'
import SimulationChart from './SimulationChart'
import SimulationStats from './SimulationStats'
import SimulationProgress from './SimulationProgress'
import { cn } from '@/lib/utils'

export interface SimulationVizProps {
  result?: SimulationResult
  progress?: {
    completed: number
    total: number
    percentage: number
    eta_seconds: number
  }
  isRunning?: boolean
  className?: string
}

export default function SimulationViz({
  result,
  progress,
  isRunning = false,
  className
}: SimulationVizProps) {
  if (isRunning && progress) {
    return (
      <div className={cn('glass rounded-xl p-6 shadow-glow-lg', className)}>
        <SimulationProgress progress={progress} />
      </div>
    )
  }

  if (!result) {
    return (
      <div className={cn('glass rounded-xl p-6 shadow-glow-lg', className)}>
        <div className="text-center text-gray-500 dark:text-gray-400">
          <div className="text-6xl mb-4">📊</div>
          <h3 className="text-lg font-semibold mb-2">No Simulation Results</h3>
          <p className="text-sm">Run a simulation to see detailed analysis and projections</p>
        </div>
      </div>
    )
  }

  return (
    <div className={cn('space-y-6', className)}>
      <SimulationStats result={result} />
      <SimulationChart result={result} />
    </div>
  )
}

export { SimulationChart, SimulationStats, SimulationProgress }