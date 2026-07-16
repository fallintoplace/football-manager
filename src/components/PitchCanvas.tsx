import { useEffect, useMemo, useRef } from 'react'
import { createAiTactic } from '../game/data'
import { getClub } from '../game/career'
import { formationRequirements } from '../game/lineup'
import type { CareerState, Club, MatchFrame, MatchResult, PlayerPosition, Tactic, TracePlayer } from '../game/types'

type PitchCanvasProps = {
  state: CareerState
  result?: MatchResult
}

type Dot = {
  x: number
  y: number
  position: PlayerPosition
}

const pitchWidth = 960
const pitchHeight = 540

export function PitchCanvas({ state, result }: PitchCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const teams = useMemo(() => {
    if (!result) {
      const selected = getClub(state, state.selectedClubId)
      return {
        home: selected,
        away: opponentForPreview(state),
        homeTactic: state.tactic,
        awayTactic: createAiTactic(opponentForPreview(state)),
      }
    }

    const home = getClub(state, result.homeId)
    const away = getClub(state, result.awayId)
    return {
      home,
      away,
      homeTactic: result.homeId === state.selectedClubId ? state.tactic : createAiTactic(home),
      awayTactic: result.awayId === state.selectedClubId ? state.tactic : createAiTactic(away),
    }
  }, [result, state])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const context = canvas.getContext('2d')
    if (!context) return
    let frame = 0
    let animation = 0

    const draw = () => {
      frame += 1
      const replay = result?.trace?.length ? getReplayFrame(result.trace, frame) : undefined
      const minute = replay ? replay.minute : result ? ((frame / 3) % 90) + 1 : 1
      const phase = Math.sin(frame / 24)
      drawPitch(context)
      drawScoreline(context, teams.home, teams.away, result, minute)
      if (replay) {
        drawTrace(context, replay, teams.home, teams.away)
      } else {
        drawTeam(context, teams.home, teams.homeTactic, false, phase, minute)
        drawTeam(context, teams.away, teams.awayTactic, true, -phase, minute)
        drawBall(context, result, minute, teams.home.id)
      }
      animation = window.requestAnimationFrame(draw)
    }

    draw()
    return () => window.cancelAnimationFrame(animation)
  }, [result, teams])

  return (
    <canvas
      ref={canvasRef}
      width={pitchWidth}
      height={pitchHeight}
      className="pitch-canvas"
      aria-label="Match visualization"
    />
  )
}

function getReplayFrame(trace: MatchFrame[], frame: number): MatchFrame {
  const cursor = (frame / 18) % trace.length
  const index = Math.floor(cursor)
  const nextIndex = (index + 1) % trace.length
  const progress = cursor - index
  return blendFrames(trace[index], trace[nextIndex], progress)
}

function blendFrames(current: MatchFrame, next: MatchFrame, progress: number): MatchFrame {
  const nextPlayers = new Map(next.players.map((player) => [player.id, player]))
  return {
    ...current,
    minute: current.minute + (next.minute - current.minute) * progress,
    ball: {
      x: lerp(current.ball.x, next.ball.x, progress),
      y: lerp(current.ball.y, next.ball.y, progress),
    },
    players: current.players.map((player) => {
      const nextPlayer = nextPlayers.get(player.id) ?? player
      return {
        ...player,
        x: lerp(player.x, nextPlayer.x, progress),
        y: lerp(player.y, nextPlayer.y, progress),
        targetX: lerp(player.targetX, nextPlayer.targetX, progress),
        targetY: lerp(player.targetY, nextPlayer.targetY, progress),
        intent: nextPlayer.intent,
      }
    }),
  }
}

function drawTrace(context: CanvasRenderingContext2D, frame: MatchFrame, home: Club, away: Club) {
  const ball = toPitchPoint(frame.ball.x, frame.ball.y)

  drawIntentLines(context, frame.players, home, away)
  for (const player of frame.players) {
    drawTracePlayer(context, player, player.teamId === home.id ? home : away, ball)
  }
  drawTraceBall(context, ball.x, ball.y)
  drawPhaseBadge(context, frame)
}

function drawIntentLines(context: CanvasRenderingContext2D, players: TracePlayer[], home: Club, away: Club) {
  context.save()
  context.lineWidth = 1.5
  for (const player of players) {
    if (player.intent === 'hold' || player.intent === 'mark') continue
    const start = toPitchPoint(player.x, player.y)
    const end = toPitchPoint(player.targetX, player.targetY)
    const club = player.teamId === home.id ? home : away
    context.strokeStyle = player.intent === 'press' || player.intent === 'recover' ? 'rgba(255,255,255,0.24)' : club.colors[1]
    context.globalAlpha = player.intent === 'shoot' ? 0.62 : 0.34
    context.beginPath()
    context.moveTo(start.x, start.y)
    context.lineTo(end.x, end.y)
    context.stroke()
  }
  context.restore()
}

function drawTracePlayer(context: CanvasRenderingContext2D, player: TracePlayer, club: Club, ball: { x: number; y: number }) {
  const point = toPitchPoint(player.x, player.y)
  const distanceToBall = Math.hypot(point.x - ball.x, point.y - ball.y)
  const radius = player.intent === 'shoot' ? 14 : distanceToBall < 42 ? 13 : 11

  context.beginPath()
  context.fillStyle = club.colors[1]
  context.arc(point.x, point.y, radius + 4, 0, Math.PI * 2)
  context.fill()
  context.beginPath()
  context.fillStyle = club.colors[0]
  context.arc(point.x, point.y, radius, 0, Math.PI * 2)
  context.fill()
  context.fillStyle = '#ffffff'
  context.font = '800 10px Inter, system-ui, sans-serif'
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(String(player.number), point.x, point.y + 0.5)

  if (player.intent === 'press' || player.intent === 'shoot') {
    context.strokeStyle = player.intent === 'shoot' ? 'rgba(255,255,255,0.86)' : 'rgba(255,255,255,0.5)'
    context.lineWidth = 2
    context.beginPath()
    context.arc(point.x, point.y, radius + 7, 0, Math.PI * 2)
    context.stroke()
  }
}

function drawTraceBall(context: CanvasRenderingContext2D, x: number, y: number) {
  context.beginPath()
  context.fillStyle = 'rgba(0,0,0,0.2)'
  context.arc(x + 3, y + 4, 9, 0, Math.PI * 2)
  context.fill()
  context.beginPath()
  context.fillStyle = '#f8faf7'
  context.arc(x, y, 8, 0, Math.PI * 2)
  context.fill()
  context.strokeStyle = 'rgba(20, 24, 22, 0.7)'
  context.lineWidth = 2
  context.stroke()
}

function drawPhaseBadge(context: CanvasRenderingContext2D, frame: MatchFrame) {
  context.fillStyle = 'rgba(20, 27, 25, 0.82)'
  roundedRect(context, 26, pitchHeight - 68, 390, 42, 8)
  context.fill()
  context.fillStyle = '#f8fff8'
  context.font = '800 13px Inter, system-ui, sans-serif'
  context.fillText(`${frame.phase.toUpperCase()} · ${Math.floor(frame.minute)}'`, 44, pitchHeight - 43)
  if (frame.note) {
    context.fillStyle = '#bedfc7'
    context.font = '600 12px Inter, system-ui, sans-serif'
    context.fillText(frame.note.slice(0, 42), 168, pitchHeight - 43)
  }
}

function drawPitch(context: CanvasRenderingContext2D) {
  context.clearRect(0, 0, pitchWidth, pitchHeight)
  context.fillStyle = '#25784d'
  context.fillRect(0, 0, pitchWidth, pitchHeight)

  for (let stripe = 0; stripe < 10; stripe += 1) {
    context.fillStyle = stripe % 2 === 0 ? 'rgba(255,255,255,0.035)' : 'rgba(0,0,0,0.035)'
    context.fillRect((pitchWidth / 10) * stripe, 0, pitchWidth / 10, pitchHeight)
  }

  context.strokeStyle = 'rgba(238, 252, 235, 0.82)'
  context.lineWidth = 3
  context.strokeRect(30, 30, pitchWidth - 60, pitchHeight - 60)
  context.beginPath()
  context.moveTo(pitchWidth / 2, 30)
  context.lineTo(pitchWidth / 2, pitchHeight - 30)
  context.stroke()
  context.beginPath()
  context.arc(pitchWidth / 2, pitchHeight / 2, 66, 0, Math.PI * 2)
  context.stroke()
  context.strokeRect(30, pitchHeight / 2 - 108, 128, 216)
  context.strokeRect(pitchWidth - 158, pitchHeight / 2 - 108, 128, 216)
  context.strokeRect(30, pitchHeight / 2 - 54, 58, 108)
  context.strokeRect(pitchWidth - 88, pitchHeight / 2 - 54, 58, 108)
}

function drawScoreline(
  context: CanvasRenderingContext2D,
  home: Club,
  away: Club,
  result: MatchResult | undefined,
  minute: number,
) {
  context.fillStyle = 'rgba(20, 27, 25, 0.82)'
  roundedRect(context, 26, 22, 292, 46, 8)
  context.fill()
  context.font = '700 18px Inter, system-ui, sans-serif'
  context.fillStyle = '#f8fff8'
  const score = result ? `${result.homeGoals} - ${result.awayGoals}` : 'v'
  context.fillText(`${home.shortName} ${score} ${away.shortName}`, 44, 51)
  context.font = '600 13px Inter, system-ui, sans-serif'
  context.fillStyle = '#bedfc7'
  context.fillText(result ? `${Math.floor(minute)}' replay` : 'next fixture shape', 214, 51)
}

function drawTeam(
  context: CanvasRenderingContext2D,
  club: Club,
  tactic: Tactic,
  reverse: boolean,
  phase: number,
  minute: number,
) {
  const dots = formationDots(tactic, reverse)
  const primary = club.colors[0]
  const secondary = club.colors[1]
  const minuteWave = Math.sin(minute / 7) * 5

  dots.forEach((dot, index) => {
    const x = dot.x + phase * laneDrift(dot.position) + Math.sin((index + minute) / 8) * 4
    const y = dot.y + minuteWave + Math.cos((index * 2 + minute) / 9) * 5
    context.beginPath()
    context.fillStyle = secondary
    context.arc(x, y, 13, 0, Math.PI * 2)
    context.fill()
    context.beginPath()
    context.fillStyle = primary
    context.arc(x, y, 9, 0, Math.PI * 2)
    context.fill()
    context.fillStyle = '#ffffff'
    context.font = '700 10px Inter, system-ui, sans-serif'
    context.textAlign = 'center'
    context.textBaseline = 'middle'
    context.fillText(String(index + 1), x, y + 0.5)
  })

  context.textAlign = 'left'
  context.textBaseline = 'alphabetic'
}

function drawBall(context: CanvasRenderingContext2D, result: MatchResult | undefined, minute: number, homeId: string) {
  const event = result?.events.reduce((closest, item) => {
    if (!closest) return item
    return Math.abs(item.minute - minute) < Math.abs(closest.minute - minute) ? item : closest
  }, result.events[0])
  const eventPressure = event ? Math.max(0, 1 - Math.abs(event.minute - minute) / 18) : 0
  const attackingHome = event ? event.teamId === homeId : Math.sin(minute / 10) > 0
  const flow = (Math.sin(minute / 8) + 1) / 2
  const x = eventPressure
    ? attackingHome
      ? 642 + flow * 168
      : 318 - flow * 168
    : 420 + Math.sin(minute / 9) * 120
  const y = eventPressure ? pitchHeight / 2 + Math.cos(minute / 6) * 72 : pitchHeight / 2 + Math.cos(minute / 8) * 112

  context.beginPath()
  context.fillStyle = '#f8faf7'
  context.arc(x, y, 8, 0, Math.PI * 2)
  context.fill()
  context.strokeStyle = 'rgba(20, 24, 22, 0.7)'
  context.lineWidth = 2
  context.stroke()
}

function formationDots(tactic: Tactic, reverse: boolean): Dot[] {
  const requirements = formationRequirements(tactic.formation)
  const xColumns = reverse ? [888, 720, 476, 232] : [72, 240, 484, 728]

  return [
    ...lineDots(1, xColumns[0], 'GK'),
    ...lineDots(requirements.DEF, xColumns[1], 'DEF'),
    ...lineDots(requirements.MID, xColumns[2], 'MID'),
    ...lineDots(requirements.FWD, xColumns[3], 'FWD'),
  ]
}

function toPitchPoint(x: number, y: number) {
  return {
    x: 30 + (pitchWidth - 60) * (x / 100),
    y: 30 + (pitchHeight - 60) * (y / 100),
  }
}

function lerp(start: number, end: number, progress: number) {
  return start + (end - start) * progress
}

function lineDots(count: number, x: number, position: PlayerPosition): Dot[] {
  const gap = 330 / Math.max(1, count - 1)
  const start = count === 1 ? pitchHeight / 2 : pitchHeight / 2 - 165

  return Array.from({ length: count }, (_, index) => ({
    x,
    y: count === 1 ? start : start + gap * index,
    position,
  }))
}

function laneDrift(position: PlayerPosition) {
  if (position === 'FWD') return 18
  if (position === 'MID') return 12
  if (position === 'DEF') return 7
  return 2
}

function roundedRect(context: CanvasRenderingContext2D, x: number, y: number, width: number, height: number, radius: number) {
  context.beginPath()
  context.moveTo(x + radius, y)
  context.arcTo(x + width, y, x + width, y + height, radius)
  context.arcTo(x + width, y + height, x, y + height, radius)
  context.arcTo(x, y + height, x, y, radius)
  context.arcTo(x, y, x + width, y, radius)
  context.closePath()
}

function opponentForPreview(state: CareerState) {
  const fixture = state.fixtures[state.roundIndex]?.find(
    (item) => item.homeId === state.selectedClubId || item.awayId === state.selectedClubId,
  )

  if (!fixture) return state.clubs.find((club) => club.id !== state.selectedClubId) ?? state.clubs[0]
  return getClub(state, fixture.homeId === state.selectedClubId ? fixture.awayId : fixture.homeId)
}
