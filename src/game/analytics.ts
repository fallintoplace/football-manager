import { playerOverall } from './lineup'
import type { CareerState, Formation, MatchResult, Mentality, PlayerPosition, SeasonAnalyticsSnapshot } from './types'

export type AnalyticsMatchdayPayload = {
  runId: string
  season: number
  round: number
  clubs: Array<{
    id: string
    name: string
    shortName: string
  }>
  players: Array<{
    id: string
    clubId: string
    name: string
    position: PlayerPosition
  }>
  standings: AnalyticsStandingSnapshot[]
  playerSnapshots: AnalyticsPlayerSnapshot[]
  results: MatchResult[]
}

export type AnalyticsStandingSnapshot = {
  clubId: string
  rank: number
  played: number
  won: number
  drawn: number
  lost: number
  goalsFor: number
  goalsAgainst: number
  goalDifference: number
  points: number
  form: string
}

export type AnalyticsPlayerSnapshot = {
  playerId: string
  clubId: string
  playerName: string
  position: PlayerPosition
  age: number
  overall: number
  potential: number
  form: number
  morale: number
  fitness: number
  value: number
  averageRating: number
  appeared: boolean
}

export type AnalyticsPlayer = {
  playerId: string
  playerName: string
  matches: number
  averageRating: number
}

export type AnalyticsSummary = {
  matches: number
  events: number
  frames: number
  averageXg: number
  averagePossession: number
  players: AnalyticsPlayer[]
  source: string
}

export type AnalyticsRun = {
  runId: string
  careerId: string
  season: number
  lastRound: number
  roundsCompleted: number
  matchesInRound: number
  clubsExpected: number
  status: 'in_progress' | 'complete'
  schemaVersion: number
  updatedAt: string
}

export type AnalyticsSeasonComparison = {
  careerId: string
  runId: string
  season: number
  status: 'in_progress' | 'complete'
  lastRound: number
  rank: number
  played: number
  won: number
  drawn: number
  lost: number
  points: number
  goalsFor: number
  goalsAgainst: number
  matches: number
  xgFor: number
  xgAgainst: number
  averageRating: number
  averagePossession: number
  averagePressure: number
  averagePressWins: number
  averageBoxEntries: number
  playerMinutes: number
}

export type TacticalMatchup = {
  runId: string
  clubId: string
  opponentId: string
  clubFormation: Formation
  clubMentality: Mentality
  clubPressing: number
  clubTempo: number
  clubDefensiveLine: number
  opponentFormation: Formation
  opponentMentality: Mentality
  opponentPressing: number
  opponentTempo: number
  opponentDefensiveLine: number
  matches: number
  goalsFor: number
  goalsAgainst: number
  xgFor: number
  xgAgainst: number
  possession: number
  pressWins: number
  opponentPressWins: number
  boxEntries: number
  opponentBoxEntries: number
  counters: number
  buildUpFails: number
  opponentBuildUpFails: number
}

export type ActionMixRow = {
  actionType: string
  actions: number
  successful: number
  successRate: number
}

export type PassNetworkLink = {
  passerId: string
  receiverId: string
  attempts: number
  completions: number
  progressivePasses: number
  completionRate: number
}

export type PlayerActionProfile = {
  playerId: string
  primaryRole: string
  actions: number
  passes: number
  completedPasses: number
  completionRate: number
  progressiveActions: number
  carries: number
  shots: number
  xg: number
  defensiveActions: number
  buildUpActions: number
  finalThirdActions: number
  boxActions: number
}

export type ActionInsights = {
  runId: string
  clubId: string
  matches: number
  actions: number
  possessions: number
  passes: number
  completedPasses: number
  passCompletion: number
  progressivePasses: number
  shots: number
  shotsOnTarget: number
  xg: number
  carries: number
  successfulCarries: number
  finalThirdActions: number
  actionMix: ActionMixRow[]
  passNetwork: PassNetworkLink[]
  playerProfiles: PlayerActionProfile[]
  analystNote: string
}

export type RealMatchSummary = {
  source: string
  sourceMatchId: string
  competition: string
  season: number
  seasonLabel: string
  matchDate: string
  homeTeamName: string
  homeScore: number
  awayTeamName: string
  awayScore: number
  actions: number
  players: number
  possessions: number
  xg: number
}

export type RealShot = {
  teamName: string
  playerName: string
  second: number
  startX: number
  startY: number
  xg: number
  outcome: string
}

export type RealPassNetworkLink = {
  teamName: string
  passer: string
  receiver: string
  attempts: number
  completions: number
  completionRate: number
}

export type RealPlayerProfile = {
  playerId: string
  playerName: string
  teamName: string
  actions: number
  passes: number
  completedPasses: number
  completionRate: number
  carries: number
  shots: number
  xg: number
  defensiveActions: number
}

export type RealMatchExplorer = {
  match: RealMatchSummary
  shots: RealShot[]
  passNetwork: RealPassNetworkLink[]
  playerProfiles: RealPlayerProfile[]
}

export type WorldCup2026Goal = {
  matchNumber?: number
  teamName: string
  playerName: string
  minute: string
  minuteValue: number
  isPenalty: boolean
  isOwnGoal: boolean
}

export type WorldCup2026MatchSummary = {
  sourceMatchId: string
  matchNumber: number
  round: string
  groupName: string
  matchDate: string
  kickoffTime: string
  venue: string
  homeTeam: string
  awayTeam: string
  regulationHomeGoals: number
  regulationAwayGoals: number
  homeGoals: number
  awayGoals: number
  shootoutHomeGoals: number
  shootoutAwayGoals: number
  penaltyShootout: boolean
  winner: string
  goals: WorldCup2026Goal[]
}

export type WorldCup2026Summary = {
  matches: number
  teams: number
  goals: number
  averageGoals: number
  venues: number
  penaltyShootouts: number
  extraTimeMatches: number
  champion: string
  runnerUp: string
  finalScore: string
}

export type WorldCup2026TeamRow = {
  teamName: string
  groupName: string
  rank: number
  played: number
  won: number
  drawn: number
  lost: number
  goalsFor: number
  goalsAgainst: number
  goalDifference: number
  points: number
  stage: string
}

export type WorldCup2026Scorer = {
  playerName: string
  teamName: string
  goals: number
  penaltyGoals: number
  matches: number
}

export type WorldCup2026TimingBucket = {
  label: string
  goals: number
}

export type WorldCup2026VenueRow = {
  venue: string
  matches: number
  goals: number
  averageGoals: number
}

export type WorldCup2026Overview = {
  source: string
  sourceVersion: string
  tournament: string
  summary: WorldCup2026Summary
  teams: WorldCup2026TeamRow[]
  topScorers: WorldCup2026Scorer[]
  goalTiming: WorldCup2026TimingBucket[]
  venues: WorldCup2026VenueRow[]
  matches: WorldCup2026MatchSummary[]
}

export type IcebergSnapshot = {
  snapshotId: number
  parentSnapshotId?: number
  sequenceNumber: number
  timestampMs: number
  occurredAt: string
  summary: string
}

export type IcebergHistory = {
  table: string
  currentSnapshotId?: number
  snapshots: IcebergSnapshot[]
}

export type AnalyticsTimelinePoint = {
  round: number
  rank: number
  played: number
  won: number
  drawn: number
  lost: number
  goalsFor: number
  goalsAgainst: number
  goalDifference: number
  points: number
  form: string
  xgFor: number
  xgAgainst: number
  possession: number
  shotsFor: number
  shotsAgainst: number
  pressure: number
  territory: number
  events: number
  frames: number
}

export type AnalyticsTimelineStanding = {
  round: number
  clubId: string
  rank: number
  played: number
  won: number
  drawn: number
  lost: number
  goalsFor: number
  goalsAgainst: number
  goalDifference: number
  points: number
  form: string
}

export type AnalyticsDevelopmentPlayer = {
  playerId: string
  playerName: string
  position: PlayerPosition
  openingOverall: number
  overall: number
  potential: number
  form: number
  fitness: number
  change: number
}

export type AnalyticsTimeline = {
  points: AnalyticsTimelinePoint[]
  table: AnalyticsTimelineStanding[]
  players: AnalyticsDevelopmentPlayer[]
  source: string
}

export type AnalyticsSyncStatus = 'unknown' | 'syncing' | 'online' | 'offline'

const analyticsBaseUrl = import.meta.env.VITE_ANALYTICS_URL ?? 'http://localhost:8787'
export const clickhouseUiUrl = import.meta.env.VITE_CLICKHOUSE_UI_URL ?? 'http://localhost:8123/clickstack'

export function seasonRunId(state: CareerState) {
  return `${state.careerId}:season:${state.season}`
}

export function buildAnalyticsPayload(
  state: CareerState,
  results: MatchResult[],
  snapshot?: SeasonAnalyticsSnapshot,
): AnalyticsMatchdayPayload {
  const players = snapshot?.players ?? state.clubs.flatMap((club) => club.squad)
  const standings = [...(snapshot?.standings ?? state.standings)].sort(compareStandings)
  const ratings = new Map<string, number[]>()

  for (const result of results) {
    for (const [playerId, rating] of Object.entries(result.playerRatings)) {
      ratings.set(playerId, [...(ratings.get(playerId) ?? []), rating])
    }
  }

  return {
    runId: seasonRunId(state),
    season: state.season,
    round: results[0]?.round ?? snapshot?.round ?? state.roundIndex,
    clubs: state.clubs.map(({ id, name, shortName }) => ({ id, name, shortName })),
    players: players.map(({ id, clubId, name, position }) => ({ id, clubId, name, position })),
    standings: standings.map((standing, index) => ({
      clubId: standing.clubId,
      rank: index + 1,
      played: standing.played,
      won: standing.won,
      drawn: standing.drawn,
      lost: standing.lost,
      goalsFor: standing.goalsFor,
      goalsAgainst: standing.goalsAgainst,
      goalDifference: standing.goalsFor - standing.goalsAgainst,
      points: standing.points,
      form: standing.form.join(''),
    })),
    playerSnapshots: players.map((player) => {
      const playerRatings = ratings.get(player.id) ?? []
      return {
        playerId: player.id,
        clubId: player.clubId,
        playerName: player.name,
        position: player.position,
        age: player.age,
        overall: playerOverall(player),
        potential: player.potential,
        form: player.form,
        morale: player.morale,
        fitness: player.fitness,
        value: player.value,
        averageRating: playerRatings.length ? average(playerRatings) : 0,
        appeared: playerRatings.length > 0,
      }
    }),
    results,
  }
}

export async function ingestMatchday(payload: AnalyticsMatchdayPayload): Promise<void> {
  const response = await fetch(`${analyticsBaseUrl}/ingest`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!response.ok) throw new Error(`Analytics ingest failed with ${response.status}`)
}

export async function fetchAnalyticsSummary(clubId: string, runId?: string): Promise<AnalyticsSummary> {
  const query = new URLSearchParams({ club_id: clubId })
  if (runId) query.set('run_id', runId)
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/summary?${query.toString()}`)
  if (!response.ok) throw new Error(`Analytics summary failed with ${response.status}`)
  return (await response.json()) as AnalyticsSummary
}

export async function fetchAnalyticsRuns(): Promise<AnalyticsRun[]> {
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/runs`)
  if (!response.ok) throw new Error(`Analytics runs failed with ${response.status}`)
  return (await response.json()) as AnalyticsRun[]
}

export async function fetchSeasonComparison(careerId: string, clubId: string): Promise<AnalyticsSeasonComparison[]> {
  const query = new URLSearchParams({ career_id: careerId, club_id: clubId })
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/season-comparison?${query.toString()}`)
  if (!response.ok) throw new Error(`Season comparison failed with ${response.status}`)
  return (await response.json()) as AnalyticsSeasonComparison[]
}

export async function fetchTacticalMatchups(runId: string, clubId: string, opponentId?: string): Promise<TacticalMatchup[]> {
  const query = new URLSearchParams({ run_id: runId, club_id: clubId })
  if (opponentId) query.set('opponent_id', opponentId)
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/tactical-matchup?${query.toString()}`)
  if (!response.ok) throw new Error(`Tactical matchup failed with ${response.status}`)
  return (await response.json()) as TacticalMatchup[]
}

export async function fetchActionInsights(runId: string, clubId: string): Promise<ActionInsights> {
  const query = new URLSearchParams({ run_id: runId, club_id: clubId })
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/action-insights?${query.toString()}`)
  if (!response.ok) throw new Error(`Action insights failed with ${response.status}`)
  return (await response.json()) as ActionInsights
}

export async function fetchRealDataMatches(source = 'statsbomb', season?: number): Promise<RealMatchSummary[]> {
  const query = new URLSearchParams({ source })
  if (season) query.set('season', String(season))
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/real-data/matches?${query.toString()}`)
  if (!response.ok) throw new Error(`Real data matches failed with ${response.status}`)
  return (await response.json()) as RealMatchSummary[]
}

export async function fetchRealDataMatch(source: string, sourceMatchId: string): Promise<RealMatchExplorer> {
  const query = new URLSearchParams({ source, source_match_id: sourceMatchId })
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/real-data/match?${query.toString()}`)
  if (!response.ok) throw new Error(`Real data match failed with ${response.status}`)
  return (await response.json()) as RealMatchExplorer
}

export async function fetchWorldCup2026Overview(): Promise<WorldCup2026Overview> {
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/worldcup-2026/overview`)
  if (!response.ok) throw new Error(`World Cup 2026 overview failed with ${response.status}`)
  return (await response.json()) as WorldCup2026Overview
}

export async function fetchIcebergHistory(table = 'player_match_facts'): Promise<IcebergHistory> {
  const query = new URLSearchParams({ table })
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/iceberg/history?${query.toString()}`)
  if (!response.ok) throw new Error(`Iceberg history failed with ${response.status}`)
  return (await response.json()) as IcebergHistory
}

export async function fetchAnalyticsTimeline(clubId: string, runId: string): Promise<AnalyticsTimeline> {
  const query = new URLSearchParams({ club_id: clubId, run_id: runId })
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/timeline?${query.toString()}`)
  if (!response.ok) throw new Error(`Analytics timeline failed with ${response.status}`)
  return (await response.json()) as AnalyticsTimeline
}

function average(values: number[]) {
  return values.reduce((sum, value) => sum + value, 0) / values.length
}

function compareStandings(left: CareerState['standings'][number], right: CareerState['standings'][number]) {
  const rightGoalDifference = right.goalsFor - right.goalsAgainst
  const leftGoalDifference = left.goalsFor - left.goalsAgainst
  return (
    right.points - left.points ||
    rightGoalDifference - leftGoalDifference ||
    right.goalsFor - left.goalsFor ||
    left.clubId.localeCompare(right.clubId)
  )
}
