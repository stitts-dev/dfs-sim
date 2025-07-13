import { memo } from 'react'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  Cell
} from 'recharts'
import { SimulationResult } from '@/types/simulation'
import { formatNumber } from '@/lib/utils'

interface SimulationChartProps {
  result: SimulationResult
  chartType?: 'distribution' | 'percentiles' | 'finish_rates'
}

const SimulationChart = memo<SimulationChartProps>(function SimulationChart({
  result,
  chartType = 'distribution'
}) {
  // Generate normal distribution data for visualization
  const generateDistributionData = () => {
    const points = []
    const mean = result.mean
    const std = result.standard_deviation
    const min = Math.max(0, mean - 4 * std)
    const max = mean + 4 * std
    
    for (let i = 0; i <= 100; i++) {
      const x = min + (max - min) * (i / 100)
      // Approximate normal distribution
      const z = (x - mean) / std
      const y = Math.exp(-0.5 * z * z) / Math.sqrt(2 * Math.PI * std * std)
      points.push({
        points: Math.round(x * 10) / 10,
        probability: y * 1000, // Scale for visibility
        fill: x >= result.percentile_75 ? '#10b981' : x >= result.percentile_25 ? '#3b82f6' : '#6b7280'
      })
    }
    return points
  }

  const getPercentilesData = () => [
    { name: '25th %ile', value: result.percentile_25, fill: '#ef4444' },
    { name: '50th %ile', value: result.median, fill: '#f59e0b' },
    { name: '75th %ile', value: result.percentile_75, fill: '#10b981' },
    { name: '90th %ile', value: result.percentile_90, fill: '#3b82f6' },
    { name: '95th %ile', value: result.percentile_95, fill: '#8b5cf6' },
    { name: '99th %ile', value: result.percentile_99, fill: '#ec4899' }
  ]

  const getFinishRatesData = () => [
    { name: 'Top 1%', rate: result.top_percent_finishes.top_1, fill: '#ec4899' },
    { name: 'Top 10%', rate: result.top_percent_finishes.top_10, fill: '#8b5cf6' },
    { name: 'Top 20%', rate: result.top_percent_finishes.top_20, fill: '#3b82f6' },
    { name: 'Top 50%', rate: result.top_percent_finishes.top_50, fill: '#10b981' },
    { name: 'Cash', rate: result.cash_probability, fill: '#f59e0b' }
  ]

  interface TooltipData {
    name?: string
    value?: number
    rate?: number
    fill?: string
  }

  const CustomTooltip = ({ active, payload, label }: {
    active?: boolean
    payload?: Array<{ payload: TooltipData; value: number }>
    label?: string | number
  }) => {
    if (active && payload && payload.length) {
      const data = payload[0]
      return (
        <div className="glass rounded-lg p-3 shadow-lg border border-white/20">
          <p className="text-sm font-medium text-gray-900 dark:text-white">
            {chartType === 'distribution' ? `${label} points` : data.payload.name}
          </p>
          <p className="text-sm text-gray-600 dark:text-gray-300">
            {chartType === 'distribution' ? 
              `Probability: ${formatNumber(data.value, 3)}` : 
              chartType === 'percentiles' ? 
                `${formatNumber(data.value, 1)} points` :
                `${formatNumber(data.value, 1)}% rate`
            }
          </p>
        </div>
      )
    }
    return null
  }

  const renderDistributionChart = () => (
    <ResponsiveContainer width="100%" height={300}>
      <AreaChart data={generateDistributionData()}>
        <defs>
          <linearGradient id="distributionGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.8}/>
            <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.1}/>
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
        <XAxis 
          dataKey="points" 
          stroke="#6b7280"
          fontSize={12}
          tickFormatter={(value) => formatNumber(value, 0)}
        />
        <YAxis 
          stroke="#6b7280"
          fontSize={12}
          tickFormatter={(value) => formatNumber(value, 2)}
        />
        <Tooltip content={<CustomTooltip />} />
        <Area
          type="monotone"
          dataKey="probability"
          stroke="#3b82f6"
          strokeWidth={2}
          fill="url(#distributionGradient)"
        />
      </AreaChart>
    </ResponsiveContainer>
  )

  const renderPercentilesChart = () => (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={getPercentilesData()}>
        <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
        <XAxis 
          dataKey="name" 
          stroke="#6b7280"
          fontSize={12}
        />
        <YAxis 
          stroke="#6b7280"
          fontSize={12}
          tickFormatter={(value) => formatNumber(value, 0)}
        />
        <Tooltip content={<CustomTooltip />} />
        <Bar dataKey="value" radius={[4, 4, 0, 0]}>
          {getPercentilesData().map((entry, index) => (
            <Cell key={`cell-${index}`} fill={entry.fill} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  )

  const renderFinishRatesChart = () => (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={getFinishRatesData()}>
        <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
        <XAxis 
          dataKey="name" 
          stroke="#6b7280"
          fontSize={12}
        />
        <YAxis 
          stroke="#6b7280"
          fontSize={12}
          tickFormatter={(value) => `${value}%`}
        />
        <Tooltip 
          content={<CustomTooltip />}
          formatter={(value: number) => [`${formatNumber(value, 1)}%`, 'Rate']}
        />
        <Bar dataKey="rate" radius={[4, 4, 0, 0]}>
          {getFinishRatesData().map((entry, index) => (
            <Cell key={`cell-${index}`} fill={entry.fill} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  )

  const getChartTitle = () => {
    switch (chartType) {
      case 'distribution':
        return 'Score Distribution'
      case 'percentiles':
        return 'Score Percentiles'
      case 'finish_rates':
        return 'Finish Rate Analysis'
      default:
        return 'Simulation Results'
    }
  }

  const getChartDescription = () => {
    switch (chartType) {
      case 'distribution':
        return 'Probability distribution of projected scores'
      case 'percentiles':
        return 'Score thresholds at various percentiles'
      case 'finish_rates':
        return 'Probability of finishing in different tiers'
      default:
        return ''
    }
  }

  return (
    <div className="glass rounded-xl p-6 shadow-glow-lg">
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
          {getChartTitle()}
        </h3>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {getChartDescription()}
        </p>
      </div>
      
      <div className="h-80">
        {chartType === 'distribution' && renderDistributionChart()}
        {chartType === 'percentiles' && renderPercentilesChart()}
        {chartType === 'finish_rates' && renderFinishRatesChart()}
      </div>
    </div>
  )
})

export default SimulationChart