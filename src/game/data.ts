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
import { premierLeagueClubSeeds, type PremierLeaguePlayerSeed } from './premierLeagueData'

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

export function createInitialCareer(seed = 7321, careerId = createCareerId()): CareerState {
  const clubs = premierLeagueClubSeeds.map((clubSeed, index) => createClub({ ...clubSeed, index, seed }))
  const fixtures = createRoundRobin(clubs.map((club) => club.id), seed)
  const selectedClubId = clubs[0].id
  const starters = recommendLineup(clubs[0], defaultTactic)
  const captainId = findBestCaptain({ ...clubs[0], squad: starters }).id
  const starterIds = starters.map((player) => player.id)

  return {
    careerId,
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
  players: PremierLeaguePlayerSeed[]
  index: number
  seed: number
}): Club {
  const random = mulberry32(input.seed + input.index * 997)
  const squad = input.players.map((player, playerIndex) =>
    createPlayer(input.id, player, input.strength, playerIndex, random),
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
  seed: PremierLeaguePlayerSeed,
  teamStrength: number,
  playerIndex: number,
  random: () => number,
): Player {
  const base = clamp(seed.ability + randomRange(random, -2, 2), Math.max(48, teamStrength - 16), 96)
  const age = seed.age
  const potential = clamp(base + randomRange(random, age < 23 ? 8 : age < 26 ? 2 : 0, age < 23 ? 17 : age < 26 ? 9 : 4), base, 96)
  const personality = personalities[Math.floor(random() * personalities.length)]
  const positionBoost = positionAttributes(seed.position)
  const rating = (attribute: keyof Player, fallback = 0) =>
    clamp(Math.round(base + (positionBoost[attribute] ?? fallback) + randomRange(random, -3, 3)), 18, 99)
  const finishing = rating('finishing')
  const passing = rating('passing')
  const dribbling = rating('dribbling')
  const firstTouch = rating('firstTouch')
  const tackling = rating('tackling')
  const marking = rating('marking')
  const heading = rating('heading')
  const crossing = rating('crossing')
  const setPieces = rating('setPieces')
  const acceleration = rating('acceleration')
  const strength = rating('strength')
  const vision = rating('vision')
  const decisions = rating('decisions')
  const composure = rating('composure')
  const workRate = rating('workRate')
  const handling = rating('handling')
  const reflexes = rating('reflexes')
  const oneOnOnes = rating('oneOnOnes')
  const positioning = rating('positioning')
  const pace = clamp(Math.round((acceleration * 0.56 + rating('pace', positionBoost.pace ?? 0) * 0.44)), 18, 99)
  const stamina = rating('stamina')
  const value = Math.max(
    2,
    Math.round(
      (Math.pow(Math.max(0, base - 45) / 50, 2) * 95 + potential * 0.18 + (age < 25 ? 10 : 0) + random() * 7) * 10,
    ) / 10,
  )
  const wage = Math.round((value * 1.75 + base * 0.22 + random() * 18) * 10) / 10

  return {
    id: `${clubId}-${seed.position.toLowerCase()}-${playerIndex}`,
    clubId,
    name: seed.name,
    age,
    position: seed.position,
    finishing,
    passing,
    dribbling,
    firstTouch,
    tackling,
    marking,
    heading,
    crossing,
    setPieces,
    acceleration,
    strength,
    vision,
    decisions,
    composure,
    workRate,
    handling,
    reflexes,
    oneOnOnes,
    positioning,
    attack: averageRating([finishing, dribbling, pace]),
    defense: averageRating([tackling, marking, heading, strength]),
    technique: averageRating([passing, firstTouch, dribbling, crossing, setPieces]),
    pace,
    stamina,
    morale: clamp(Math.round(55 + randomRange(random, -9, 16)), 0, 100),
    fitness: clamp(Math.round(84 + randomRange(random, -7, 10)), 0, 100),
    potential: Math.round(potential),
    leadership: clamp(Math.round(base + randomRange(random, -20, 18) + (personality === 'Leader' ? 18 : 0)), 18, 98),
    form: clamp(Math.round(seed.form + randomRange(random, -4, 4)), 0, 100),
    wage,
    value,
    personality,
    development: {},
  }
}

type PositionAttributeBoost = Partial<Record<keyof Player, number>>

function positionAttributes(position: PlayerPosition): PositionAttributeBoost {
  return {
    GK: {
      finishing: -18,
      passing: -1,
      dribbling: -8,
      firstTouch: 1,
      tackling: -4,
      marking: 1,
      heading: 3,
      crossing: -8,
      setPieces: -4,
      acceleration: -4,
      strength: 3,
      vision: 3,
      decisions: 5,
      composure: 5,
      workRate: 1,
      handling: 9,
      reflexes: 10,
      oneOnOnes: 8,
      positioning: 9,
      pace: -5,
      stamina: -4,
    },
    DEF: {
      finishing: -18,
      passing: 0,
      dribbling: -5,
      firstTouch: -1,
      tackling: 8,
      marking: 8,
      heading: 7,
      crossing: 1,
      setPieces: -4,
      acceleration: 2,
      strength: 8,
      vision: 1,
      decisions: 5,
      composure: 3,
      workRate: 4,
      handling: -20,
      reflexes: -20,
      oneOnOnes: -18,
      positioning: -16,
      pace: 1,
      stamina: 5,
    },
    MID: {
      finishing: 1,
      passing: 8,
      dribbling: 6,
      firstTouch: 7,
      tackling: -2,
      marking: -5,
      heading: -3,
      crossing: 5,
      setPieces: 3,
      acceleration: 3,
      strength: -1,
      vision: 9,
      decisions: 8,
      composure: 4,
      workRate: 6,
      handling: -22,
      reflexes: -22,
      oneOnOnes: -20,
      positioning: -18,
      pace: 2,
      stamina: 8,
    },
    FWD: {
      finishing: 9,
      passing: -1,
      dribbling: 8,
      firstTouch: 7,
      tackling: -12,
      marking: -16,
      heading: 2,
      crossing: 2,
      setPieces: 0,
      acceleration: 8,
      strength: 3,
      vision: 1,
      decisions: 4,
      composure: 8,
      workRate: 1,
      handling: -24,
      reflexes: -24,
      oneOnOnes: -22,
      positioning: -20,
      pace: 8,
      stamina: 1,
    },
  }[position]
}

function averageRating(values: number[]) {
  return Math.round(values.reduce((sum, value) => sum + value, 0) / values.length)
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

export function createCareerId() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `career-${Date.now().toString(36)}-${Math.floor(Math.random() * 0x7fffffff).toString(36)}`
}

export function createSimulationSeed() {
  return Math.floor(Math.random() * 0x7fffffff)
}

export function hashString(value: string) {
  let hash = 2166136261
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return hash | 0
}

export function clamp(value: number, min = 0, max = 100) {
  return Math.max(min, Math.min(max, value))
}
