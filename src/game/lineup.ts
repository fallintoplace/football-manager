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

export function playerOverall(player: Player) {
  const overall =
    player.position === 'GK'
      ? weighted(player, [
          ['handling', 0.28],
          ['reflexes', 0.24],
          ['positioning', 0.2],
          ['oneOnOnes', 0.14],
          ['composure', 0.08],
          ['passing', 0.06],
        ])
      : player.position === 'DEF'
        ? weighted(player, [
            ['tackling', 0.22],
            ['marking', 0.2],
            ['heading', 0.12],
            ['strength', 0.12],
            ['pace', 0.1],
            ['passing', 0.08],
            ['decisions', 0.08],
            ['composure', 0.08],
          ])
        : player.position === 'MID'
          ? weighted(player, [
              ['passing', 0.18],
              ['vision', 0.15],
              ['firstTouch', 0.14],
              ['dribbling', 0.14],
              ['decisions', 0.13],
              ['stamina', 0.1],
              ['workRate', 0.08],
              ['finishing', 0.04],
              ['acceleration', 0.04],
            ])
          : weighted(player, [
              ['finishing', 0.24],
              ['dribbling', 0.17],
              ['firstTouch', 0.14],
              ['pace', 0.12],
              ['acceleration', 0.1],
              ['composure', 0.1],
              ['strength', 0.06],
              ['passing', 0.04],
              ['heading', 0.03],
            ])

  return clampRating(Math.round(overall), 1, 99)
}

export function playerScore(player: Player) {
  return clampRating(
    playerOverall(player) + (player.form - 50) * 0.08 + (player.fitness - 70) * 0.06 + (player.morale - 50) * 0.03,
    1,
    99,
  )
}

export function playerAttributeSummary(player: Player) {
  const attributes =
    player.position === 'GK'
      ? [
          ['HAN', player.handling],
          ['REF', player.reflexes],
          ['POS', player.positioning],
        ]
      : player.position === 'DEF'
        ? [
            ['TAC', player.tackling],
            ['MAR', player.marking],
            ['HEA', player.heading],
          ]
        : player.position === 'MID'
          ? [
              ['PAS', player.passing],
              ['VIS', player.vision],
              ['DRI', player.dribbling],
            ]
          : [
              ['FIN', player.finishing],
              ['DRI', player.dribbling],
              ['PAC', player.pace],
            ]

  return attributes.map(([label, value]) => `${label} ${value}`).join(' · ')
}

type PlayerAttribute = keyof Player

function weighted(player: Player, attributes: Array<[PlayerAttribute, number]>) {
  return attributes.reduce((total, [attribute, weight]) => total + Number(player[attribute]) * weight, 0)
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

function clampRating(value: number, min: number, max: number) {
  return Math.max(min, Math.min(max, value))
}
