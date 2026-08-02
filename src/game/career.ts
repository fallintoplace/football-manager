import { clamp, createInitialCareer, createRoundRobin, createStanding } from './data'
import { recommendLineup, playerScore, validateLineup } from './lineup'
import { simulateMatch } from './sim'
import type { CareerState, Club, MatchResult, NewsItem, Player, Standing, Tactic, TrainingFocus } from './types'

const saveKey = 'touchline-career-v2'

export function loadCareer(): CareerState | undefined {
  try {
    const raw = window.localStorage.getItem(saveKey)
    if (!raw) return undefined
    return hydrateCareer(JSON.parse(raw) as CareerState)
  } catch {
    return undefined
  }
}

export function saveCareer(state: CareerState): CareerState {
  const savedAt = new Date().toISOString()
  const next = { ...state, savedAt }
  window.localStorage.setItem(saveKey, JSON.stringify(next))
  return next
}

export function resetCareer(seed = Math.floor(Date.now() % 100000)) {
  window.localStorage.removeItem(saveKey)
  return createInitialCareer(seed)
}

export function changeSelectedClub(state: CareerState, selectedClubId: string): CareerState {
  const selected = state.clubs.find((club) => club.id === selectedClubId)
  if (!selected) return state
  const captain = [...selected.squad].sort((left, right) => playerScore(right) - playerScore(left))[0]

  return withStarterIds({
    ...state,
    selectedClubId,
    captainId: captain.id,
    starterIds: recommendLineup(selected, state.tactic).map((player) => player.id),
    news: [
      {
        id: `appointment-${selectedClubId}-${state.roundIndex}`,
        tone: 'neutral',
        title: `${selected.shortName} Appointment`,
        body: `${selected.name} hand you control of matchday selection and the tactical identity.`,
      },
      ...state.news.slice(0, 7),
    ],
  })
}

export function updateTactic(state: CareerState, tactic: Partial<Tactic>): CareerState {
  const nextTactic = {
    ...state.tactic,
    ...tactic,
  }
  const selectedClub = getClub(state, state.selectedClubId)
  const starterIds = tactic.formation
    ? recommendLineup(selectedClub, nextTactic).map((player) => player.id)
    : state.starterIds

  return withStarterIds({
    ...state,
    tactic: nextTactic,
    starterIds,
  })
}

export function updateTraining(state: CareerState, trainingFocus: TrainingFocus): CareerState {
  return { ...state, trainingFocus }
}

export function updateCaptain(state: CareerState, captainId: string): CareerState {
  return { ...state, captainId }
}

export function recommendStarters(state: CareerState): CareerState {
  const club = getClub(state, state.selectedClubId)
  return withStarterIds({
    ...state,
    starterIds: recommendLineup(club, state.tactic).map((player) => player.id),
  })
}

export function toggleStarter(state: CareerState, playerId: string): CareerState {
  const club = getClub(state, state.selectedClubId)
  const player = club.squad.find((item) => item.id === playerId)
  if (!player) return state

  if (state.starterIds.includes(playerId)) {
    return withStarterIds({
      ...state,
      starterIds: state.starterIds.filter((id) => id !== playerId),
    })
  }

  if (state.starterIds.length < 11) {
    return withStarterIds({
      ...state,
      starterIds: [...state.starterIds, playerId],
    })
  }

  const starters = state.starterIds
    .map((id) => club.squad.find((item) => item.id === id))
    .filter((item): item is Player => Boolean(item))
  const samePosition = starters.filter((starter) => starter.position === player.position)
  const replacementPool = samePosition.length ? samePosition : starters
  const outgoing = [...replacementPool].sort((left, right) => playerScore(left) - playerScore(right))[0]

  return withStarterIds({
    ...state,
    starterIds: state.starterIds.map((id) => (id === outgoing.id ? playerId : id)),
  })
}

export type MatchdayAdvance = {
  state: CareerState
  results: MatchResult[]
}

export function advanceMatchday(state: CareerState): CareerState {
  return advanceMatchdayWithResults(state).state
}

export function advanceMatchdayWithResults(state: CareerState): MatchdayAdvance {
  if (state.roundIndex >= state.fixtures.length) {
    return { state: startNextSeason(state), results: [] }
  }

  const lineup = validateLineup(getClub(state, state.selectedClubId), state.tactic, state.starterIds)
  if (!lineup.isLegal) {
    return {
      state: {
        ...state,
        news: [
          {
            id: `illegal-lineup-${state.roundIndex}-${lineup.warnings.length}`,
            tone: 'bad',
            title: 'Team Sheet Blocked',
            body: lineup.warnings[0] ?? 'The starting XI does not match the selected formation.',
          },
          ...state.news.slice(0, 7),
        ],
      },
      results: [],
    }
  }

  const trainedClubs = state.clubs.map((club) =>
    club.id === state.selectedClubId ? applyTraining(club, state.trainingFocus, state.tactic) : passiveRecovery(club),
  )
  const trainedState = { ...state, clubs: trainedClubs }
  const roundFixtures = state.fixtures[state.roundIndex]
  const roundResults = roundFixtures.map((fixture) => simulateMatch(trainedState, fixture))
  const standings = applyResultsToStandings(state.standings, roundResults)
  const selectedResult = roundResults.find((result) => result.homeId === state.selectedClubId || result.awayId === state.selectedClubId)
  const clubs = trainedClubs.map((club) => applyMatchMood(club, roundResults, state.selectedClubId, state.tactic))
  const news = buildNews(state, clubs, roundResults, selectedResult)

  return {
    state: {
      ...state,
      clubs,
      standings,
      results: [...state.results, ...roundResults],
      roundIndex: state.roundIndex + 1,
      lastMatchId: selectedResult?.fixtureId ?? state.lastMatchId,
      news,
    },
    results: roundResults,
  }
}

export function getSortedStandings(standings: Standing[]) {
  return [...standings].sort((left, right) => {
    const goalDiffRight = right.goalsFor - right.goalsAgainst
    const goalDiffLeft = left.goalsFor - left.goalsAgainst
    return (
      right.points - left.points ||
      goalDiffRight - goalDiffLeft ||
      right.goalsFor - left.goalsFor ||
      left.clubId.localeCompare(right.clubId)
    )
  })
}

export function getClub(state: CareerState, clubId: string): Club {
  const club = state.clubs.find((item) => item.id === clubId)
  if (!club) throw new Error(`Missing club ${clubId}`)
  return club
}

export function getLastMatch(state: CareerState) {
  if (!state.lastMatchId) return undefined
  return state.results.find((result) => result.fixtureId === state.lastMatchId)
}

export function getUpcomingFixture(state: CareerState) {
  return state.fixtures[state.roundIndex]?.find(
    (fixture) => fixture.homeId === state.selectedClubId || fixture.awayId === state.selectedClubId,
  )
}

export function teamAverage(club: Club) {
  return Math.round(club.squad.reduce((sum, player) => sum + playerScore(player), 0) / club.squad.length)
}

function applyResultsToStandings(standings: Standing[], results: MatchResult[]) {
  const next = standings.map((standing) => ({ ...standing, form: [...standing.form] }))

  for (const result of results) {
    const home = next.find((standing) => standing.clubId === result.homeId)
    const away = next.find((standing) => standing.clubId === result.awayId)
    if (!home || !away) continue
    home.played += 1
    away.played += 1
    home.goalsFor += result.homeGoals
    home.goalsAgainst += result.awayGoals
    away.goalsFor += result.awayGoals
    away.goalsAgainst += result.homeGoals

    if (result.homeGoals > result.awayGoals) {
      home.won += 1
      away.lost += 1
      home.points += 3
      home.form = [...home.form, 'W'].slice(-5)
      away.form = [...away.form, 'L'].slice(-5)
    } else if (result.homeGoals < result.awayGoals) {
      away.won += 1
      home.lost += 1
      away.points += 3
      away.form = [...away.form, 'W'].slice(-5)
      home.form = [...home.form, 'L'].slice(-5)
    } else {
      home.drawn += 1
      away.drawn += 1
      home.points += 1
      away.points += 1
      home.form = [...home.form, 'D'].slice(-5)
      away.form = [...away.form, 'D'].slice(-5)
    }
  }

  return next
}

function applyTraining(club: Club, focus: TrainingFocus, tactic: Tactic): Club {
  const deltas: Record<TrainingFocus, Partial<Record<keyof Player, number>>> = {
    Recovery: { fitness: 9, morale: 2 },
    Finishing: { attack: 1.2, technique: 0.6, fitness: -2 },
    Structure: { defense: 1.1, morale: 1, fitness: 1 },
    Pressing: { stamina: 1, pace: 0.6, fitness: -3 },
    Conditioning: { stamina: 1.4, fitness: 3, morale: -1 },
  }
  const selected = deltas[focus]
  const fatigueTax = tactic.pressing > 74 ? -1.4 : tactic.tempo > 72 ? -0.8 : 0

  return {
    ...club,
    squad: club.squad.map((player) => ({
      ...player,
      attack: clamp(Math.round(player.attack + (selected.attack ?? 0)), 18, 99),
      defense: clamp(Math.round(player.defense + (selected.defense ?? 0)), 18, 99),
      technique: clamp(Math.round(player.technique + (selected.technique ?? 0)), 18, 99),
      pace: clamp(Math.round(player.pace + (selected.pace ?? 0)), 18, 99),
      stamina: clamp(Math.round(player.stamina + (selected.stamina ?? 0)), 18, 99),
      morale: clamp(Math.round(player.morale + (selected.morale ?? 0)), 0, 100),
      fitness: clamp(Math.round(player.fitness + (selected.fitness ?? 0) + fatigueTax), 0, 100),
    })),
  }
}

function passiveRecovery(club: Club): Club {
  return {
    ...club,
    squad: club.squad.map((player) => ({
      ...player,
      fitness: clamp(player.fitness + 2, 0, 100),
      morale: clamp(player.morale + (player.form > 62 ? 1 : 0), 0, 100),
    })),
  }
}

function applyMatchMood(club: Club, results: MatchResult[], selectedClubId: string, tactic: Tactic): Club {
  const result = results.find((item) => item.homeId === club.id || item.awayId === club.id)
  if (!result) return club
  const goalsFor = result.homeId === club.id ? result.homeGoals : result.awayGoals
  const goalsAgainst = result.homeId === club.id ? result.awayGoals : result.homeGoals
  const won = goalsFor > goalsAgainst
  const lost = goalsFor < goalsAgainst
  const isSelected = club.id === selectedClubId
  const pressureFitnessCost = isSelected ? Math.max(0, tactic.pressing - 55) * 0.05 : 1
  const mood = won ? 4 : lost ? -4 : 1
  const budgetDelta = isSelected ? (won ? 0.55 : lost ? 0.1 : 0.28) : 0

  return {
    ...club,
    fanMood: clamp(club.fanMood + (won ? 5 : lost ? -6 : 1), 0, 100),
    boardTrust: clamp(club.boardTrust + (won ? 4 : lost ? -5 : 0), 0, 100),
    budget: Number((club.budget + budgetDelta).toFixed(1)),
    squad: club.squad.map((player) => {
      const rating = result.playerRatings[player.id]
      return {
        ...player,
        morale: clamp(Math.round(player.morale + mood + (rating ? rating - 6.7 : 0)), 0, 100),
        fitness: clamp(Math.round(player.fitness - (rating ? 5 + pressureFitnessCost : 1)), 0, 100),
        form: clamp(Math.round(player.form * 0.72 + (rating ? rating * 10 : player.form) * 0.28), 0, 100),
      }
    }),
  }
}

function buildNews(state: CareerState, clubs: Club[], results: MatchResult[], selectedResult?: MatchResult) {
  const selectedClub = clubs.find((club) => club.id === state.selectedClubId)
  const items: NewsItem[] = []

  if (selectedClub && selectedResult) {
    const goalsFor = selectedResult.homeId === selectedClub.id ? selectedResult.homeGoals : selectedResult.awayGoals
    const goalsAgainst = selectedResult.homeId === selectedClub.id ? selectedResult.awayGoals : selectedResult.homeGoals
    const tone = goalsFor > goalsAgainst ? 'good' : goalsFor < goalsAgainst ? 'bad' : 'neutral'
    const topLine =
      goalsFor > goalsAgainst
        ? 'Dressing Room Lift'
        : goalsFor < goalsAgainst
          ? 'Pressure Builds'
          : 'Questions Remain'

    items.push({
      id: `${selectedResult.fixtureId}-headline`,
      tone,
      title: topLine,
      body: `${selectedClub.shortName} ${goalsFor}-${goalsAgainst}: ${selectedResult.report[0]}`,
    })
  }

  const biggestWin = [...results].sort(
    (left, right) => Math.abs(right.homeGoals - right.awayGoals) - Math.abs(left.homeGoals - left.awayGoals),
  )[0]

  if (biggestWin && Math.abs(biggestWin.homeGoals - biggestWin.awayGoals) >= 2) {
    const club = clubs.find((item) => item.id === (biggestWin.homeGoals > biggestWin.awayGoals ? biggestWin.homeId : biggestWin.awayId))
    if (club) {
      items.push({
        id: `${biggestWin.fixtureId}-statement`,
        tone: 'good',
        title: `${club.shortName} Statement`,
        body: `${club.name} turn the matchday into a warning shot for the rest of the league.`,
      })
    }
  }

  return [...items, ...state.news].slice(0, 8)
}

function startNextSeason(state: CareerState): CareerState {
  const seed = state.seed + state.season * 131
  const clubs = state.clubs.map((club) => ({
    ...club,
    fanMood: clamp(Math.round(club.fanMood * 0.66 + 18), 0, 100),
    boardTrust: clamp(Math.round(club.boardTrust * 0.7 + 16), 0, 100),
    squad: club.squad.map(developPlayer),
  }))
  const selected = clubs.find((club) => club.id === state.selectedClubId) ?? clubs[0]
  const starterIds = recommendLineup(selected, state.tactic).map((player) => player.id)

  return withStarterIds({
    ...state,
    seed,
    season: state.season + 1,
    roundIndex: 0,
    clubs,
    fixtures: createRoundRobin(clubs.map((club) => club.id), seed),
    standings: clubs.map(createStanding),
    results: [],
    lastMatchId: undefined,
    captainId: selected.squad.some((player) => player.id === state.captainId) ? state.captainId : selected.squad[0].id,
    starterIds,
    news: [
      {
        id: `season-${state.season + 1}`,
        tone: 'neutral',
        title: `Season ${state.season + 1} Begins`,
        body: 'Contracts reset, supporters recalibrate, and every tactical promise is back on trial.',
      },
      ...state.news.slice(0, 6),
    ],
  })
}

function hydrateCareer(state: CareerState): CareerState {
  const selectedClub = state.clubs.find((club) => club.id === state.selectedClubId) ?? state.clubs[0]
  const starterIds = state.starterIds?.length
    ? state.starterIds.filter((id) => selectedClub.squad.some((player) => player.id === id)).slice(0, 11)
    : recommendLineup(selectedClub, state.tactic).map((player) => player.id)

  return withStarterIds({
    ...state,
    starterIds,
  })
}

function withStarterIds(state: CareerState): CareerState {
  const club = getClub(state, state.selectedClubId)
  const uniqueStarterIds = [...new Set(state.starterIds)]
    .filter((id) => club.squad.some((player) => player.id === id))
    .slice(0, 11)
  const captainId = uniqueStarterIds.includes(state.captainId)
    ? state.captainId
    : [...club.squad]
        .filter((player) => uniqueStarterIds.includes(player.id))
        .sort((left, right) => playerScore(right) - playerScore(left))[0]?.id || club.squad[0].id

  return {
    ...state,
    starterIds: uniqueStarterIds,
    captainId,
  }
}

function developPlayer(player: Player): Player {
  const youthGrowth = player.age <= 23 ? Math.max(0, player.potential - playerScore(player) * 0.72) * 0.035 : 0
  const ageDrag = player.age >= 31 ? -1.2 : 0
  const growth = clamp(youthGrowth + ageDrag, -2, 3)

  return {
    ...player,
    age: player.age + 1,
    attack: clamp(Math.round(player.attack + growth), 18, 99),
    defense: clamp(Math.round(player.defense + growth), 18, 99),
    technique: clamp(Math.round(player.technique + growth), 18, 99),
    pace: clamp(Math.round(player.pace + (player.age >= 31 ? -2 : growth * 0.4)), 18, 99),
    stamina: clamp(Math.round(player.stamina + (player.age >= 31 ? -1 : growth * 0.3)), 18, 99),
    morale: clamp(Math.round(player.morale * 0.82 + 11), 0, 100),
    fitness: clamp(Math.round(player.fitness * 0.75 + 22), 0, 100),
  }
}
