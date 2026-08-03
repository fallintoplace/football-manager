const actionTypes = new Map([
  ['Pass', 'pass'],
  ['Carry', 'carry'],
  ['Shot', 'shot'],
  ['Duel', 'duel'],
  ['Interception', 'interception'],
  ['Clearance', 'clearance'],
  ['Block', 'block'],
  ['Ball Recovery', 'recovery'],
  ['Pressure', 'pressure'],
  ['Foul Committed', 'foul'],
  ['Foul Won', 'foul_won'],
  ['Dispossessed', 'dispossessed'],
  ['Dribbled Past', 'dribbled_past'],
  ['Miscontrol', 'miscontrol'],
  ['Goal Keeper', 'keeper_action'],
])

const sourceRoot = 'https://raw.githubusercontent.com/hudl/open-data/master/data'
const analyticsUrl = option('--analytics-url', 'http://localhost:8787')
const competitionId = option('--competition-id', '2')
const seasonId = option('--season-id', '27')
const matchId = option('--match-id')
const requestedLimit = Number(option('--limit', '1'))
const limit = Number.isFinite(requestedLimit) ? Math.max(1, Math.min(Math.floor(requestedLimit), 20)) : 1

const matches = await getJson(`${sourceRoot}/matches/${competitionId}/${seasonId}.json`)
const orderedMatches = [...matches].sort((left, right) => left.match_date.localeCompare(right.match_date) || left.match_id - right.match_id)
const selectedMatches = matchId ? orderedMatches.filter((match) => String(match.match_id) === matchId) : sampleMatches(orderedMatches, limit)
if (!selectedMatches.length) throw new Error(`No StatsBomb match found for ${matchId ?? `${competitionId}/${seasonId}`}`)

const firstMatch = selectedMatches[0]
const payload = {
  source: 'statsbomb',
  sourceVersion: 'hudl/open-data@master',
  competition: firstMatch.competition.competition_name,
  season: Number(firstMatch.season.season_name.slice(0, 4)),
  seasonLabel: firstMatch.season.season_name,
  matches: await Promise.all(selectedMatches.map(toRealMatch)),
}

const response = await fetch(`${analyticsUrl}/api/analytics/real-data/import`, {
  method: 'POST',
  headers: { 'content-type': 'application/json' },
  body: JSON.stringify(payload),
})
const body = await response.json()
if (!response.ok) throw new Error(JSON.stringify(body))
console.log(JSON.stringify(body, null, 2))

async function toRealMatch(match) {
  const events = await getJson(`${sourceRoot}/events/${match.match_id}.json`)
  return {
    sourceMatchId: String(match.match_id),
    matchDate: match.match_date,
    homeTeamId: `statsbomb:team:${match.home_team.home_team_id}`,
    homeTeamName: match.home_team.home_team_name,
    awayTeamId: `statsbomb:team:${match.away_team.away_team_id}`,
    awayTeamName: match.away_team.away_team_name,
    homeScore: match.home_score,
    awayScore: match.away_score,
    actions: events.map((event) => toRealAction(match.match_id, event)).filter(Boolean),
  }
}

function toRealAction(matchId, event) {
  const actionType = actionTypes.get(event.type?.name)
  if (!actionType || !event.player || !event.team || !Array.isArray(event.location)) return null

  const endLocation = event.pass?.end_location ?? event.carry?.end_location
  const startX = normalizeX(event.location[0])
  const startY = normalizeY(event.location[1])
  const qualifiers = {
    sourceCoordinateSystem: 'statsbomb-120x80',
    sourceEventType: event.type.name,
    playPattern: event.play_pattern?.name,
    underPressure: Boolean(event.under_pressure),
    rawStartX: event.location[0],
    rawStartY: event.location[1],
    forwardDistance: endLocation ? round(normalizeX(endLocation[0]) - startX) : undefined,
    outcome: event.pass?.outcome?.name ?? event.shot?.outcome?.name ?? event.duel?.outcome?.name,
    shotXg: event.shot?.statsbomb_xg,
    shotOutcome: event.shot?.outcome?.name,
    bodyPart: event.pass?.body_part?.name ?? event.shot?.body_part?.name,
    passHeight: event.pass?.height?.name,
    position: event.position?.name,
  }

  const action = {
    sourceActionId: `${matchId}:${event.id}`,
    possessionId: `statsbomb:${matchId}:possession:${event.possession ?? event.index}`,
    sequenceId: `statsbomb:${matchId}:sequence:${event.possession ?? event.index}`,
    period: event.period,
    second: event.minute * 60 + event.second,
    teamId: `statsbomb:team:${event.team.id}`,
    teamName: event.team.name,
    playerId: `statsbomb:player:${event.player.id}`,
    playerName: event.player.name,
    recipientPlayerId: event.pass?.recipient ? `statsbomb:player:${event.pass.recipient.id}` : '',
    recipientPlayerName: event.pass?.recipient?.name ?? '',
    actionType,
    outcome: actionOutcome(event),
    startX,
    startY,
    endX: endLocation ? normalizeX(endLocation[0]) : undefined,
    endY: endLocation ? normalizeY(endLocation[1]) : undefined,
    qualifiers: withoutUndefined(qualifiers),
  }
  return withoutUndefined(action)
}

function actionOutcome(event) {
  const outcome = event.pass?.outcome?.name ?? event.shot?.outcome?.name ?? event.duel?.outcome?.name ?? event.interception?.outcome?.name
  if ((event.type?.name === 'Pass' || event.type?.name === 'Carry' || event.type?.name === 'Ball Recovery') && !outcome) return 'successful'
  if (!outcome) return 'neutral'
  if (['Complete', 'Goal', 'Won', 'Success', 'Successful'].includes(outcome)) return 'successful'
  if (['Incomplete', 'Lost', 'Blocked', 'Offside', 'Missed', 'Post', 'Saved'].includes(outcome)) return 'unsuccessful'
  return 'neutral'
}

function normalizeX(value) {
  return round((value / 120) * 100)
}

function normalizeY(value) {
  return round((value / 80) * 100)
}

function round(value) {
  return Number(value.toFixed(2))
}

function withoutUndefined(value) {
  return Object.fromEntries(Object.entries(value).filter(([, item]) => item !== undefined && item !== null))
}

async function getJson(url) {
  const response = await fetch(url)
  if (!response.ok) throw new Error(`${response.status} while fetching ${url}`)
  return response.json()
}

function option(name, fallback) {
  const argument = process.argv.find((value) => value.startsWith(`${name}=`))
  return argument ? argument.slice(name.length + 1) : fallback
}

function sampleMatches(items, max) {
  if (items.length <= max) return items
  if (max === 1) return [items[0]]
  return Array.from({ length: max }, (_, index) => items[Math.round(index * (items.length - 1) / (max - 1))])
}
