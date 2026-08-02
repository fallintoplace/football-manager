import { clamp, createAiTactic, hashString, mulberry32, randomRange } from './data'
import { getLineupPlayers, playerOverall, playerScore } from './lineup'
import type {
  CareerState,
  Club,
  Fixture,
  MatchEvent,
  MatchFrame,
  MatchPhase,
  MatchResult,
  Player,
  PlayerIntent,
  PlayerPosition,
  Tactic,
} from './types'

type Phase = 'build' | 'midfield' | 'final-third' | 'box'

type TeamProfile = {
  club: Club
  tactic: Tactic
  lineup: Player[]
  keeper: Player
  defenders: Player[]
  midfielders: Player[]
  forwards: Player[]
  buildUp: number
  pressResistance: number
  midfieldControl: number
  lineBreaking: number
  boxThreat: number
  boxDefense: number
  press: number
  recovery: number
  keeperSkill: number
  disciplineRisk: number
  fatigueRisk: number
  captainLift: number
}

type TeamLedger = {
  possessions: number
  finalThirdEntries: number
  boxEntries: number
  shots: number
  shotsOnTarget: number
  goals: number
  xg: number
  pressWins: number
  buildUpFails: number
  midfieldWins: number
  lineBreaks: number
  ballsBehind: number
  counters: number
  saves: number
  cards: number
  territory: number
  lateFatigueLosses: number
}

type MatchLedger = Record<string, TeamLedger>

type PossessionOutcome = {
  nextControlId?: string
  events: MatchEvent[]
  frames: MatchFrame[]
}

export function simulateMatch(state: CareerState, fixture: Fixture): MatchResult {
  const home = getClub(state, fixture.homeId)
  const away = getClub(state, fixture.awayId)
  const homeTactic = fixture.homeId === state.selectedClubId ? state.tactic : createAiTactic(home)
  const awayTactic = fixture.awayId === state.selectedClubId ? state.tactic : createAiTactic(away)
  const seed = state.seed ^ hashString(`simulation-v1:${state.season}:${fixture.id}`)
  const random = mulberry32(seed)
  const homeProfile = buildProfile(
    home,
    homeTactic,
    fixture.homeId === state.selectedClubId ? state.captainId : undefined,
    fixture.homeId === state.selectedClubId ? state.starterIds : undefined,
  )
  const awayProfile = buildProfile(
    away,
    awayTactic,
    fixture.awayId === state.selectedClubId ? state.captainId : undefined,
    fixture.awayId === state.selectedClubId ? state.starterIds : undefined,
  )
  const ledger = createLedger(homeProfile, awayProfile)
  const events: MatchEvent[] = []
  const trace: MatchFrame[] = [
    createTraceFrame({
      minute: 0,
      phase: 'build',
      homeId: home.id,
      attacking: homeProfile,
      defending: awayProfile,
      ball: { x: 50, y: 50 },
      possessingTeamId: home.id,
      note: 'Kickoff shape',
    }),
  ]
  const playerRatings = seedRatings(homeProfile, awayProfile)
  const possessionHome = calculatePossessionShare(homeProfile, awayProfile)
  const possessions = clamp(
    Math.round(50 + (homeTactic.tempo + awayTactic.tempo) * 0.08 + (homeTactic.pressing + awayTactic.pressing) * 0.035 + randomRange(random, -4, 5)),
    52,
    72,
  )
  let nextControlId: string | undefined

  for (let index = 0; index < possessions; index += 1) {
    const minute = clamp(Math.round((index / possessions) * 90 + randomRange(random, 1, 4)), 1, 90)
    const homeControls =
      nextControlId === home.id
        ? true
        : nextControlId === away.id
          ? false
          : random() < (possessionHome + (minute < 8 ? 2 : 0)) / 100
    const attacking = homeControls ? homeProfile : awayProfile
    const defending = homeControls ? awayProfile : homeProfile

    const outcome = simulatePossession({
      minute,
      attacking,
      defending,
      homeId: home.id,
      random,
      ledger,
      playerRatings,
    })

    events.push(...outcome.events)
    trace.push(...outcome.frames)
    nextControlId = outcome.nextControlId
  }

  maybeAddCard(events, homeProfile, random, ledger[home.id])
  maybeAddCard(events, awayProfile, random, ledger[away.id])
  ensureMatchHasEvents(events, homeProfile, awayProfile)

  const homeLedger = ledger[home.id]
  const awayLedger = ledger[away.id]
  finishRatings(homeProfile, awayProfile, ledger, playerRatings)

  return {
    fixtureId: fixture.id,
    round: fixture.round,
    homeId: home.id,
    awayId: away.id,
    homeGoals: homeLedger.goals,
    awayGoals: awayLedger.goals,
    events: retainImportantEvents(events, 24),
    trace: retainTraceWindow(trace, 280),
    metrics: {
      homeXg: roundOne(homeLedger.xg),
      awayXg: roundOne(awayLedger.xg),
      homeShots: homeLedger.shots,
      awayShots: awayLedger.shots,
      homeShotsOnTarget: homeLedger.shotsOnTarget,
      awayShotsOnTarget: awayLedger.shotsOnTarget,
      homePossession: Math.round(possessionHome),
      homePressure: Math.round(homeProfile.press),
      awayPressure: Math.round(awayProfile.press),
      homeTerritory: territoryShare(homeLedger, awayLedger),
    },
    report: buildReport(homeProfile, awayProfile, ledger, possessionHome),
    playerRatings,
  }
}

function retainImportantEvents(events: MatchEvent[], limit: number) {
  const sorted = [...events].sort((left, right) => left.minute - right.minute)
  const important = sorted.filter((event) => event.type === 'goal' || event.type === 'card')
  const other = sorted.filter((event) => event.type !== 'goal' && event.type !== 'card')
  return [...important, ...other.slice(0, Math.max(0, limit - important.length))].sort((left, right) => left.minute - right.minute)
}

function retainTraceWindow(trace: MatchFrame[], limit: number) {
  if (trace.length <= limit) return trace
  const head = Math.floor(limit / 2)
  return [...trace.slice(0, head), ...trace.slice(-head)]
}

function simulatePossession({
  minute,
  attacking,
  defending,
  homeId,
  random,
  ledger,
  playerRatings,
}: {
  minute: number
  attacking: TeamProfile
  defending: TeamProfile
  homeId: string
  random: () => number
  ledger: MatchLedger
  playerRatings: Record<string, number>
}): PossessionOutcome {
  const events: MatchEvent[] = []
  const frames: MatchFrame[] = []
  const attackLog = ledger[attacking.club.id]
  const defendLog = ledger[defending.club.id]
  attackLog.possessions += 1
  let momentum = 0
  const lane = possessionLane(attacking, random)
  frames.push(
    createTraceFrame({
      minute,
      phase: 'build',
      homeId,
      attacking,
      defending,
      ball: { x: 14, y: lane },
      possessingTeamId: attacking.club.id,
      note: `${attacking.club.shortName} begin the possession`,
    }),
  )

  const buildup = battle(
    attacking.buildUp + attacking.pressResistance * 0.24 + tempoCalm(attacking.tactic),
    defending.press * 0.72 + defending.recovery * 0.1 + pressAggression(defending.tactic),
    random,
  )

  if (buildup < -7) {
    attackLog.buildUpFails += 1
    const presser = pickWeighted(
      [...defending.forwards, ...defending.midfielders],
      random,
      (player) => player.workRate + player.stamina + player.decisions,
    )

    if (buildup < -14) {
      defendLog.pressWins += 1
      nudgeRating(playerRatings, presser.id, 0.12)
      frames.push(
        createTraceFrame({
          minute: minute + 0.15,
          phase: 'transition',
          homeId,
          attacking: defending,
          defending: attacking,
          ball: { x: 34, y: lane },
          possessingTeamId: defending.club.id,
          focusPlayerId: presser.id,
          note: `${presser.name} jumps the first pass`,
        }),
      )

      if (random() < dangerousTurnoverChance(defending, attacking, minute)) {
        defendLog.counters += 1
        events.push({
          minute,
          type: 'pressure',
          teamId: defending.club.id,
          playerName: presser.name,
          text: `${presser.name} wins it high and turns ${defending.club.shortName} toward goal.`,
        })
        events.push(resolveShot(minute, defending, attacking, random, ledger, playerRatings, 'counter'))
        frames.push(
          createTraceFrame({
            minute: minute + 0.35,
            phase: 'shot',
            homeId,
            attacking: defending,
            defending: attacking,
            ball: { x: 86, y: lane + randomRange(random, -8, 8) },
            possessingTeamId: defending.club.id,
            note: `${defending.club.shortName} counter into a shot`,
          }),
        )
        return { events, frames, nextControlId: attacking.club.id }
      }
    }

    if (buildup < -14 && random() < 0.2) {
      events.push({
        minute,
        type: 'pressure',
        teamId: defending.club.id,
        playerName: presser.name,
        text: `${defending.club.shortName}'s press traps the buildup before it reaches midfield.`,
      })
    }
    return { events, frames, nextControlId: defending.club.id }
  }

  momentum += buildup > 10 ? 4 : 1
  attackLog.territory += 1
  frames.push(
    createTraceFrame({
      minute: minute + 0.2,
      phase: 'midfield',
      homeId,
      attacking,
      defending,
      ball: { x: 42, y: lane + randomRange(random, -9, 9) },
      possessingTeamId: attacking.club.id,
      note: `${attacking.club.shortName} move through midfield`,
    }),
  )

  const midfieldDuel = battle(
    attacking.midfieldControl + momentum + tempoControl(attacking.tactic),
    defending.midfieldControl * 0.72 + defending.press * 0.24 + defensiveBlock(defending.tactic),
    random,
  )

  if (midfieldDuel < -5) {
    defendLog.midfieldWins += 1
    const winner = pickWeighted(
      defending.midfielders,
      random,
      (player) => player.tackling + player.stamina + player.decisions,
    )
    nudgeRating(playerRatings, winner.id, 0.1)
    if (minute > 70 && attacking.fatigueRisk > 30) {
      attackLog.lateFatigueLosses += 1
    }
    if (random() < 0.18) {
      events.push({
        minute,
        type: 'shift',
        teamId: defending.club.id,
        playerName: winner.name,
        text: `${winner.name} wins the second ball and stops ${attacking.club.shortName}'s rhythm.`,
      })
    }
    frames.push(
      createTraceFrame({
        minute: minute + 0.38,
        phase: 'transition',
        homeId,
        attacking: defending,
        defending: attacking,
        ball: { x: 47, y: lane + randomRange(random, -8, 8) },
        possessingTeamId: defending.club.id,
        focusPlayerId: winner.id,
        note: `${winner.name} wins the midfield duel`,
      }),
    )
    return { events, frames, nextControlId: defending.club.id }
  }

  attackLog.finalThirdEntries += 1
  attackLog.territory += 2
  momentum += midfieldDuel > 11 ? 5 : 2
  frames.push(
    createTraceFrame({
      minute: minute + 0.4,
      phase: 'final-third',
      homeId,
      attacking,
      defending,
      ball: { x: 64, y: lane + randomRange(random, -12, 12) },
      possessingTeamId: attacking.club.id,
      note: `${attacking.club.shortName} reach the final third`,
    }),
  )

  const behindRisk = highLineBehindRisk(attacking, defending, random)
  if (behindRisk > 0.72) {
    attackLog.ballsBehind += 1
    attackLog.lineBreaks += 1
    const passer = pickWeighted(attacking.midfielders, random, (player) => player.passing + player.vision + player.decisions)
    const runner = pickWeighted(attacking.forwards, random, (player) => player.acceleration + player.pace + player.dribbling)
    nudgeRating(playerRatings, passer.id, 0.12)
    nudgeRating(playerRatings, runner.id, 0.1)
    events.push({
      minute,
      type: 'shift',
      teamId: attacking.club.id,
      playerName: passer.name,
      text: `${passer.name} slides a pass behind the high line for ${runner.name}.`,
    })
    events.push(resolveShot(minute, attacking, defending, random, ledger, playerRatings, 'behind-line', runner))
    frames.push(
      createTraceFrame({
        minute: minute + 0.58,
        phase: 'shot',
        homeId,
        attacking,
        defending,
        ball: { x: 89, y: lane + randomRange(random, -10, 10) },
        possessingTeamId: attacking.club.id,
        focusPlayerId: runner.id,
        note: `${runner.name} runs beyond the line`,
      }),
    )
    return { events, frames, nextControlId: defending.club.id }
  }

  const finalThirdDuel = battle(
    attacking.lineBreaking + attacking.tactic.tempo * 0.13 + attacking.captainLift + momentum,
    defending.boxDefense * 0.5 + defending.recovery * 0.34 + defensiveLineControl(defending.tactic),
    random,
  )
  const leadDrag = Math.max(0, attackLog.goals - defendLog.goals)
  const entryVolumeDrag = Math.max(0, attackLog.finalThirdEntries - 16) * 1.4

  if (finalThirdDuel < -4) {
    const defender = pickWeighted(defending.defenders, random, (player) => player.tackling + player.marking + player.pace)
    nudgeRating(playerRatings, defender.id, 0.12)
    if (random() < 0.16) {
      events.push({
        minute,
        type: 'pressure',
        teamId: defending.club.id,
        playerName: defender.name,
        text: `${defender.name} steps out to kill the move before the box.`,
      })
    }
    frames.push(
      createTraceFrame({
        minute: minute + 0.56,
        phase: 'transition',
        homeId,
        attacking: defending,
        defending: attacking,
        ball: { x: 68, y: lane + randomRange(random, -8, 8) },
        possessingTeamId: defending.club.id,
        focusPlayerId: defender.id,
        note: `${defender.name} steps out`,
      }),
    )
    return { events, frames, nextControlId: defending.club.id }
  }

  if (finalThirdDuel < 17 + leadDrag * 4 + entryVolumeDrag) {
    if (random() < 0.16) {
      const carrier = pickWeighted(
        [...attacking.midfielders, ...attacking.forwards],
        random,
        (player) => player.dribbling + player.firstTouch + player.pace,
      )
      events.push({
        minute,
        type: 'pressure',
        teamId: attacking.club.id,
        playerName: carrier.name,
        text: `${carrier.name} carries ${attacking.club.shortName} into the final third, but the move stalls.`,
      })
    }
    frames.push(
      createTraceFrame({
        minute: minute + 0.56,
        phase: 'final-third',
        homeId,
        attacking,
        defending,
        ball: { x: 69, y: lane + randomRange(random, -12, 12) },
        possessingTeamId: attacking.club.id,
        note: `${attacking.club.shortName} recycle at the edge of pressure`,
      }),
    )
    return { events, frames, nextControlId: random() < 0.58 ? defending.club.id : attacking.club.id }
  }

  attackLog.boxEntries += 1
  attackLog.lineBreaks += 1
  attackLog.territory += 3
  frames.push(
    createTraceFrame({
      minute: minute + 0.62,
      phase: 'box',
      homeId,
      attacking,
      defending,
      ball: { x: 80, y: lane + randomRange(random, -14, 14) },
      possessingTeamId: attacking.club.id,
      note: `${attacking.club.shortName} enter the box`,
    }),
  )

  const boxDuel = battle(
    attacking.boxThreat + attacking.lineBreaking * 0.22 + mentalityEdge(attacking.tactic),
    defending.boxDefense + defending.keeperSkill * 0.16 + compactness(defending.tactic),
    random,
  )

  const shotVolumeDrag = Math.max(0, attackLog.shots - 12) * 2
  if (boxDuel < 19 + leadDrag * 5 + shotVolumeDrag) {
    const defender = pickWeighted(defending.defenders, random, (player) => player.tackling + player.heading + player.strength)
    nudgeRating(playerRatings, defender.id, 0.14)
    if (random() < 0.22) {
      events.push({
        minute,
        type: 'chance',
        teamId: attacking.club.id,
        playerName: pickAttacker(attacking, random, 'box').name,
        text: `${attacking.club.shortName} enter the box, but ${defender.name} blocks the cutback.`,
        xg: 0.04,
      })
    }
    frames.push(
      createTraceFrame({
        minute: minute + 0.74,
        phase: 'transition',
        homeId,
        attacking: defending,
        defending: attacking,
        ball: { x: 82, y: lane + randomRange(random, -12, 12) },
        possessingTeamId: defending.club.id,
        focusPlayerId: defender.id,
        note: `${defender.name} blocks the final action`,
      }),
    )
    return { events, frames, nextControlId: defending.club.id }
  }

  events.push(resolveShot(minute, attacking, defending, random, ledger, playerRatings, boxDuel > 34 + leadDrag * 5 ? 'clear' : 'box'))
  frames.push(
    createTraceFrame({
      minute: minute + 0.82,
      phase: 'shot',
      homeId,
      attacking,
      defending,
      ball: { x: 90, y: lane + randomRange(random, -10, 10) },
      possessingTeamId: attacking.club.id,
      note: `${attacking.club.shortName} get the shot away`,
    }),
  )
  return { events, frames, nextControlId: defending.club.id }
}

function resolveShot(
  minute: number,
  attacking: TeamProfile,
  defending: TeamProfile,
  random: () => number,
  ledger: MatchLedger,
  playerRatings: Record<string, number>,
  chanceType: 'box' | 'clear' | 'counter' | 'behind-line',
  forcedShooter?: Player,
): MatchEvent {
  const shooter = forcedShooter ?? pickAttacker(attacking, random, chanceType === 'counter' ? 'final-third' : 'box')
  const creator = pickWeighted(
    [...attacking.midfielders, ...attacking.forwards],
    random,
    (player) => player.passing + player.vision + player.decisions,
  )
  const attackingLog = ledger[attacking.club.id]
  const defendingLog = ledger[defending.club.id]
  const chanceBoost = chanceType === 'clear' ? 0.085 : chanceType === 'behind-line' ? 0.065 : chanceType === 'counter' ? 0.038 : 0
  const pressurePenalty = Math.max(0, defending.recovery - attacking.lineBreaking) * 0.0012
  const xg = clamp(
    0.031 +
      chanceBoost +
      (shooter.finishing - defending.boxDefense) * 0.0017 +
      (creator.passing - 58) * 0.0009 +
      attacking.tactic.tempo * 0.0005 -
      pressurePenalty +
      randomRange(random, -0.018, 0.026),
    0.018,
    0.32,
  )
  const onTargetProbability = clamp(0.21 + xg * 0.66 + shooter.composure * 0.0015 + shooter.morale * 0.0007, 0.18, 0.6)
  const onTarget = random() < onTargetProbability
  const finishingModifier = clamp(0.92 + shooter.finishing * 0.003 - defending.keeperSkill * 0.0019, 0.7, 1.18)
  const goalProbability = clamp((xg * finishingModifier) / onTargetProbability, 0.025, 0.72)
  const goal = onTarget && random() < goalProbability

  attackingLog.shots += 1
  attackingLog.xg += xg
  if (onTarget) attackingLog.shotsOnTarget += 1
  if (goal) attackingLog.goals += 1
  if (onTarget && !goal) defendingLog.saves += 1

  nudgeRating(playerRatings, shooter.id, goal ? 0.95 : onTarget ? 0.18 : 0.04)
  nudgeRating(playerRatings, creator.id, goal ? 0.34 : 0.08)
  if (onTarget && !goal) nudgeRating(playerRatings, defending.keeper.id, 0.24)
  if (goal) nudgeRating(playerRatings, defending.keeper.id, -0.16)

  return {
    minute,
    type: goal ? 'goal' : onTarget ? 'save' : 'chance',
    teamId: attacking.club.id,
    playerName: shooter.name,
    xg,
    text: shotText({ goal, onTarget, xg, chanceType, shooter, creator, defending }),
  }
}

function buildProfile(club: Club, tactic: Tactic, captainId?: string, starterIds?: string[]): TeamProfile {
  const lineup = getLineupPlayers(club, tactic, starterIds)
  const keeper = lineup.find((player) => player.position === 'GK') ?? lineup[0]
  const defenders = groupOrFallback(lineup, 'DEF')
  const midfielders = groupOrFallback(lineup, 'MID')
  const forwards = groupOrFallback(lineup, 'FWD')
  const captain = captainId ? lineup.find((player) => player.id === captainId) : undefined
  const morale = average(lineup.map((player) => player.morale))
  const fitness = average(lineup.map((player) => player.fitness))
  const captainLift = captain ? captain.leadership * 0.075 + (captain.personality === 'Leader' ? 2.5 : 0) : 0
  const highPressLoad = Math.max(0, tactic.pressing - 62) * 0.18 + Math.max(0, tactic.tempo - 64) * 0.12
  const volatileCount = lineup.filter((player) => player.personality === 'Volatile').length

  return {
    club,
    tactic,
    lineup,
    keeper,
    defenders,
    midfielders,
    forwards,
    buildUp:
      average(
        [...defenders, ...midfielders, keeper].map(
          (player) =>
            player.passing * 0.38 +
            player.firstTouch * 0.22 +
            player.vision * 0.18 +
            player.decisions * 0.14 +
            player.composure * 0.08 +
            player.morale * 0.1 +
            player.form * 0.08,
        ),
      ) +
      captainLift * 0.22,
    pressResistance:
      average(
        lineup.map(
          (player) =>
            player.firstTouch * 0.32 +
            player.composure * 0.22 +
            player.strength * 0.14 +
            player.stamina * 0.14 +
            player.decisions * 0.1 +
            player.dribbling * 0.08 +
            player.morale * 0.14,
        ),
      ) +
      (tactic.mentality === 'Measured' ? 4 : 0),
    midfieldControl:
      average(
        midfielders.map(
          (player) =>
            player.passing * 0.26 +
            player.vision * 0.2 +
            player.decisions * 0.18 +
            player.tackling * 0.15 +
            player.stamina * 0.12 +
            player.workRate * 0.09,
        ),
      ) +
      morale * 0.05 +
      captainLift * 0.16,
    lineBreaking:
      average(
        [...midfielders, ...forwards].map(
          (player) => player.passing * 0.25 + player.vision * 0.2 + player.dribbling * 0.22 + player.pace * 0.18 + player.firstTouch * 0.15,
        ),
      ) +
      tactic.tempo * 0.09,
    boxThreat:
      average(
        forwards.map(
          (player) =>
            player.finishing * 0.4 +
            player.composure * 0.18 +
            player.dribbling * 0.18 +
            player.firstTouch * 0.12 +
            player.pace * 0.07 +
            player.heading * 0.05 +
            player.morale * 0.08,
        ),
      ) +
      mentalityEdge(tactic),
    boxDefense:
      average(
        [...defenders, keeper].map(
          (player) =>
            player.tackling * 0.3 +
            player.marking * 0.25 +
            player.heading * 0.16 +
            player.strength * 0.14 +
            player.pace * 0.08 +
            player.positioning * 0.07 +
            player.morale * 0.08,
        ),
      ) -
      Math.max(0, tactic.defensiveLine - 68) * 0.13,
    press:
      clamp(
        average(lineup.map((player) => player.stamina * 0.25 + player.workRate * 0.25 + player.decisions * 0.15)) * 0.28 +
          tactic.pressing * 0.62 +
          captainLift * 0.3 -
          highPressLoad * 0.35,
        20,
        94,
      ),
    recovery: average(
      [...defenders, ...midfielders].map(
        (player) => player.acceleration * 0.24 + player.pace * 0.24 + player.stamina * 0.25 + player.strength * 0.14 + player.tackling * 0.13,
      ),
    ),
    keeperSkill:
      keeper.handling * 0.42 +
      keeper.reflexes * 0.3 +
      keeper.positioning * 0.2 +
      keeper.oneOnOnes * 0.08 +
      keeper.composure * 0.05 +
      keeper.form * 0.06,
    disciplineRisk: clamp(tactic.pressing * 0.42 + volatileCount * 5 + (tactic.mentality === 'Relentless' ? 9 : 0) - morale * 0.12, 5, 85),
    fatigueRisk: clamp(highPressLoad + tactic.pressing * 0.24 + tactic.tempo * 0.16 - fitness * 0.3, 0, 70),
    captainLift,
  }
}

function buildReport(home: TeamProfile, away: TeamProfile, ledger: MatchLedger, possessionHome: number) {
  const homeLog = ledger[home.club.id]
  const awayLog = ledger[away.club.id]
  const winner = homeLog.goals === awayLog.goals ? undefined : homeLog.goals > awayLog.goals ? home : away
  const loser = winner?.club.id === home.club.id ? away : home
  const higherXg = homeLog.xg >= awayLog.xg ? home : away
  const higherXgLog = ledger[higherXg.club.id]
  const pressTeam = homeLog.pressWins >= awayLog.pressWins ? home : away
  const pressLog = ledger[pressTeam.club.id]
  const report = [
    winner
      ? `${winner.club.shortName} won because their best phases reached the box ${ledger[winner.club.id].boxEntries} times and produced ${roundOne(ledger[winner.club.id].xg)} xG.`
      : `The draw came from balanced chance quality: ${home.club.shortName} ${roundOne(homeLog.xg)} xG, ${away.club.shortName} ${roundOne(awayLog.xg)} xG.`,
    `${higherXg.club.shortName} had the better shot diet with ${higherXgLog.shots} shots and ${higherXgLog.boxEntries} box entries.`,
    `${pressTeam.club.shortName} created ${pressLog.pressWins} high regains from pressing, but that came with a fatigue risk of ${pressTeam.fatigueRisk.toFixed(0)}/70.`,
  ]

  const highLineVictim = homeLog.ballsBehind > awayLog.ballsBehind ? home : awayLog.ballsBehind > homeLog.ballsBehind ? away : undefined
  if (highLineVictim && ledger[highLineVictim.club.id].ballsBehind >= 2) {
    report.push(`${highLineVictim.club.shortName}'s defensive line was exposed by ${ledger[highLineVictim.club.id].ballsBehind} balls behind.`)
  } else if (loser && ledger[loser.club.id].buildUpFails >= 4) {
    report.push(`${loser.club.shortName} lost too many first-phase possessions and could not settle the match.`)
  } else if (Math.abs(possessionHome - 50) > 8) {
    const controlTeam = possessionHome > 50 ? home : away
    report.push(`${controlTeam.club.shortName} controlled the rhythm, but territory mattered more than possession alone.`)
  } else {
    report.push(`The match was decided by transition details more than raw possession.`)
  }

  return report
}

function finishRatings(home: TeamProfile, away: TeamProfile, ledger: MatchLedger, ratings: Record<string, number>) {
  applyResultRatings(home, away, ledger, ratings)
  applyResultRatings(away, home, ledger, ratings)

  for (const id of Object.keys(ratings)) {
    ratings[id] = Number(clamp(ratings[id], 5.1, 9.8).toFixed(1))
  }
}

function applyResultRatings(profile: TeamProfile, opponent: TeamProfile, ledger: MatchLedger, ratings: Record<string, number>) {
  const own = ledger[profile.club.id]
  const against = ledger[opponent.club.id]
  const resultBoost = own.goals > against.goals ? 0.28 : own.goals === against.goals ? 0.04 : -0.22
  const cleanDefenseBoost = against.goals === 0 ? 0.18 : -against.goals * 0.06

  for (const player of profile.lineup) {
    const roleBoost =
      player.position === 'GK'
        ? own.saves * 0.05 + cleanDefenseBoost
        : player.position === 'DEF'
          ? cleanDefenseBoost + own.lineBreaks * 0.005
          : player.position === 'MID'
            ? own.midfieldWins * 0.025 + own.finalThirdEntries * 0.004
            : own.goals * 0.08 + own.shots * 0.008
    ratings[player.id] = (ratings[player.id] ?? baseRating(player)) + resultBoost + roleBoost
  }
}

function seedRatings(home: TeamProfile, away: TeamProfile) {
  const ratings: Record<string, number> = {}
  for (const player of [...home.lineup, ...away.lineup]) {
    ratings[player.id] = baseRating(player)
  }
  return ratings
}

function baseRating(player: Player) {
  return 5.85 + playerOverall(player) * 0.012
}

function createLedger(home: TeamProfile, away: TeamProfile): MatchLedger {
  return {
    [home.club.id]: blankLedger(),
    [away.club.id]: blankLedger(),
  }
}

function blankLedger(): TeamLedger {
  return {
    possessions: 0,
    finalThirdEntries: 0,
    boxEntries: 0,
    shots: 0,
    shotsOnTarget: 0,
    goals: 0,
    xg: 0,
    pressWins: 0,
    buildUpFails: 0,
    midfieldWins: 0,
    lineBreaks: 0,
    ballsBehind: 0,
    counters: 0,
    saves: 0,
    cards: 0,
    territory: 0,
    lateFatigueLosses: 0,
  }
}

function createTraceFrame({
  minute,
  phase,
  homeId,
  attacking,
  defending,
  ball,
  possessingTeamId,
  focusPlayerId,
  note,
}: {
  minute: number
  phase: MatchPhase
  homeId: string
  attacking: TeamProfile
  defending: TeamProfile
  ball: { x: number; y: number }
  possessingTeamId?: string
  focusPlayerId?: string
  note?: string
}): MatchFrame {
  const attackDirection = attacking.club.id === homeId ? 1 : -1
  return {
    minute: Number(minute.toFixed(2)),
    phase,
    possessingTeamId,
    ball: orientBall(ball, attackDirection),
    players: [
      ...traceTeamPlayers({
        profile: attacking,
        phase,
        attacking: true,
        attackDirection,
        ball,
        focusPlayerId,
      }),
      ...traceTeamPlayers({
        profile: defending,
        phase,
        attacking: false,
        attackDirection,
        ball,
        focusPlayerId,
      }),
    ],
    note,
  }
}

function traceTeamPlayers({
  profile,
  phase,
  attacking,
  attackDirection,
  ball,
  focusPlayerId,
}: {
  profile: TeamProfile
  phase: MatchPhase
  attacking: boolean
  attackDirection: 1 | -1
  ball: { x: number; y: number }
  focusPlayerId?: string
}) {
  return profile.lineup.map((player, index) => {
    const roleIndex = roleIndexFor(profile.lineup, player)
    const roleCount = profile.lineup.filter((item) => item.position === player.position).length
    const lane = laneFor(roleIndex, roleCount)
    const intent = playerIntent({ player, profile, phase, attacking, focusPlayerId })
    const baseX = roleX({ position: player.position, phase, attacking, tactic: profile.tactic })
    const ballPull = movementPull({ player, profile, phase, attacking, intent })
    const runTarget = intentTarget({ baseX, lane, ball, intent, attacking })
    const x = clamp(baseX + (ball.x - baseX) * ballPull.x, 5, 95)
    const y = clamp(lane + (ball.y - lane) * ballPull.y, 9, 91)

    return {
      id: player.id,
      teamId: profile.club.id,
      name: player.name,
      number: index + 1,
      position: player.position,
      x: orientX(x, attackDirection),
      y,
      targetX: orientX(runTarget.x, attackDirection),
      targetY: runTarget.y,
      intent,
    }
  })
}

function roleX({
  position,
  phase,
  attacking,
  tactic,
}: {
  position: PlayerPosition
  phase: MatchPhase
  attacking: boolean
  tactic: Tactic
}) {
  const attackShape: Record<PlayerPosition, Record<MatchPhase, number>> = {
    GK: { build: 8, midfield: 10, 'final-third': 12, box: 13, transition: 12, shot: 12 },
    DEF: { build: 21, midfield: 30, 'final-third': 39, box: 44, transition: 34, shot: 45 },
    MID: { build: 36, midfield: 49, 'final-third': 63, box: 70, transition: 52, shot: 72 },
    FWD: { build: 56, midfield: 66, 'final-third': 78, box: 86, transition: 70, shot: 88 },
  }
  const defendShape: Record<PlayerPosition, Record<MatchPhase, number>> = {
    GK: { build: 94, midfield: 94, 'final-third': 94, box: 95, transition: 92, shot: 96 },
    DEF: { build: 67, midfield: 74, 'final-third': 83, box: 88, transition: 76, shot: 90 },
    MID: { build: 46, midfield: 55, 'final-third': 66, box: 72, transition: 59, shot: 75 },
    FWD: { build: 29, midfield: 38, 'final-third': 48, box: 55, transition: 42, shot: 58 },
  }
  const shape = attacking ? attackShape : defendShape
  const highLineAdjustment = !attacking && position === 'DEF' ? (tactic.defensiveLine - 58) * -0.14 : 0
  const pressAdjustment = !attacking && (position === 'FWD' || position === 'MID') ? (tactic.pressing - 55) * -0.08 : 0
  const ambitionAdjustment = attacking && position === 'FWD' && tactic.mentality === 'Relentless' ? 3 : 0
  return shape[position][phase] + highLineAdjustment + pressAdjustment + ambitionAdjustment
}

function playerIntent({
  player,
  profile,
  phase,
  attacking,
  focusPlayerId,
}: {
  player: Player
  profile: TeamProfile
  phase: MatchPhase
  attacking: boolean
  focusPlayerId?: string
}): PlayerIntent {
  if (player.id === focusPlayerId) {
    if (phase === 'shot') return 'shoot'
    if (phase === 'transition') return attacking ? 'run' : 'press'
    return attacking ? 'run' : 'press'
  }

  if (attacking) {
    if (phase === 'shot' && player.position === 'FWD') return 'shoot'
    if ((phase === 'final-third' || phase === 'box') && player.position === 'FWD') return 'run'
    if (player.position === 'MID') return 'support'
    if (phase === 'build' && player.position === 'DEF') return 'hold'
    return 'support'
  }

  if (phase === 'transition') return 'recover'
  if (profile.tactic.pressing > 64 && (player.position === 'FWD' || player.position === 'MID') && phase !== 'box') return 'press'
  if (player.position === 'DEF' || player.position === 'MID') return 'mark'
  return 'recover'
}

function movementPull({
  player,
  profile,
  phase,
  attacking,
  intent,
}: {
  player: Player
  profile: TeamProfile
  phase: MatchPhase
  attacking: boolean
  intent: PlayerIntent
}) {
  if (intent === 'press') return { x: clamp(profile.tactic.pressing / 230, 0.18, 0.42), y: 0.32 }
  if (intent === 'run') return { x: 0.08 + (player.pace + player.acceleration) * 0.0005, y: 0.12 }
  if (intent === 'shoot') return { x: 0.18, y: 0.26 }
  if (intent === 'support') return { x: attacking ? 0.12 : 0.08, y: 0.18 }
  if (intent === 'mark') return { x: phase === 'box' ? 0.18 : 0.1, y: 0.2 }
  if (intent === 'recover') return { x: 0.13, y: 0.18 }
  return { x: 0.04, y: 0.08 }
}

function intentTarget({
  baseX,
  lane,
  ball,
  intent,
  attacking,
}: {
  baseX: number
  lane: number
  ball: { x: number; y: number }
  intent: PlayerIntent
  attacking: boolean
}) {
  if (intent === 'run') return { x: clamp(ball.x + (attacking ? 12 : -6), 5, 95), y: clamp(ball.y + (lane > 50 ? 8 : -8), 8, 92) }
  if (intent === 'press') return { x: ball.x, y: ball.y }
  if (intent === 'shoot') return { x: 94, y: 50 }
  if (intent === 'support') return { x: clamp(ball.x - 10, 5, 95), y: clamp(ball.y + (lane > ball.y ? 10 : -10), 8, 92) }
  if (intent === 'mark') return { x: clamp(ball.x + 5, 5, 95), y: clamp(ball.y + (lane > 50 ? 7 : -7), 8, 92) }
  if (intent === 'recover') return { x: baseX, y: lane }
  return { x: baseX, y: lane }
}

function possessionLane(profile: TeamProfile, random: () => number) {
  const widthBias = profile.tactic.formation === '4-3-3' ? 19 : profile.tactic.formation === '3-5-2' ? 12 : 15
  const side = random() < 0.5 ? -1 : 1
  return clamp(50 + side * randomRange(random, 4, widthBias) + randomRange(random, -6, 6), 18, 82)
}

function roleIndexFor(lineup: Player[], player: Player) {
  return lineup.filter((item) => item.position === player.position).findIndex((item) => item.id === player.id)
}

function laneFor(index: number, count: number) {
  if (count <= 1) return 50
  const start = 18
  const end = 82
  return start + ((end - start) / (count - 1)) * index
}

function orientBall(ball: { x: number; y: number }, direction: 1 | -1) {
  return {
    x: orientX(ball.x, direction),
    y: ball.y,
  }
}

function orientX(x: number, direction: 1 | -1) {
  return direction === 1 ? x : 100 - x
}

function calculatePossessionShare(home: TeamProfile, away: TeamProfile) {
  return clamp(
    51 +
      (home.midfieldControl - away.midfieldControl) * 0.22 +
      (home.pressResistance - away.press) * 0.08 -
      (home.tactic.tempo - 58) * 0.035 +
      (away.tactic.tempo - 58) * 0.025,
    36,
    64,
  )
}

function battle(attack: number, defense: number, random: () => number) {
  return attack - defense + randomRange(random, -17, 17)
}

function dangerousTurnoverChance(attacking: TeamProfile, defending: TeamProfile, minute: number) {
  const fatigue = minute > 70 ? Math.max(0, defending.fatigueRisk - attacking.fatigueRisk) * 0.002 : 0
  const mentality = attacking.tactic.mentality === 'Relentless' ? 0.05 : attacking.tactic.mentality === 'Assertive' ? 0.02 : 0
  return clamp(0.03 + attacking.press * 0.001 + mentality + fatigue, 0.025, 0.12)
}

function highLineBehindRisk(attacking: TeamProfile, defending: TeamProfile, random: () => number) {
  const highLine = Math.max(0, defending.tactic.defensiveLine - 62) * 0.016
  const paceGap = (average(attacking.forwards.map((player) => player.pace + player.acceleration)) * 0.5 - defending.recovery) * 0.006
  const directness = Math.max(0, attacking.tactic.tempo - 58) * 0.006
  return 0.18 + highLine + paceGap + directness + randomRange(random, -0.16, 0.18)
}

function maybeAddCard(events: MatchEvent[], profile: TeamProfile, random: () => number, ledger: TeamLedger) {
  const cardChance = clamp(profile.disciplineRisk * 0.006 + profile.fatigueRisk * 0.003, 0.04, 0.42)
  if (random() > cardChance) return
  const player = pickWeighted([...profile.midfielders, ...profile.defenders], random, (item) => item.defense + profile.tactic.pressing)
  ledger.cards += 1
  events.push({
    minute: clamp(Math.round(randomRange(random, 24, 84)), 1, 90),
    type: 'card',
    teamId: profile.club.id,
    playerName: player.name,
    text: `${player.name} is booked after ${profile.club.shortName}'s pressure arrives late.`,
  })
}

function ensureMatchHasEvents(events: MatchEvent[], home: TeamProfile, away: TeamProfile) {
  if (events.length) return
  const player = pickBest([...home.midfielders, ...away.midfielders], 'MID')
  events.push({
    minute: 45,
    type: 'shift',
    teamId: home.club.id,
    playerName: player.name,
    text: `The first half stays compact, with both midfields denying clean entries.`,
  })
}

function territoryShare(home: TeamLedger, away: TeamLedger) {
  const total = home.territory + away.territory || 1
  return Math.round((home.territory / total) * 100)
}

function shotText({
  goal,
  onTarget,
  xg,
  chanceType,
  shooter,
  creator,
  defending,
}: {
  goal: boolean
  onTarget: boolean
  xg: number
  chanceType: 'box' | 'clear' | 'counter' | 'behind-line'
  shooter: Player
  creator: Player
  defending: TeamProfile
}) {
  const chanceLabel = xg > 0.24 ? 'clear chance' : xg > 0.12 ? 'good look' : 'half chance'
  const route =
    chanceType === 'counter'
      ? `after a transition`
      : chanceType === 'behind-line'
        ? `after beating ${defending.club.shortName}'s line`
        : `from ${creator.name}'s final pass`

  if (goal) return `${shooter.name} scores a ${chanceLabel} ${route}.`
  if (onTarget) return `${shooter.name} makes the keeper work with a ${chanceLabel} ${route}.`
  return `${shooter.name} gets a ${chanceLabel} ${route}, but misses.`
}

function pickAttacker(profile: TeamProfile, random: () => number, phase: Phase) {
  if (phase === 'box') {
    return pickWeighted(
      profile.forwards,
      random,
      (player) => player.finishing * 1.2 + player.composure + player.dribbling + player.morale * 0.25,
    )
  }
  return pickWeighted(
    [...profile.forwards, ...profile.midfielders],
    random,
    (player) => player.dribbling + player.pace + player.passing * 0.4,
  )
}

function pickWeighted(players: Player[], random: () => number, score: (player: Player) => number) {
  const pool = players.length ? players : []
  const total = pool.reduce((sum, player) => sum + Math.max(1, score(player)), 0)
  let cursor = random() * total

  for (const player of pool) {
    cursor -= Math.max(1, score(player))
    if (cursor <= 0) return player
  }

  return pool[0]
}

function pickBest(players: Player[], preferred: Player['position']) {
  return [...players]
    .filter((player) => player.position === preferred)
    .sort((left, right) => playerScore(right) - playerScore(left))[0]
}

function groupOrFallback(lineup: Player[], position: Player['position']) {
  const group = lineup.filter((player) => player.position === position)
  return group.length ? group : lineup
}

function average(values: number[]) {
  if (!values.length) return 0
  return values.reduce((sum, value) => sum + value, 0) / values.length
}

function nudgeRating(ratings: Record<string, number>, playerId: string, delta: number) {
  ratings[playerId] = (ratings[playerId] ?? 6.2) + delta
}

function tempoCalm(tactic: Tactic) {
  return tactic.tempo < 50 ? 5 : tactic.tempo > 72 ? -4 : 0
}

function tempoControl(tactic: Tactic) {
  return tactic.tempo < 50 ? 4 : tactic.tempo > 70 ? -2 : 1
}

function pressAggression(tactic: Tactic) {
  return tactic.mentality === 'Relentless' ? 5 : tactic.mentality === 'Assertive' ? 2 : -1
}

function defensiveBlock(tactic: Tactic) {
  return tactic.mentality === 'Measured' ? 4 : tactic.defensiveLine < 50 ? 3 : 0
}

function defensiveLineControl(tactic: Tactic) {
  return tactic.defensiveLine < 50 ? 4 : tactic.defensiveLine > 70 ? -3 : 0
}

function compactness(tactic: Tactic) {
  return tactic.mentality === 'Measured' ? 4 : tactic.defensiveLine < 48 ? 5 : 0
}

function mentalityEdge(tactic: Tactic) {
  return tactic.mentality === 'Relentless' ? 5 : tactic.mentality === 'Assertive' ? 2 : -1
}

function getClub(state: CareerState, clubId: string) {
  const club = state.clubs.find((item) => item.id === clubId)
  if (!club) throw new Error(`Missing club ${clubId}`)
  return club
}

function roundOne(value: number) {
  return Math.round(value * 10) / 10
}
