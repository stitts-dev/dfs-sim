import React from 'react'
import Tooltip from './Tooltip'

// DFS terminology explanations
const DFS_TERMS = {
  'Proj Pts': {
    term: 'Projected Points',
    description: 'Expected fantasy points based on statistical models and matchup analysis.',
    example: 'A player projected for 45.5 points is expected to score around that amount.',
    emoji: '📊',
  },
  '$/Pt': {
    term: 'Dollars Per Point',
    description: 'Salary divided by projected points. Lower is better - indicates value.',
    example: '$200 per point means you pay $200 in salary for each projected point.',
    emoji: '💰',
  },
  'Own%': {
    term: 'Ownership Percentage',
    description: 'Expected percentage of lineups that will include this player in tournaments.',
    example: '30% ownership means 3 out of 10 lineups will have this player.',
    emoji: '👥',
  },
  'Floor': {
    term: 'Floor Points',
    description: 'Realistic worst-case scenario for fantasy points. Important for cash games.',
    example: 'A 25-point floor means the player rarely scores below 25 points.',
    emoji: '🔻',
  },
  'Ceiling': {
    term: 'Ceiling Points',
    description: 'Realistic best-case scenario. Maximum upside important for tournaments.',
    example: 'A 60-point ceiling means the player could explode for 60+ points.',
    emoji: '🔺',
  },
  'Salary': {
    term: 'Player Salary',
    description: 'Cost to roster a player. Must fit total lineup within salary cap.',
    example: '$8,500 salary uses 17% of a $50,000 cap.',
    emoji: '💵',
  },
  'Value': {
    term: 'Value Rating',
    description: 'Points per $1,000 of salary. 5x or higher indicates good value.',
    example: '5.2x value means 5.2 points expected per $1,000 spent.',
    emoji: '💎',
  },
  'Correlation': {
    term: 'Player Correlation',
    description: 'How closely two players\' scores relate. QB-WR pairs often correlate.',
    example: 'QB and his WR1 have positive correlation - both do well together.',
    emoji: '🔗',
  },
  'Stacking': {
    term: 'Lineup Stacking',
    description: 'Pairing correlated players to maximize upside when they perform well together.',
    example: 'Stacking a QB with 2 pass catchers from the same game.',
    emoji: '📚',
  },
  'GPP': {
    term: 'Guaranteed Prize Pool',
    description: 'Large tournaments with guaranteed payouts. Require high ceiling plays.',
    example: 'A $1M GPP tournament pays out $1M regardless of entries.',
    emoji: '🏆',
  },
  'Cash': {
    term: 'Cash Games',
    description: '50/50s and Double-Ups where ~50% of entries win. Prioritize floor.',
    example: 'In a 50/50, top half of entries double their money.',
    emoji: '💸',
  },
  'Fade': {
    term: 'Fade',
    description: 'Avoiding a popular player to differentiate your lineup.',
    example: 'Fading a 40% owned player to be contrarian in GPPs.',
    emoji: '🚫',
  },
  'Chalk': {
    term: 'Chalk Play',
    description: 'Highly owned, obvious plays. Safe but limits tournament upside.',
    example: 'A $4,000 player projected for 30 points will be chalk.',
    emoji: '✏️',
  },
  'Punt': {
    term: 'Punt Play',
    description: 'Cheap player used to afford expensive players elsewhere.',
    example: 'A $3,000 minimum-priced player who might see playing time.',
    emoji: '🏈',
  },
  'Lock': {
    term: 'Locked Player',
    description: 'Force player into all optimized lineups. Use for must-have players.',
    example: 'Locking your favorite RB ensures he\'s in every lineup.',
    emoji: '🔒',
  },
  'Exclude': {
    term: 'Excluded Player',
    description: 'Prevent player from appearing in any optimized lineups.',
    example: 'Exclude an injured player who\'s still in the player pool.',
    emoji: '❌',
  },
}

interface DFSTermTooltipProps {
  term: keyof typeof DFS_TERMS
  children: React.ReactElement
  className?: string
}

export default function DFSTermTooltip({ term, children, className }: DFSTermTooltipProps) {
  const info = DFS_TERMS[term]

  if (!info) {
    return children
  }

  const tooltipContent = (
    <div className="space-y-2 max-w-xs">
      <div className="flex items-center gap-2">
        <span className="text-xl">{info.emoji}</span>
        <div className="font-semibold text-white">{info.term}</div>
      </div>
      
      <div className="text-sm text-gray-300">
        {info.description}
      </div>
      
      {info.example && (
        <div className="border-t border-gray-700 pt-2">
          <div className="text-xs font-semibold text-gray-400 mb-1">Example:</div>
          <div className="text-xs text-gray-300 italic">
            {info.example}
          </div>
        </div>
      )}
    </div>
  )

  return (
    <Tooltip
      content={tooltipContent}
      placement="top"
      delay={200}
      className={className}
    >
      {children}
    </Tooltip>
  )
}

// Export for use in other components
export { DFS_TERMS }