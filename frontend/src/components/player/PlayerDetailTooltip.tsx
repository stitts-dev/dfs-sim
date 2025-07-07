import { useMemo } from 'react'
import { Player } from '@/types/player'
import { formatCurrency, formatNumber, formatPercentage } from '@/lib/utils'
import { TooltipSection } from '@/components/ui/Tooltip'

interface PlayerDetailTooltipProps {
  player: Player
  showAdvanced: boolean
  sport: string
}

// Position explanations for different sports
const POSITION_DETAILS: Record<string, Record<string, { name: string; description: string }>> = {
  nfl: {
    QB: { name: 'Quarterback', description: 'Throws passes, leads the offense' },
    RB: { name: 'Running Back', description: 'Runs with the ball, catches short passes' },
    WR: { name: 'Wide Receiver', description: 'Catches passes, primary receiving target' },
    TE: { name: 'Tight End', description: 'Hybrid receiver/blocker, catches passes' },
    FLEX: { name: 'Flexible Position', description: 'Can be filled by RB, WR, or TE' },
    DST: { name: 'Defense/Special Teams', description: 'Entire defensive unit + special teams' },
  },
  nba: {
    PG: { name: 'Point Guard', description: 'Primary ball handler, sets up plays' },
    SG: { name: 'Shooting Guard', description: 'Scorer, shoots from outside' },
    SF: { name: 'Small Forward', description: 'Versatile wing player' },
    PF: { name: 'Power Forward', description: 'Inside scorer, rebounder' },
    C: { name: 'Center', description: 'Tallest player, protects the rim' },
    G: { name: 'Guard', description: 'Can be filled by PG or SG' },
    F: { name: 'Forward', description: 'Can be filled by SF or PF' },
    UTIL: { name: 'Utility', description: 'Can be filled by any position' },
  },
  mlb: {
    P: { name: 'Pitcher', description: 'Throws to batters, earns points for strikeouts' },
    C: { name: 'Catcher', description: 'Catches pitches, defensive position' },
    '1B': { name: 'First Baseman', description: 'Fields near first base' },
    '2B': { name: 'Second Baseman', description: 'Fields between first and second base' },
    '3B': { name: 'Third Baseman', description: 'Fields near third base' },
    SS: { name: 'Shortstop', description: 'Fields between second and third base' },
    OF: { name: 'Outfielder', description: 'Fields in the outfield' },
  },
  nhl: {
    C: { name: 'Center', description: 'Takes faceoffs, two-way player' },
    W: { name: 'Winger', description: 'Scoring focused forward' },
    D: { name: 'Defenseman', description: 'Protects goal, can score points' },
    G: { name: 'Goalie', description: 'Prevents goals, earns points for saves/wins' },
    UTIL: { name: 'Utility', description: 'Can be filled by any skater' },
  },
}

// Scoring explanations
const SCORING_INFO: Record<string, string> = {
  nfl: 'Points for yards, touchdowns, receptions (PPR)',
  nba: 'Points for scoring, rebounds, assists, steals, blocks',
  mlb: 'Points for hits, runs, RBIs, home runs, stolen bases',
  nhl: 'Points for goals, assists, shots, blocks, +/-',
}

export default function PlayerDetailTooltip({ player, showAdvanced, sport }: PlayerDetailTooltipProps) {
  const positionInfo = POSITION_DETAILS[sport]?.[player.position]
  const value = player.projected_points / (player.salary / 1000)
  
  const valueIndicator = useMemo(() => {
    if (value >= 5) return { text: 'Excellent Value', color: 'text-green-400' }
    if (value >= 4) return { text: 'Good Value', color: 'text-yellow-400' }
    if (value >= 3) return { text: 'Fair Value', color: 'text-orange-400' }
    return { text: 'Poor Value', color: 'text-red-400' }
  }, [value])

  const injuryInfo = useMemo(() => {
    if (!player.is_injured) return null
    const status = player.injury_status || 'Injured'
    if (status.toLowerCase().includes('questionable')) {
      return { text: 'Questionable - 50% chance to play', color: 'text-yellow-400' }
    }
    if (status.toLowerCase().includes('doubtful')) {
      return { text: 'Doubtful - 25% chance to play', color: 'text-orange-400' }
    }
    if (status.toLowerCase().includes('out')) {
      return { text: 'Out - Will not play', color: 'text-red-400' }
    }
    return { text: status, color: 'text-yellow-400' }
  }, [player.is_injured, player.injury_status])

  return (
    <div className="space-y-3 p-1">
      {/* Position Information */}
      <TooltipSection title="Position">
        <div className="space-y-1">
          <div className="font-medium">{positionInfo?.name || player.position}</div>
          {positionInfo?.description && (
            <div className="text-xs opacity-80">{positionInfo.description}</div>
          )}
        </div>
      </TooltipSection>

      {/* Basic Stats */}
      <TooltipSection title="Projections">
        <div className="space-y-1">
          <div className="flex justify-between">
            <span>Projected Points:</span>
            <span className="font-semibold">{formatNumber(player.projected_points)}</span>
          </div>
          <div className="flex justify-between">
            <span>Salary:</span>
            <span>{formatCurrency(player.salary)}</span>
          </div>
          <div className="flex justify-between">
            <span>Value:</span>
            <span className={valueIndicator.color}>
              {formatNumber(value, 2)}x • {valueIndicator.text}
            </span>
          </div>
        </div>
      </TooltipSection>

      {/* Matchup Information */}
      <TooltipSection title="Matchup">
        <div className="space-y-1">
          <div className="flex justify-between">
            <span>Opponent:</span>
            <span>{player.opponent}</span>
          </div>
          <div className="flex justify-between">
            <span>Game Time:</span>
            <span>{new Date(player.game_time).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}</span>
          </div>
        </div>
      </TooltipSection>

      {/* Injury Status */}
      {injuryInfo && (
        <TooltipSection title="Injury Status">
          <div className={injuryInfo.color}>{injuryInfo.text}</div>
        </TooltipSection>
      )}

      {/* Advanced Stats (if enabled) */}
      {showAdvanced && (
        <>
          <TooltipSection title="Advanced Metrics">
            <div className="space-y-1">
              <div className="flex justify-between">
                <span>Floor:</span>
                <span>{formatNumber(player.floor_points)}</span>
              </div>
              <div className="flex justify-between">
                <span>Ceiling:</span>
                <span>{formatNumber(player.ceiling_points)}</span>
              </div>
              <div className="flex justify-between">
                <span>Ownership:</span>
                <span>{formatPercentage(player.ownership)}</span>
              </div>
            </div>
          </TooltipSection>

          <TooltipSection title="GPP vs Cash">
            <div className="text-xs space-y-1">
              <div className="opacity-80">
                {player.ownership > 30 
                  ? '⚠️ High ownership - better for cash games'
                  : '✅ Low ownership - good GPP leverage'}
              </div>
              <div className="opacity-80">
                {(player.ceiling_points - player.floor_points) / player.projected_points > 0.5
                  ? '🎲 High variance - GPP play'
                  : '🛡️ Consistent - cash game play'}
              </div>
            </div>
          </TooltipSection>
        </>
      )}

      {/* Scoring System Info */}
      <TooltipSection>
        <div className="text-xs opacity-70 border-t border-white/20 pt-2 mt-2">
          <div className="font-medium mb-1">How {sport.toUpperCase()} scoring works:</div>
          <div>{SCORING_INFO[sport]}</div>
        </div>
      </TooltipSection>

      {/* Keyboard Shortcut Hint */}
      <div className="text-xs opacity-50 text-center pt-1">
        Press F1 to toggle all tooltips
      </div>
    </div>
  )
}