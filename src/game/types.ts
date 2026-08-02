export type PlayerPosition = 'GK' | 'DEF' | 'MID' | 'FWD'

export type Formation = '4-3-3' | '4-2-3-1' | '4-4-2' | '3-5-2'

export type Mentality = 'Measured' | 'Assertive' | 'Relentless'

export type TrainingFocus = 'Recovery' | 'Finishing' | 'Structure' | 'Pressing' | 'Conditioning'

export type Personality =
  | 'Leader'
  | 'Professional'
  | 'Maverick'
  | 'Volatile'
  | 'Resilient'
  | 'Ambitious'

export type Tactic = {
  formation: Formation
  mentality: Mentality
  pressing: number
  tempo: number
  defensiveLine: number
}

export type Player = {
  id: string
  clubId: string
  name: string
  age: number
  position: PlayerPosition
  finishing: number
  passing: number
  dribbling: number
  firstTouch: number
  tackling: number
  marking: number
  heading: number
  crossing: number
  setPieces: number
  acceleration: number
  strength: number
  vision: number
  decisions: number
  composure: number
  workRate: number
  handling: number
  reflexes: number
  oneOnOnes: number
  positioning: number
  attack: number
  defense: number
  technique: number
  pace: number
  stamina: number
  morale: number
  fitness: number
  potential: number
  leadership: number
  form: number
  wage: number
  value: number
  personality: Personality
}

export type Club = {
  id: string
  name: string
  shortName: string
  city: string
  badgeUrl: string
  colors: [string, string]
  budget: number
  prestige: number
  fanMood: number
  boardTrust: number
  managerStyle: Mentality
  squad: Player[]
}

export type Fixture = {
  id: string
  round: number
  homeId: string
  awayId: string
}

export type Standing = {
  clubId: string
  played: number
  won: number
  drawn: number
  lost: number
  goalsFor: number
  goalsAgainst: number
  points: number
  form: string[]
}

export type MatchEventType = 'goal' | 'chance' | 'save' | 'pressure' | 'card' | 'shift'

export type MatchEvent = {
  minute: number
  type: MatchEventType
  teamId: string
  playerName: string
  text: string
  xg?: number
}

export type MatchPhase = 'build' | 'midfield' | 'final-third' | 'box' | 'transition' | 'shot'

export type PlayerIntent = 'hold' | 'press' | 'support' | 'run' | 'mark' | 'recover' | 'shoot'

export type TraceBall = {
  x: number
  y: number
}

export type TracePlayer = {
  id: string
  teamId: string
  name: string
  number: number
  position: PlayerPosition
  x: number
  y: number
  targetX: number
  targetY: number
  intent: PlayerIntent
}

export type MatchFrame = {
  minute: number
  phase: MatchPhase
  possessingTeamId?: string
  ball: TraceBall
  players: TracePlayer[]
  note?: string
}

export type MatchMetrics = {
  homeXg: number
  awayXg: number
  homeShots: number
  awayShots: number
  homeShotsOnTarget: number
  awayShotsOnTarget: number
  homePossession: number
  homePressure: number
  awayPressure: number
  homeTerritory: number
}

export type MatchResult = {
  fixtureId: string
  round: number
  homeId: string
  awayId: string
  homeGoals: number
  awayGoals: number
  events: MatchEvent[]
  trace: MatchFrame[]
  metrics: MatchMetrics
  report: string[]
  playerRatings: Record<string, number>
}

export type NewsItem = {
  id: string
  tone: 'good' | 'bad' | 'neutral'
  title: string
  body: string
}

export type CareerState = {
  seed: number
  season: number
  roundIndex: number
  selectedClubId: string
  tactic: Tactic
  trainingFocus: TrainingFocus
  captainId: string
  starterIds: string[]
  clubs: Club[]
  fixtures: Fixture[][]
  standings: Standing[]
  results: MatchResult[]
  lastMatchId?: string
  news: NewsItem[]
  savedAt?: string
}

export type SeasonAnalyticsSnapshot = {
  round: number
  standings: Standing[]
  players: Player[]
}
