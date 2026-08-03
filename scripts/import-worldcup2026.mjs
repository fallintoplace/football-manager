const sourceUrl = option('--source-url', 'https://raw.githubusercontent.com/openfootball/worldcup.json/master/2026/worldcup.json')
const analyticsUrl = option('--analytics-url', 'http://localhost:8787')

const source = await getJson(sourceUrl)
const matches = source.matches ?? []
if (matches.length !== 104) throw new Error(`Expected 104 World Cup matches, received ${matches.length}`)

const payload = {
  source: 'openfootball',
  sourceVersion: 'openfootball/worldcup.json@master',
  tournament: 'world-cup-2026',
  matches: matches.map((match, index) => toWorldCupMatch(match, index + 1)),
}

const response = await fetch(`${analyticsUrl}/api/analytics/worldcup-2026/import`, {
  method: 'POST',
  headers: { 'content-type': 'application/json' },
  body: JSON.stringify(payload),
})
const body = await response.json()
if (!response.ok) throw new Error(JSON.stringify(body))
console.log(JSON.stringify({ ...body, sourceUrl }, null, 2))

function toWorldCupMatch(match, fallbackNumber) {
  const score = match.score ?? {}
  const regulation = score.ft ?? [0, 0]
  const final = score.et ?? regulation
  const shootout = score.p ?? [0, 0]
  const matchNumber = Number(match.num ?? fallbackNumber)
  return {
    sourceMatchId: `openfootball:2026:${matchNumber}`,
    matchNumber,
    round: match.round ?? '',
    groupName: match.group ?? '',
    matchDate: match.date,
    kickoffTime: match.time ?? '',
    venue: match.ground ?? 'Unknown venue',
    homeTeam: match.team1,
    awayTeam: match.team2,
    regulationHomeGoals: Number(regulation[0] ?? 0),
    regulationAwayGoals: Number(regulation[1] ?? 0),
    homeGoals: Number(final[0] ?? 0),
    awayGoals: Number(final[1] ?? 0),
    shootoutHomeGoals: Number(shootout[0] ?? 0),
    shootoutAwayGoals: Number(shootout[1] ?? 0),
    goals: [
      ...(match.goals1 ?? []).map((goal) => toGoal(match.team1, goal)),
      ...(match.goals2 ?? []).map((goal) => toGoal(match.team2, goal)),
    ].sort((left, right) => left.minuteValue - right.minuteValue),
  }
}

function toGoal(teamName, goal) {
  return {
    teamName,
    playerName: goal.name,
    minute: String(goal.minute),
    minuteValue: parseMinute(goal.minute),
    isPenalty: Boolean(goal.penalty),
    isOwnGoal: Boolean(goal.owngoal),
  }
}

function parseMinute(value) {
  const [base, added] = String(value).split('+')
  return Number(base) + Number(added ?? 0)
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
