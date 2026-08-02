import type { CareerState, MatchResult, PlayerPosition } from './types'

export type AnalyticsMatchdayPayload = {
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
  results: MatchResult[]
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

export type AnalyticsSyncStatus = 'unknown' | 'syncing' | 'online' | 'offline'

const analyticsBaseUrl = import.meta.env.VITE_ANALYTICS_URL ?? 'http://localhost:8787'

export function buildAnalyticsPayload(state: CareerState, results: MatchResult[]): AnalyticsMatchdayPayload {
  return {
    season: state.season,
    round: results[0]?.round ?? state.roundIndex,
    clubs: state.clubs.map(({ id, name, shortName }) => ({ id, name, shortName })),
    players: state.clubs.flatMap((club) =>
      club.squad.map(({ id, clubId, name, position }) => ({ id, clubId, name, position })),
    ),
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

export async function fetchAnalyticsSummary(clubId: string): Promise<AnalyticsSummary> {
  const response = await fetch(`${analyticsBaseUrl}/api/analytics/summary?club_id=${encodeURIComponent(clubId)}`)
  if (!response.ok) throw new Error(`Analytics summary failed with ${response.status}`)
  return (await response.json()) as AnalyticsSummary
}
