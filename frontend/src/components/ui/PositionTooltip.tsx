import React from 'react'
import Tooltip from './Tooltip'

// Position explanations for different sports
const POSITION_INFO = {
  // Basketball positions
  PG: {
    name: 'Point Guard',
    description: 'Primary ball handler and playmaker. Typically scores through assists and three-pointers.',
    keyStats: ['Assists', '3-Pointers', 'Steals'],
    emoji: '🎯',
  },
  SG: {
    name: 'Shooting Guard',
    description: 'Primary perimeter scorer. Focuses on outside shooting and creating own shots.',
    keyStats: ['Points', '3-Pointers', 'Free Throws'],
    emoji: '🏹',
  },
  SF: {
    name: 'Small Forward',
    description: 'Versatile wing player. Balances scoring, rebounding, and defense.',
    keyStats: ['Points', 'Rebounds', 'Defensive Stats'],
    emoji: '⚡',
  },
  PF: {
    name: 'Power Forward',
    description: 'Inside-outside player. Strong rebounder who can also stretch the floor.',
    keyStats: ['Rebounds', 'Points', 'Blocks'],
    emoji: '💪',
  },
  C: {
    name: 'Center',
    description: 'Anchor of the defense. Dominates the paint with rebounds and blocks.',
    keyStats: ['Rebounds', 'Blocks', 'Field Goal %'],
    emoji: '🗼',
  },
  G: {
    name: 'Guard',
    description: 'Can play either guard position (PG or SG). Versatile backcourt player.',
    keyStats: ['Points', 'Assists', '3-Pointers'],
    emoji: '🔄',
  },
  F: {
    name: 'Forward',
    description: 'Can play either forward position (SF or PF). Flexible frontcourt player.',
    keyStats: ['Points', 'Rebounds', 'Versatility'],
    emoji: '🔄',
  },
  UTIL: {
    name: 'Utility',
    description: 'Can be filled by any position. Provides roster flexibility.',
    keyStats: ['Best Available Player'],
    emoji: '🎲',
  },

  // Football positions
  QB: {
    name: 'Quarterback',
    description: 'Team leader who throws passes and calls plays. Most important offensive position.',
    keyStats: ['Passing Yards', 'Touchdowns', 'Completion %'],
    emoji: '🏈',
  },
  RB: {
    name: 'Running Back',
    description: 'Carries the ball and catches passes out of the backfield. Versatile weapon.',
    keyStats: ['Rushing Yards', 'Receptions', 'Touchdowns'],
    emoji: '🏃',
  },
  WR: {
    name: 'Wide Receiver',
    description: 'Primary pass catchers. Run routes and make big plays downfield.',
    keyStats: ['Receptions', 'Receiving Yards', 'Touchdowns'],
    emoji: '🙌',
  },
  TE: {
    name: 'Tight End',
    description: 'Hybrid blocker/receiver. Can impact both running and passing game.',
    keyStats: ['Receptions', 'Yards', 'Red Zone Targets'],
    emoji: '🎯',
  },
  DST: {
    name: 'Defense/Special Teams',
    description: 'Entire defensive unit plus special teams. Points from turnovers and scores.',
    keyStats: ['Sacks', 'Interceptions', 'Points Allowed'],
    emoji: '🛡️',
  },
  K: {
    name: 'Kicker',
    description: 'Scores through field goals and extra points. Consistent but limited upside.',
    keyStats: ['Field Goals Made', 'Extra Points', 'Distance'],
    emoji: '🦵',
  },
  FLEX: {
    name: 'Flex',
    description: 'Can be RB, WR, or TE. Allows lineup flexibility for best matchups.',
    keyStats: ['Best Skill Position Player'],
    emoji: '🔀',
  },

  // Baseball positions
  P: {
    name: 'Pitcher',
    description: 'Controls the game from the mound. Points from strikeouts and wins.',
    keyStats: ['Strikeouts', 'Wins', 'ERA'],
    emoji: '⚾',
  },
  'C (Baseball)': {
    name: 'Catcher',
    description: 'Defensive leader. Manages pitchers and controls running game.',
    keyStats: ['Hits', 'RBI', 'Defensive Stats'],
    emoji: '🥊',
  },
  '1B': {
    name: 'First Base',
    description: 'Power hitter position. Focus on home runs and RBI.',
    keyStats: ['Home Runs', 'RBI', 'Batting Average'],
    emoji: '💥',
  },
  '2B': {
    name: 'Second Base',
    description: 'Middle infielder. Balance of hitting and defense.',
    keyStats: ['Hits', 'Runs', 'Stolen Bases'],
    emoji: '🏃',
  },
  '3B': {
    name: 'Third Base',
    description: 'Hot corner. Strong arm and power hitting.',
    keyStats: ['Home Runs', 'RBI', 'Doubles'],
    emoji: '🔥',
  },
  SS: {
    name: 'Shortstop',
    description: 'Premium defensive position. Often good hitters too.',
    keyStats: ['Hits', 'Runs', 'Defensive Plays'],
    emoji: '🌟',
  },
  OF: {
    name: 'Outfield',
    description: 'Covers LF, CF, or RF. Mix of power and speed.',
    keyStats: ['Home Runs', 'RBI', 'Stolen Bases'],
    emoji: '🌾',
  },
}

interface PositionTooltipProps {
  position: string
  sport?: string
  children: React.ReactElement
  className?: string
}

export default function PositionTooltip({ position, sport, children, className }: PositionTooltipProps) {
  // Adjust position key for baseball catcher if needed
  const positionKey = position === 'C' && sport === 'MLB' ? 'C (Baseball)' : position
  const info = POSITION_INFO[positionKey as keyof typeof POSITION_INFO]

  if (!info) {
    // No tooltip for unknown positions
    return children
  }

  const tooltipContent = (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <span className="text-xl">{info.emoji}</span>
        <div>
          <div className="font-semibold text-white">{info.name}</div>
          <div className="text-xs text-gray-400">{position}</div>
        </div>
      </div>
      
      <div className="text-sm text-gray-300">
        {info.description}
      </div>
      
      <div className="border-t border-gray-700 pt-2">
        <div className="text-xs font-semibold text-gray-400 mb-1">Key Stats:</div>
        <div className="flex flex-wrap gap-1">
          {info.keyStats.map((stat, index) => (
            <span
              key={index}
              className="px-2 py-0.5 text-xs rounded-full bg-gray-800 text-gray-300"
            >
              {stat}
            </span>
          ))}
        </div>
      </div>
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

// Export position info for use in other components
export { POSITION_INFO }