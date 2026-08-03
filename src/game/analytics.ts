import { playerOverall } from './lineup'
import type { CareerState, MatchResult, PlayerPosition, SeasonAnalyticsSnapshot } from './types'

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
