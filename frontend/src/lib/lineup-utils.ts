import { Player } from '@/types/player'
import { Contest } from '@/types/contest'

// Position compatibility mappings for different sports
const POSITION_COMPATIBILITY: Record<string, Record<string, string[]>> = {
  nfl: {
    QB: ['QB'],
    RB: ['RB'],
    WR: ['WR'],
    TE: ['TE'],
    FLEX: ['RB', 'WR', 'TE'], // FLEX can be RB, WR, or TE
    DST: ['DST', 'D/ST'],
  },
  nba: {
    PG: ['PG'],
    SG: ['SG'],
    SF: ['SF'],
    PF: ['PF'],
    C: ['C'],
    G: ['PG', 'SG'], // Guard can be PG or SG
    F: ['SF', 'PF'], // Forward can be SF or PF
    UTIL: ['PG', 'SG', 'SF', 'PF', 'C'], // Utility can be any position
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
    UTIL: ['C', 'W', 'LW', 'RW', 'D'], // Utility can be any skater
  },
  golf: {
    G: ['G'], // Golfer position
    G1: ['G'], // All golfer slots can be filled by any golfer
    G2: ['G'],
    G3: ['G'],
    G4: ['G'],
    G5: ['G'],
    G6: ['G'],
  },
}

export function canPlayerFillPosition(
  player: Player,
  requiredPosition: string,
  sport: string
): boolean {
  const compatibilityMap = POSITION_COMPATIBILITY[sport]
  if (!compatibilityMap) return false

  const allowedPositions = compatibilityMap[requiredPosition]
  if (!allowedPositions) return false

  return allowedPositions.includes(player.position)
}

export function getPositionRequirements(contest: Contest | undefined): string[] {
  if (!contest) return []

  // Use position_requirements from contest if available
  if (contest.position_requirements && Object.keys(contest.position_requirements).length > 0) {
    const requirements: string[] = []
    for (const [position, count] of Object.entries(contest.position_requirements)) {
      for (let i = 0; i < count; i++) {
        requirements.push(position)
      }
    }
    return requirements
  }

  // Fallback to default requirements
  switch (contest.sport) {
    case 'nfl':
      return ['QB', 'RB', 'RB', 'WR', 'WR', 'WR', 'TE', 'FLEX', 'DST']
    case 'nba':
      return ['PG', 'SG', 'SF', 'PF', 'C', 'G', 'F', 'UTIL']
    case 'mlb':
      return ['P', 'C', '1B', '2B', '3B', 'SS', 'OF', 'OF', 'OF']
    case 'nhl':
      return ['C', 'C', 'W', 'W', 'W', 'D', 'D', 'G', 'UTIL']
    case 'golf':
      return ['G', 'G', 'G', 'G', 'G', 'G'] // 6 golfers for DFS golf contests
    default:
      return []
  }
}

export function validateLineup(
  lineup: Player[],
  contest: Contest | undefined
): { valid: boolean; errors: string[] } {
  if (!contest) {
    return { valid: false, errors: ['No contest selected'] }
  }

  const errors: string[] = []
  const positionRequirements = getPositionRequirements(contest)

  // Check if lineup is complete
  if (lineup.length !== positionRequirements.length) {
    errors.push(`Lineup must have ${positionRequirements.length} players`)
  }

  // Check salary cap
  const totalSalary = lineup.reduce((sum, player) => sum + player.salary, 0)
  if (totalSalary > contest.salary_cap) {
    errors.push(`Lineup exceeds salary cap ($${totalSalary} > $${contest.salary_cap})`)
  }

  // Check for duplicate players
  const playerIds = lineup.map(p => p.id)
  const uniquePlayerIds = new Set(playerIds)
  if (playerIds.length !== uniquePlayerIds.size) {
    errors.push('Lineup contains duplicate players')
  }

  // Validate each position
  for (let i = 0; i < positionRequirements.length; i++) {
    const requiredPosition = positionRequirements[i]
    const player = lineup[i]
    
    if (player && !canPlayerFillPosition(player, requiredPosition, contest.sport)) {
      errors.push(`${player.name} cannot fill ${requiredPosition} position`)
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

export function findBestSlotForPlayer(
  player: Player,
  positionRequirements: string[],
  currentLineup: (Player | null)[],
  sport: string
): number | null {
  // First, try to find an empty slot that the player can fill
  for (let i = 0; i < positionRequirements.length; i++) {
    if (!currentLineup[i] && canPlayerFillPosition(player, positionRequirements[i], sport)) {
      return i
    }
  }

  // If no empty slot, find a slot where we can replace a player
  // Prefer replacing players in more flexible positions (FLEX, UTIL)
  const flexPositions = ['FLEX', 'UTIL', 'G', 'F']
  
  for (let i = 0; i < positionRequirements.length; i++) {
    const position = positionRequirements[i]
    if (flexPositions.includes(position) && canPlayerFillPosition(player, position, sport)) {
      return i
    }
  }

  // Finally, try any compatible position
  for (let i = 0; i < positionRequirements.length; i++) {
    if (canPlayerFillPosition(player, positionRequirements[i], sport)) {
      return i
    }
  }

  return null
}