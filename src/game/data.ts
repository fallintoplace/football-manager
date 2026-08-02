import type {
  CareerState,
  Club,
  Fixture,
  Formation,
  Mentality,
  Personality,
  Player,
  PlayerPosition,
  Standing,
  Tactic,
} from './types'
import { recommendLineup } from './lineup'
import { premierLeagueClubSeeds } from './premierLeagueData'

export const defaultTactic: Tactic = {
  formation: '4-2-3-1',
  mentality: 'Assertive',
  pressing: 66,
  tempo: 58,
  defensiveLine: 61,
}

const personalities: Personality[] = [
  'Leader',
  'Professional',
  'Maverick',
  'Volatile',
  'Resilient',
  'Ambitious',
]

export function createInitialCareer(seed = 7321): CareerState {
  const clubs = premierLeagueClubSeeds.map((clubSeed, index) => createClub({ ...clubSeed, index, seed }))
  const fixtures = createRoundRobin(clubs.map((club) => club.id), seed)
  const selectedClubId = clubs[0].id
  const starters = recommendLineup(clubs[0], defaultTactic)
  const captainId = findBestCaptain({ ...clubs[0], squad: starters }).id
  const starterIds = starters.map((player) => player.id)

  return {
    seed,
    season: 1,
    roundIndex: 0,
    selectedClubId,
    tactic: defaultTactic,
    trainingFocus: 'Structure',
    captainId,
    starterIds,
    clubs,
    fixtures,
    standings: clubs.map(createStanding),
    results: [],
    news: [
      {
        id: 'welcome',
        tone: 'neutral',
        title: 'Preseason Mandate',
        body: 'The board wants a top-half finish, the supporters want a recognizable identity, and the dressing room is watching the first team sheet closely.',
      },
    ],
  }
}

export function createAiTactic(club: Club): Tactic {
  const formationByStyle: Record<Mentality, Formation> = {
    Measured: '4-4-2',
    Assertive: '4-2-3-1',
    Relentless: '4-3-3',
  }

  return {
    formation: formationByStyle[club.managerStyle],
    mentality: club.managerStyle,
    pressing: clamp(Math.round(48 + club.prestige * 0.32 + styleOffset(club.managerStyle)), 38, 84),
    tempo: clamp(Math.round(44 + club.prestige * 0.28 + (club.managerStyle === 'Measured' ? -5 : 8)), 36, 82),
    defensiveLine: clamp(Math.round(45 + club.prestige * 0.25 + (club.managerStyle === 'Relentless' ? 12 : 0)), 34, 80),
  }
}

export function createStanding(club: Club): Standing {
  return {
    clubId: club.id,
    played: 0,
    won: 0,
    drawn: 0,
    lost: 0,
    goalsFor: 0,
    goalsAgainst: 0,
    points: 0,
    form: [],
  }
}

export function findBestCaptain(club: Club): Player {
  return [...club.squad].sort((left, right) => {
    const rightScore = right.leadership + right.morale * 0.35 + (right.personality === 'Leader' ? 18 : 0)
    const leftScore = left.leadership + left.morale * 0.35 + (left.personality === 'Leader' ? 18 : 0)
    return rightScore - leftScore
  })[0]
}

export function createRoundRobin(teamIds: string[], seed: number): Fixture[][] {
  const shuffled = stableShuffle(teamIds, seed + 91)
  const rounds: Fixture[][] = []
  let rotation = [...shuffled]
  const roundsCount = rotation.length - 1

  for (let round = 0; round < roundsCount; round += 1) {
    const fixtures: Fixture[] = []
    for (let index = 0; index < rotation.length / 2; index += 1) {
      const left = rotation[index]
      const right = rotation[rotation.length - 1 - index]
      const flip = (round + index) % 2 === 0
      fixtures.push({
        id: `s${seed}-r${round + 1}-m${index + 1}`,
        round,
        homeId: flip ? left : right,
        awayId: flip ? right : left,
      })
    }

    rounds.push(fixtures)
    rotation = [rotation[0], rotation[rotation.length - 1], ...rotation.slice(1, rotation.length - 1)]
  }

  const returnRounds = rounds.map((fixtures, roundIndex) =>
    fixtures.map((fixture, fixtureIndex) => ({
      ...fixture,
      id: `s${seed}-r${roundIndex + roundsCount + 1}-m${fixtureIndex + 1}`,
      round: roundIndex + roundsCount,
      homeId: fixture.awayId,
      awayId: fixture.homeId,
    })),
  )

  return [...rounds, ...returnRounds]
}

function createClub(input: {
  id: string
  name: string
  shortName: string
  city: string
  badgeUrl: string
  colors: [string, string]
  strength: number
  style: Mentality
  players: Array<{ name: string; position: PlayerPosition }>
  index: number
  seed: number
}): Club {
  const random = mulberry32(input.seed + input.index * 997)
  const squad = input.players.map((player, playerIndex) =>
    createPlayer(input.id, player.name, player.position, input.strength, playerIndex, random),
  )

  return {
    id: input.id,
    name: input.name,
    shortName: input.shortName,
    city: input.city,
    badgeUrl: input.badgeUrl,
    colors: input.colors,
    budget: Number((4.5 + input.strength * 0.16 + random() * 3).toFixed(1)),
    prestige: input.strength,
    fanMood: clamp(48 + Math.round(random() * 18), 0, 100),
    boardTrust: clamp(52 + Math.round(random() * 18), 0, 100),
    managerStyle: input.style,
    squad,
  }
}

function createPlayer(
  clubId: string,
  name: string,
  position: PlayerPosition,
  teamStrength: number,
  playerIndex: number,
  random: () => number,
): Player {
  const base = teamStrength + randomRange(random, -12, 9)
  const age = Math.round(randomRange(random, 18, 33))
  const potential = clamp(base + randomRange(random, age < 23 ? 8 : -4, age < 24 ? 22 : 8), 42, 96)
  const personality = personalities[Math.floor(random() * personalities.length)]
  const wage = Math.round((base * 0.075 + random() * 1.4) * 10) / 10
  const value = Math.round((base * 0.32 + potential * 0.25 + random() * 8) * 10) / 10

  const positionBoost = {
    GK: { attack: -22, defense: 21, technique: 0, pace: -4, stamina: -6 },
    DEF: { attack: -9, defense: 13, technique: -2, pace: 1, stamina: 5 },
    MID: { attack: 3, defense: 2, technique: 11, pace: 2, stamina: 8 },
    FWD: { attack: 16, defense: -12, technique: 5, pace: 8, stamina: 0 },
  }[position]

  return {
    id: `${clubId}-${position.toLowerCase()}-${playerIndex}`,
    clubId,
    name,
    age,
    position,
    attack: clamp(Math.round(base + positionBoost.attack + randomRange(random, -8, 8)), 18, 96),
    defense: clamp(Math.round(base + positionBoost.defense + randomRange(random, -8, 8)), 18, 96),
    technique: clamp(Math.round(base + positionBoost.technique + randomRange(random, -8, 8)), 18, 96),
    pace: clamp(Math.round(base + positionBoost.pace + randomRange(random, -9, 9)), 18, 96),
    stamina: clamp(Math.round(base + positionBoost.stamina + randomRange(random, -8, 8)), 18, 96),
    morale: clamp(Math.round(55 + randomRange(random, -9, 16)), 0, 100),
    fitness: clamp(Math.round(84 + randomRange(random, -7, 10)), 0, 100),
    potential,
    leadership: clamp(Math.round(base + randomRange(random, -20, 18) + (personality === 'Leader' ? 18 : 0)), 18, 98),
    form: clamp(Math.round(56 + randomRange(random, -12, 15)), 0, 100),
    wage,
    value,
    personality,
  }
}

function stableShuffle(items: string[], seed: number): string[] {
  const random = mulberry32(seed)
  const output = [...items]

  for (let index = output.length - 1; index > 0; index -= 1) {
    const swapIndex = Math.floor(random() * (index + 1))
    const item = output[index]
    output[index] = output[swapIndex]
    output[swapIndex] = item
  }

  return output
}

function styleOffset(style: Mentality) {
  if (style === 'Relentless') return 14
  if (style === 'Assertive') return 6
  return -2
}

export function mulberry32(seed: number) {
  return function random() {
    let t = (seed += 0x6d2b79f5)
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

export function randomRange(random: () => number, min: number, max: number) {
  return min + random() * (max - min)
}

export function clamp(value: number, min = 0, max = 100) {
  return Math.max(min, Math.min(max, value))
}
