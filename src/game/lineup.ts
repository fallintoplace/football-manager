import type { Club, Formation, Player, PlayerPosition, Tactic } from './types'

export type LineupValidation = {
  isLegal: boolean
  starters: Player[]
  counts: Record<PlayerPosition, number>
  required: Record<PlayerPosition, number>
  warnings: string[]
  averageScore: number
  averageFitness: number
  averageMorale: number
}

const positions: PlayerPosition[] = ['GK', 'DEF', 'MID', 'FWD']

export function formationRequirements(formation: Formation): Record<PlayerPosition, number> {
  if (formation === '4-2-3-1') {
    return { GK: 1, DEF: 4, MID: 5, FWD: 1 }
  }

  const [defenders, midfielders, forwards] = formation.split('-').map(Number)
  return {
    GK: 1,
    DEF: defenders,
    MID: midfielders,
    FWD: forwards,
  }
}

export function recommendLineup(club: Club, tactic: Tactic): Player[] {
  const required = formationRequirements(tactic.formation)
  const lineup: Player[] = []

  for (const position of positions) {
    const players = club.squad
      .filter((player) => player.position === position)
      .sort((left, right) => playerScore(right) - playerScore(left))
      .slice(0, required[position])
    lineup.push(...players)
  }

  if (lineup.length < 11) {
    const extras = club.squad
      .filter((player) => !lineup.includes(player))
      .sort((left, right) => playerScore(right) - playerScore(left))
      .slice(0, 11 - lineup.length)
    lineup.push(...extras)
  }

  return lineup.slice(0, 11)
}

export function getLineupPlayers(club: Club, tactic: Tactic, starterIds?: string[]) {
  if (!starterIds?.length) {
    return recommendLineup(club, tactic)
  }

  const selected = starterIds
    .map((id) => club.squad.find((player) => player.id === id))
    .filter((player): player is Player => Boolean(player))

  return selected.length ? selected : recommendLineup(club, tactic)
}

export function validateLineup(club: Club, tactic: Tactic, starterIds: string[]): LineupValidation {
  const required = formationRequirements(tactic.formation)
  const starters = getLineupPlayers(club, tactic, starterIds)
  const counts = countPositions(starters)
  const warnings: string[] = []

  if (starters.length !== 11) {
    warnings.push(`Pick exactly 11 starters. Current team sheet has ${starters.length}.`)
  }

  for (const position of positions) {
    const delta = counts[position] - required[position]
    if (delta < 0) {
      warnings.push(`Need ${Math.abs(delta)} more ${position}.`)
    } else if (delta > 0) {
      warnings.push(`Too many ${position}: reduce by ${delta}.`)
    }
  }

  const tired = starters.filter((player) => player.fitness < 62)
  const unsettled = starters.filter((player) => player.morale < 42)
  const volatile = starters.filter((player) => player.personality === 'Volatile' && player.morale < 52)

  if (tired.length) {
    warnings.push(`${tired.length} starter${tired.length === 1 ? '' : 's'} carrying fatigue risk.`)
  }

  if (unsettled.length) {
    warnings.push(`${unsettled.length} starter${unsettled.length === 1 ? '' : 's'} low on morale.`)
  }

  if (volatile.length) {
    warnings.push(`${volatile[0].name} may react badly if the match turns.`)
  }

  return {
    isLegal: starters.length === 11 && positions.every((position) => counts[position] === required[position]),
    starters,
    counts,
    required,
    warnings,
    averageScore: average(starters.map(playerScore)),
    averageFitness: average(starters.map((player) => player.fitness)),
    averageMorale: average(starters.map((player) => player.morale)),
  }
}

export function playerScore(player: Player) {
  const roleScore =
    player.position === 'GK'
      ? player.defense * 0.76 + player.technique * 0.12 + player.leadership * 0.12
      : player.position === 'DEF'
        ? player.defense * 0.58 + player.pace * 0.18 + player.stamina * 0.14 + player.technique * 0.1
        : player.position === 'MID'
          ? player.technique * 0.42 + player.stamina * 0.24 + player.attack * 0.18 + player.defense * 0.16
          : player.attack * 0.5 + player.pace * 0.22 + player.technique * 0.2 + player.stamina * 0.08

  return roleScore + player.morale * 0.08 + player.fitness * 0.1 + player.form * 0.06
}

function countPositions(players: Player[]): Record<PlayerPosition, number> {
  return players.reduce(
    (counts, player) => ({
      ...counts,
      [player.position]: counts[player.position] + 1,
    }),
    { GK: 0, DEF: 0, MID: 0, FWD: 0 },
  )
}

function average(values: number[]) {
  if (!values.length) return 0
  return Math.round(values.reduce((sum, value) => sum + value, 0) / values.length)
}
