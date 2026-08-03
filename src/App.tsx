import type { CSSProperties, ReactNode } from 'react'
import { useEffect, useMemo, useState } from 'react'
import {
  Activity,
  AlertTriangle,
  BarChart3,
  CalendarDays,
  CheckCircle2,
  CircleDot,
  ClipboardList,
  Database,
  Dumbbell,
  ExternalLink,
  FastForward,
  Gauge,
  ListChecks,
  MapPin,
  Play,
  RotateCcw,
  Save,
  Settings2,
  Shield,
  Target,
  Trophy,
  UserCheck,
  Users,
} from 'lucide-react'
import './App.css'
import { PitchCanvas } from './components/PitchCanvas'
import {
  advanceMatchdayWithResults,
  changeSelectedClub,
  getClub,
  getLastMatch,
  getSortedStandings,
  getUpcomingFixture,
  loadCareer,
  recommendStarters,
  resetCareer,
  saveCareer,
  simulateSeasonWithResults,
  teamAverage,
  toggleStarter,
  updateCaptain,
  updateTactic,
  updateTraining,
} from './game/career'
import { buildAnalyticsPayload, clickhouseUiUrl, fetchActionInsights, fetchAnalyticsRuns, fetchAnalyticsSummary, fetchAnalyticsTimeline, fetchIcebergHistory, fetchRealDataMatch, fetchRealDataMatches, fetchSeasonComparison, fetchTacticalMatchups, fetchWorldCup2026Overview, ingestMatchday, seasonRunId } from './game/analytics'
import { createAiTactic, createInitialCareer } from './game/data'
import { playerAttributeSummary, playerOverall, playerScore, validateLineup } from './game/lineup'
import type {
  AnalyticsDevelopmentPlayer,
  ActionInsights,
  AnalyticsPlayer,
  AnalyticsRun,
  AnalyticsSeasonComparison,
  AnalyticsSummary,
  AnalyticsSyncStatus,
  AnalyticsTimeline,
  AnalyticsTimelinePoint,
  AnalyticsTimelineStanding,
  IcebergHistory,
  RealMatchExplorer,
  RealMatchSummary,
  RealPassNetworkLink,
  RealPlayerProfile,
  RealShot,
  TacticalMatchup,
  WorldCup2026MatchSummary,
  WorldCup2026Overview,
} from './game/analytics'
import type {
  CareerState,
  Club,
  Formation,
  MatchResult,
  Mentality,
  PlayerPosition,
  SeasonAnalyticsSnapshot,
  Tactic,
  TrainingFocus,
} from './game/types'

type Tab = 'club' | 'squad' | 'tactics' | 'match' | 'league' | 'analytics'
type SeasonSyncProgress = { completed: number; total: number }

const tabs: { id: Tab; label: string; Icon: typeof Gauge }[] = [
  { id: 'club', label: 'Matchday', Icon: Gauge },
  { id: 'squad', label: 'Squad', Icon: Users },
  { id: 'tactics', label: 'Tactics', Icon: Settings2 },
  { id: 'match', label: 'Match', Icon: CircleDot },
  { id: 'league', label: 'League', Icon: Trophy },
  { id: 'analytics', label: 'Analytics', Icon: Database },
]

const formations: Formation[] = ['4-2-3-1', '4-3-3', '4-4-2', '3-5-2']
const mentalities: Mentality[] = ['Measured', 'Assertive', 'Relentless']
const trainingOptions: TrainingFocus[] = ['Structure', 'Finishing', 'Pressing', 'Recovery', 'Conditioning']
const positionOrder: PlayerPosition[] = ['GK', 'DEF', 'MID', 'FWD']

function App() {
  const [state, setState] = useState<CareerState>(() => loadCareer() ?? createInitialCareer())
  const [activeTab, setActiveTab] = useState<Tab>('club')
  const [analyticsStatus, setAnalyticsStatus] = useState<AnalyticsSyncStatus>('unknown')
  const [simulatingSeason, setSimulatingSeason] = useState(false)
  const [seasonSyncProgress, setSeasonSyncProgress] = useState<SeasonSyncProgress>()
  const [saveMessage, setSaveMessage] = useState('')

  const selectedClub = useMemo(() => getClub(state, state.selectedClubId), [state])
  const sortedStandings = useMemo(() => getSortedStandings(state.standings), [state.standings])
  const upcomingFixture = getUpcomingFixture(state)
  const lastMatch = getLastMatch(state)
  const seasonComplete = state.roundIndex >= state.fixtures.length
  const nextOpponent = upcomingFixture
    ? getClub(state, upcomingFixture.homeId === selectedClub.id ? upcomingFixture.awayId : upcomingFixture.homeId)
    : undefined
  const selectedRank = sortedStandings.findIndex((standing) => standing.clubId === selectedClub.id) + 1
  const selectedStanding = sortedStandings.find((standing) => standing.clubId === selectedClub.id)
  const lineup = useMemo(() => validateLineup(selectedClub, state.tactic, state.starterIds), [selectedClub, state.starterIds, state.tactic])
  const canPlay = seasonComplete || lineup.isLegal

  const clubStyle = {
    '--club-primary': selectedClub.colors[0],
    '--club-secondary': selectedClub.colors[1],
  } as CSSProperties

  function playMatchday() {
    if (!canPlay) {
      setActiveTab('club')
      return
    }
    const advanced = advanceMatchdayWithResults(state)
    setState(advanced.state)
    if (advanced.results.length) {
      setSeasonSyncProgress(undefined)
      setAnalyticsStatus('syncing')
      void ingestMatchday(buildAnalyticsPayload(advanced.state, advanced.results))
        .then(() => setAnalyticsStatus('online'))
        .catch(() => setAnalyticsStatus('offline'))
    }
    setActiveTab('match')
  }

  function simulateSeason() {
    if (!canPlay || seasonComplete || simulatingSeason) {
      if (!canPlay) setActiveTab('club')
      return
    }

    setSimulatingSeason(true)
    window.setTimeout(() => {
      const simulation = simulateSeasonWithResults(state)
      setState(simulation.state)
      setActiveTab('league')
      if (simulation.results.length) {
        setSeasonSyncProgress({ completed: 0, total: simulation.matchdays.length })
        setAnalyticsStatus('syncing')
        void ingestSeason(simulation.state, simulation.matchdays, simulation.snapshots, (completed, total) => {
          setSeasonSyncProgress({ completed, total })
        })
          .then(() => setAnalyticsStatus('online'))
          .catch(() => setAnalyticsStatus('offline'))
      }
      setSimulatingSeason(false)
    }, 0)
  }

  function save() {
    try {
      setState((current) => saveCareer(current))
      setSaveMessage('Saved locally')
    } catch {
      setSaveMessage('Save failed: browser storage is full')
    }
    window.setTimeout(() => setSaveMessage(''), 2400)
  }

  function reset() {
    setState(resetCareer())
    setActiveTab('club')
  }

  return (
    <main className="app-shell" style={clubStyle}>
      <header className="topbar">
        <div className="club-lockup">
          <ClubBadge club={selectedClub} size="lg" />
          <div>
            <p className="eyebrow">Season {state.season}</p>
            <h1>{selectedClub.name}</h1>
          </div>
        </div>

        <div className="header-actions">
          <label className="club-select">
            <span>Club</span>
            <select value={state.selectedClubId} onChange={(event) => setState(changeSelectedClub(state, event.target.value))}>
              {state.clubs.map((club) => (
                <option value={club.id} key={club.id}>
                  {club.name}
                </option>
              ))}
            </select>
          </label>
          <IconButton label="Save" onClick={save}>
            <Save size={18} />
          </IconButton>
          {saveMessage && <span className="save-feedback">{saveMessage}</span>}
          <IconButton label="Reset" onClick={reset}>
            <RotateCcw size={18} />
          </IconButton>
        </div>
      </header>

      <section className="status-strip" aria-label="Club status">
        <Metric icon={<Trophy size={17} />} label="Rank" value={selectedRank ? `#${selectedRank}` : '-'} />
        <Metric icon={<BarChart3 size={17} />} label="Points" value={selectedStanding?.points ?? 0} />
        <Metric icon={<Users size={17} />} label="Board" value={`${selectedClub.boardTrust}%`} />
        <Metric icon={<Activity size={17} />} label="Fans" value={`${selectedClub.fanMood}%`} />
        <Metric icon={<Gauge size={17} />} label="Squad" value={teamAverage(selectedClub)} />
        <Metric icon={<CalendarDays size={17} />} label="Matchday" value={`${Math.min(state.roundIndex + 1, state.fixtures.length)}/${state.fixtures.length}`} />
      </section>

      <nav className="tabbar" aria-label="Main views">
        {tabs.map(({ id, label, Icon }) => (
          <button className={activeTab === id ? 'tab active' : 'tab'} type="button" onClick={() => setActiveTab(id)} key={id}>
            <Icon size={18} />
            <span>{label}</span>
          </button>
        ))}
      </nav>

      {activeTab === 'club' && (
        <ClubView
          state={state}
          selectedClub={selectedClub}
          nextOpponent={nextOpponent}
          upcomingLabel={upcomingFixture ? fixtureLabel(state, upcomingFixture.homeId, upcomingFixture.awayId) : 'Season review'}
          seasonComplete={seasonComplete}
          lineup={lineup}
          tactic={state.tactic}
          captainId={state.captainId}
          canPlay={canPlay}
          simulatingSeason={simulatingSeason}
          onPlay={playMatchday}
          onSimSeason={simulateSeason}
          onTactic={(tactic) => setState((current) => updateTactic(current, tactic))}
          onTraining={(focus) => setState((current) => updateTraining(current, focus))}
          onCaptain={(playerId) => setState((current) => updateCaptain(current, playerId))}
          onToggleStarter={(playerId) => setState((current) => toggleStarter(current, playerId))}
          onRecommendLineup={() => setState((current) => recommendStarters(current))}
          lastMatch={lastMatch}
        />
      )}

      {activeTab === 'squad' && (
        <SquadView
          selectedClub={selectedClub}
          starterIds={state.starterIds}
          captainId={state.captainId}
          onCaptain={(playerId) => setState((current) => updateCaptain(current, playerId))}
          onToggleStarter={(playerId) => setState((current) => toggleStarter(current, playerId))}
        />
      )}

      {activeTab === 'tactics' && (
        <TacticsView tactic={state.tactic} trainingFocus={state.trainingFocus} onTactic={(tactic) => setState((current) => updateTactic(current, tactic))} />
      )}

      {activeTab === 'match' && <MatchView state={state} lastMatch={lastMatch} />}

      {activeTab === 'league' && <LeagueView state={state} standings={sortedStandings} />}

      {activeTab === 'analytics' && (
        <AnalyticsView
          clubId={selectedClub.id}
          selectedClub={selectedClub}
          clubs={state.clubs}
          runId={seasonRunId(state)}
          status={analyticsStatus}
          seasonSyncProgress={seasonSyncProgress}
          localResults={state.results}
          onStatusChange={setAnalyticsStatus}
          onOpenMatch={(matchId) => {
            setState((current) => ({ ...current, lastMatchId: matchId }))
            setActiveTab('match')
          }}
        />
      )}
    </main>
  )
}

function ClubBadge({
  club,
  size = 'md',
}: {
  club: Pick<Club, 'name' | 'badgeUrl' | 'colors'>
  size?: 'sm' | 'md' | 'lg'
}) {
  const [failed, setFailed] = useState(false)
  const iconSize = size === 'lg' ? 26 : size === 'md' ? 20 : 15

  return (
    <span className={`club-badge club-badge-${size}`} style={{ backgroundColor: club.colors[0], color: club.colors[1] }}>
      {failed ? <Shield size={iconSize} /> : <img src={club.badgeUrl} alt={`${club.name} badge`} onError={() => setFailed(true)} />}
    </span>
  )
}

async function ingestSeason(
  state: CareerState,
  matchdays: MatchResult[][],
  snapshots: SeasonAnalyticsSnapshot[],
  onProgress: (completed: number, total: number) => void,
) {
  for (const [index, results] of matchdays.entries()) {
    await ingestMatchday(buildAnalyticsPayload(state, results, snapshots[index]))
    onProgress(index + 1, matchdays.length)
  }
}

function ClubView({
  state,
  selectedClub,
  nextOpponent,
  upcomingLabel,
  seasonComplete,
  lineup,
  tactic,
  captainId,
  canPlay,
  simulatingSeason,
  onPlay,
  onSimSeason,
  onTactic,
  onTraining,
  onCaptain,
  onToggleStarter,
  onRecommendLineup,
  lastMatch,
}: {
  state: CareerState
  selectedClub: ReturnType<typeof getClub>
  nextOpponent?: ReturnType<typeof getClub>
  upcomingLabel: string
  seasonComplete: boolean
  lineup: ReturnType<typeof validateLineup>
  tactic: Tactic
  captainId: string
  canPlay: boolean
  simulatingSeason: boolean
  onPlay: () => void
  onSimSeason: () => void
  onTactic: (tactic: Partial<Tactic>) => void
  onTraining: (focus: TrainingFocus) => void
  onCaptain: (playerId: string) => void
  onToggleStarter: (playerId: string) => void
  onRecommendLineup: () => void
  lastMatch?: MatchResult
}) {
  const opponentTactic = nextOpponent ? createAiTactic(nextOpponent) : undefined
  const sortedStandings = getSortedStandings(state.standings)
  const opponentRank = nextOpponent ? sortedStandings.findIndex((standing) => standing.clubId === nextOpponent.id) + 1 : 0
  const captain = selectedClub.squad.find((player) => player.id === captainId)

  return (
    <section className="view-grid club-view matchday-hub">
      <section className="panel primary-panel command-panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Matchday Hub</p>
            <h2>{upcomingLabel}</h2>
          </div>
          <div className="command-actions">
            <button className="primary-action" type="button" onClick={onPlay} disabled={!canPlay}>
              <Play size={18} />
              <span>{seasonComplete ? 'Start Season' : 'Play Matchday'}</span>
            </button>
            <button
              className="secondary-action"
              type="button"
              onClick={onSimSeason}
              disabled={!canPlay || seasonComplete || simulatingSeason}
            >
              <FastForward size={18} />
              <span>{simulatingSeason ? 'Simulating…' : 'Sim Season'}</span>
            </button>
          </div>
        </div>

        <div className="fixture-line">
          <div className="fixture-team">
            <ClubBadge club={selectedClub} size="md" />
            <div>
              <span className="team-code">{selectedClub.shortName}</span>
              <strong>{selectedClub.name}</strong>
            </div>
          </div>
          <span className="versus">vs</span>
          <div className="fixture-team">
            {nextOpponent ? <ClubBadge club={nextOpponent} size="md" /> : <span className="club-badge club-badge-md"><Shield size={20} /></span>}
            <div>
              <span className="team-code">{nextOpponent?.shortName ?? 'END'}</span>
              <strong>{nextOpponent?.name ?? 'Season Complete'}</strong>
            </div>
          </div>
        </div>

        <div className="tempo-row">
          <ValueBar label="XI Score" value={lineup.averageScore} suffix={String(lineup.averageScore)} />
          <ValueBar label="Fitness" value={lineup.averageFitness} />
          <ValueBar label="Morale" value={lineup.averageMorale} />
        </div>

        <LineupReadiness lineup={lineup} captainName={captain?.name ?? 'No captain'} />
      </section>

      <section className="panel opponent-panel">
        <div className="section-heading compact">
          <div>
            <p className="eyebrow">Opponent</p>
            <h2>{nextOpponent?.name ?? 'Season Review'}</h2>
          </div>
          {nextOpponent ? <ClubBadge club={nextOpponent} size="md" /> : <Shield size={20} />}
        </div>
        {nextOpponent && opponentTactic ? (
          <div className="scout-brief">
            <Metric icon={<Trophy size={16} />} label="Rank" value={`#${opponentRank || '-'}`} />
            <Metric icon={<Gauge size={16} />} label="Squad" value={teamAverage(nextOpponent)} />
            <Metric icon={<Activity size={16} />} label="Style" value={nextOpponent.managerStyle} />
            <Metric icon={<BarChart3 size={16} />} label="Line" value={opponentTactic.defensiveLine} />
          </div>
        ) : (
          <p className="empty-state">The table resets after the season review.</p>
        )}
      </section>

      <section className="panel quick-plan-panel">
        <div className="section-heading compact">
          <div>
            <p className="eyebrow">Plan</p>
            <h2>{tactic.formation} · {tactic.mentality}</h2>
          </div>
          <ListChecks size={20} />
        </div>
        <TacticalQuickPlan tactic={tactic} trainingFocus={state.trainingFocus} onTactic={onTactic} onTraining={onTraining} />
      </section>

      <section className="panel wide-panel lineup-panel">
        <div className="section-heading compact">
          <div>
            <p className="eyebrow">Team Sheet</p>
            <h2>{lineup.starters.length}/11 Starters</h2>
          </div>
          <button className="secondary-action" type="button" onClick={onRecommendLineup}>
            <CheckCircle2 size={17} />
            <span>Recommend XI</span>
          </button>
        </div>

        <PositionCounts lineup={lineup} />
        <PlayerPicker
          club={selectedClub}
          starterIds={state.starterIds}
          captainId={captainId}
          onCaptain={onCaptain}
          onToggleStarter={onToggleStarter}
        />
      </section>

      <section className="panel">
        <div className="section-heading compact">
          <div>
            <p className="eyebrow">Last Match</p>
            <h2>{lastMatch ? resultLine(state, lastMatch) : 'Preseason'}</h2>
          </div>
          <CircleDot size={20} />
        </div>
        {lastMatch ? (
          <ul className="report-list">
            {lastMatch.report.slice(0, 3).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        ) : (
          <p className="empty-state">No competitive data yet.</p>
        )}
      </section>

      <section className="panel">
        <div className="section-heading compact">
          <div>
            <p className="eyebrow">Newsroom</p>
            <h2>Club Pulse</h2>
          </div>
          <ClipboardList size={20} />
        </div>
        <div className="news-list">
          {state.news.slice(0, 3).map((item) => (
            <article className={`news-item ${item.tone}`} key={item.id}>
              <strong>{item.title}</strong>
              <p>{item.body}</p>
            </article>
          ))}
        </div>
      </section>
    </section>
  )
}

function LineupReadiness({ lineup, captainName }: { lineup: ReturnType<typeof validateLineup>; captainName: string }) {
  return (
    <div className={lineup.isLegal ? 'readiness ready' : 'readiness blocked'}>
      <div>
        {lineup.isLegal ? <CheckCircle2 size={19} /> : <AlertTriangle size={19} />}
        <strong>{lineup.isLegal ? 'Ready' : 'Blocked'}</strong>
        <span>{captainName}</span>
      </div>
      <ul>
        {(lineup.warnings.length ? lineup.warnings.slice(0, 3) : ['Legal XI for the selected formation.']).map((warning) => (
          <li key={warning}>{warning}</li>
        ))}
      </ul>
    </div>
  )
}

function TacticalQuickPlan({
  tactic,
  trainingFocus,
  onTactic,
  onTraining,
}: {
  tactic: Tactic
  trainingFocus: TrainingFocus
  onTactic: (tactic: Partial<Tactic>) => void
  onTraining: (focus: TrainingFocus) => void
}) {
  return (
    <div className="quick-plan">
      <div className="quick-plan-group">
        <span>Shape</span>
        <div className="segmented compact-segments">
          {formations.map((formation) => (
            <button
              className={tactic.formation === formation ? 'segment active' : 'segment'}
              type="button"
              onClick={() => onTactic({ formation })}
              key={formation}
            >
              {formation}
            </button>
          ))}
        </div>
      </div>

      <div className="quick-plan-group">
        <span>Intent</span>
        <div className="segmented compact-segments">
          {mentalities.map((mentality) => (
            <button
              className={tactic.mentality === mentality ? 'segment active' : 'segment'}
              type="button"
              onClick={() => onTactic({ mentality })}
              key={mentality}
            >
              {mentality}
            </button>
          ))}
        </div>
      </div>

      <div className="quick-plan-group">
        <span>Training</span>
        <div className="segmented compact-segments">
          {trainingOptions.map((focus) => (
            <button
              className={trainingFocus === focus ? 'segment active' : 'segment'}
              type="button"
              onClick={() => onTraining(focus)}
              key={focus}
            >
              {focus}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

function PositionCounts({ lineup }: { lineup: ReturnType<typeof validateLineup> }) {
  return (
    <div className="position-counts">
      {positionOrder.map((position) => (
        <span className={lineup.counts[position] === lineup.required[position] ? 'matched' : 'missing'} key={position}>
          <strong>{position}</strong>
          {lineup.counts[position]}/{lineup.required[position]}
        </span>
      ))}
    </div>
  )
}

function PlayerPicker({
  club,
  starterIds,
  captainId,
  onCaptain,
  onToggleStarter,
}: {
  club: ReturnType<typeof getClub>
  starterIds: string[]
  captainId: string
  onCaptain: (playerId: string) => void
  onToggleStarter: (playerId: string) => void
}) {
  const players = [...club.squad].sort((left, right) => {
    const positionDiff = positionOrder.indexOf(left.position) - positionOrder.indexOf(right.position)
    return positionDiff || playerScore(right) - playerScore(left)
  })

  return (
    <div className="player-picker">
      {players.map((player) => {
        const selected = starterIds.includes(player.id)
        const captain = captainId === player.id

        return (
          <div className={`player-pick ${selected ? 'selected' : ''} ${captain ? 'captain' : ''}`} key={player.id}>
            <button className="player-toggle-row" type="button" onClick={() => onToggleStarter(player.id)}>
              <span className="player-role">{selected ? 'XI' : player.position}</span>
              <span className="player-main">
                <strong>{player.name}</strong>
                <small>{player.personality} · {playerAttributeSummary(player)}</small>
              </span>
              <span className="player-score" title="Overall rating">{Math.round(playerOverall(player))}</span>
              <span className="player-meter">
                <b>Fit</b>
                <i style={{ inlineSize: `${player.fitness}%` }} />
              </span>
              <span className="player-meter">
                <b>Mor</b>
                <i style={{ inlineSize: `${player.morale}%` }} />
              </span>
            </button>
            <button
              className={captain ? 'icon-button selected' : 'icon-button'}
              type="button"
              title={`Set ${player.name} as captain`}
              aria-label={`Set ${player.name} as captain`}
              onClick={() => onCaptain(player.id)}
              disabled={!selected}
            >
              <UserCheck size={17} />
            </button>
          </div>
        )
      })}
    </div>
  )
}

function SquadView({
  selectedClub,
  starterIds,
  captainId,
  onCaptain,
  onToggleStarter,
}: {
  selectedClub: ReturnType<typeof getClub>
  starterIds: string[]
  captainId: string
  onCaptain: (playerId: string) => void
  onToggleStarter: (playerId: string) => void
}) {
  const players = [...selectedClub.squad].sort((left, right) => playerScore(right) - playerScore(left))

  return (
    <section className="single-view">
      <section className="panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">{selectedClub.shortName}</p>
            <h2>Squad Room</h2>
          </div>
          <ClubBadge club={selectedClub} size="md" />
        </div>

        <div className="table-wrap">
          <table className="data-table squad-table">
            <thead>
              <tr>
                <th>Player</th>
                <th>Pos</th>
                <th>OVR</th>
                <th>POT</th>
                <th>Key attributes</th>
                <th>Morale</th>
                <th>Fit</th>
                <th>Age</th>
                <th>Trait</th>
                <th>XI</th>
                <th>Captain</th>
              </tr>
            </thead>
            <tbody>
              {players.map((player) => {
                const starter = starterIds.includes(player.id)
                return (
                  <tr className={starter ? 'starter-row' : undefined} key={player.id}>
                    <td>
                      <strong>{player.name}</strong>
                      <span>€{player.value.toFixed(1)}m · €{player.wage.toFixed(1)}k/w</span>
                    </td>
                    <td>{player.position}</td>
                    <td>{Math.round(playerOverall(player))}</td>
                    <td>{Math.round(player.potential)}</td>
                    <td className="attribute-summary">{playerAttributeSummary(player)}</td>
                    <td>
                      <InlineMeter value={player.morale} />
                    </td>
                    <td>
                      <InlineMeter value={player.fitness} />
                    </td>
                    <td>{player.age}</td>
                    <td>{player.personality}</td>
                    <td>
                      <button
                        className={starter ? 'mini-command active' : 'mini-command'}
                        type="button"
                        onClick={() => onToggleStarter(player.id)}
                      >
                        {starter ? 'XI' : 'Bench'}
                      </button>
                    </td>
                    <td>
                      <button
                        className={captainId === player.id ? 'icon-button selected' : 'icon-button'}
                        type="button"
                        title={`Set ${player.name} as captain`}
                        aria-label={`Set ${player.name} as captain`}
                        onClick={() => onCaptain(player.id)}
                        disabled={!starter}
                      >
                        <UserCheck size={17} />
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </section>
    </section>
  )
}

function TacticsView({
  tactic,
  trainingFocus,
  onTactic,
}: {
  tactic: Tactic
  trainingFocus: TrainingFocus
  onTactic: (tactic: Partial<Tactic>) => void
}) {
  const control = Math.round(45 + (100 - Math.abs(tactic.tempo - 52)) * 0.25 + (tactic.mentality === 'Measured' ? 12 : 0))
  const intensity = Math.round((tactic.pressing + tactic.tempo) / 2)
  const risk = Math.round((tactic.defensiveLine + tactic.pressing + (tactic.mentality === 'Relentless' ? 20 : 0)) / 2.4)

  return (
    <section className="view-grid tactics-view">
      <section className="panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Shape</p>
            <h2>{tactic.formation}</h2>
          </div>
          <Settings2 size={22} />
        </div>
        <div className="segmented large">
          {formations.map((formation) => (
            <button
              className={tactic.formation === formation ? 'segment active' : 'segment'}
              type="button"
              onClick={() => onTactic({ formation })}
              key={formation}
            >
              {formation}
            </button>
          ))}
        </div>
      </section>

      <section className="panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Intent</p>
            <h2>{tactic.mentality}</h2>
          </div>
          <Activity size={22} />
        </div>
        <div className="segmented large">
          {mentalities.map((mentality) => (
            <button
              className={tactic.mentality === mentality ? 'segment active' : 'segment'}
              type="button"
              onClick={() => onTactic({ mentality })}
              key={mentality}
            >
              {mentality}
            </button>
          ))}
        </div>
      </section>

      <section className="panel wide-panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Levers</p>
            <h2>Match Model</h2>
          </div>
          <Gauge size={22} />
        </div>
        <div className="slider-grid">
          <RangeControl label="Pressing" value={tactic.pressing} onChange={(pressing) => onTactic({ pressing })} />
          <RangeControl label="Tempo" value={tactic.tempo} onChange={(tempo) => onTactic({ tempo })} />
          <RangeControl label="Line" value={tactic.defensiveLine} onChange={(defensiveLine) => onTactic({ defensiveLine })} />
        </div>
      </section>

      <section className="panel">
        <div className="section-heading compact">
          <div>
            <p className="eyebrow">Identity</p>
            <h2>{trainingFocus}</h2>
          </div>
          <Dumbbell size={20} />
        </div>
        <div className="tempo-row stacked">
          <ValueBar label="Control" value={control} />
          <ValueBar label="Intensity" value={intensity} />
          <ValueBar label="Risk" value={risk} />
        </div>
      </section>
    </section>
  )
}

function MatchView({ state, lastMatch }: { state: CareerState; lastMatch?: MatchResult }) {
  return (
    <section className="single-view match-view">
      <section className="panel pitch-panel">
        <PitchCanvas state={state} result={lastMatch} />
      </section>

      <section className="match-grid">
        <section className="panel">
          <div className="section-heading compact">
            <div>
              <p className="eyebrow">Score</p>
              <h2>{lastMatch ? resultLine(state, lastMatch) : 'No Match'}</h2>
            </div>
            <CircleDot size={20} />
          </div>
          {lastMatch ? (
            <div className="metric-comparison">
              <StatDuel label="xG" left={lastMatch.metrics.homeXg} right={lastMatch.metrics.awayXg} />
              <StatDuel label="Shots" left={lastMatch.metrics.homeShots} right={lastMatch.metrics.awayShots} />
              <StatDuel label="On Target" left={lastMatch.metrics.homeShotsOnTarget} right={lastMatch.metrics.awayShotsOnTarget} />
              <StatDuel label="Possession" left={lastMatch.metrics.homePossession} right={100 - lastMatch.metrics.homePossession} suffix="%" />
            </div>
          ) : (
            <p className="empty-state">Play a matchday to create the first report.</p>
          )}
        </section>

        <section className="panel">
          <div className="section-heading compact">
            <div>
              <p className="eyebrow">Events</p>
              <h2>Timeline</h2>
            </div>
            <ClipboardList size={20} />
          </div>
          <ol className="event-list">
            {(lastMatch?.events ?? []).slice(0, 9).map((event) => (
              <li className={event.type} key={`${event.minute}-${event.text}`}>
                <span>{event.minute}'</span>
                <p>{event.text}</p>
              </li>
            ))}
          </ol>
        </section>

        <section className="panel wide-panel">
          <div className="section-heading compact">
            <div>
              <p className="eyebrow">Readout</p>
              <h2>Tactical Report</h2>
            </div>
            <BarChart3 size={20} />
          </div>
          {lastMatch ? (
            <ul className="report-list columns">
              {lastMatch.report.map((line) => (
                <li key={line}>{line}</li>
              ))}
            </ul>
          ) : (
            <p className="empty-state">The analyst desk is waiting.</p>
          )}
        </section>
      </section>
    </section>
  )
}

function LeagueView({ state, standings }: { state: CareerState; standings: ReturnType<typeof getSortedStandings> }) {
  const currentRound = state.fixtures[Math.min(state.roundIndex, state.fixtures.length - 1)] ?? []
  const recentResults = state.results.filter((result) => result.round === state.roundIndex - 1)

  return (
    <section className="view-grid league-view">
      <section className="panel wide-panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Table</p>
            <h2>Premier League</h2>
          </div>
          <Trophy size={22} />
        </div>
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Club</th>
                <th>P</th>
                <th>W</th>
                <th>D</th>
                <th>L</th>
                <th>GD</th>
                <th>Pts</th>
                <th>Form</th>
              </tr>
            </thead>
            <tbody>
              {standings.map((standing, index) => {
                const club = getClub(state, standing.clubId)
                return (
                  <tr className={club.id === state.selectedClubId ? 'managed-row' : undefined} key={club.id}>
                    <td>{index + 1}</td>
                    <td>
                      <div className="table-club">
                        <ClubBadge club={club} size="sm" />
                        <div>
                          <strong>{club.name}</strong>
                          <span>{club.managerStyle}</span>
                        </div>
                      </div>
                    </td>
                    <td>{standing.played}</td>
                    <td>{standing.won}</td>
                    <td>{standing.drawn}</td>
                    <td>{standing.lost}</td>
                    <td>{standing.goalsFor - standing.goalsAgainst}</td>
                    <td>{standing.points}</td>
                    <td>
                      <div className="form-dots">
                        {standing.form.map((item, formIndex) => (
                          <span className={`form-dot ${item.toLowerCase()}`} key={`${club.id}-${formIndex}`}>
                            {item}
                          </span>
                        ))}
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel">
        <div className="section-heading compact">
          <div>
            <p className="eyebrow">Fixtures</p>
            <h2>Matchday {Math.min(state.roundIndex + 1, state.fixtures.length)}</h2>
          </div>
          <CalendarDays size={20} />
        </div>
        <div className="fixture-list">
          {currentRound.map((fixture) => {
            const home = getClub(state, fixture.homeId)
            const away = getClub(state, fixture.awayId)
            return (
            <div className="fixture-row" key={fixture.id}>
              <span className="fixture-club"><ClubBadge club={home} size="sm" />{home.shortName}</span>
              <strong>v</strong>
              <span className="fixture-club"><ClubBadge club={away} size="sm" />{away.shortName}</span>
            </div>
            )
          })}
        </div>
      </section>

      <section className="panel">
        <div className="section-heading compact">
          <div>
            <p className="eyebrow">Results</p>
            <h2>Latest</h2>
          </div>
          <ClipboardList size={20} />
        </div>
        <div className="fixture-list">
          {recentResults.length ? (
            recentResults.map((result) => {
              const home = getClub(state, result.homeId)
              const away = getClub(state, result.awayId)
              return (
              <div className="fixture-row result-row" key={result.fixtureId}>
                <span className="fixture-club"><ClubBadge club={home} size="sm" />{home.shortName}</span>
                <strong>
                  {result.homeGoals}-{result.awayGoals}
                </strong>
                <span className="fixture-club"><ClubBadge club={away} size="sm" />{away.shortName}</span>
              </div>
              )
            })
          ) : (
            <p className="empty-state">No results yet.</p>
          )}
        </div>
      </section>
    </section>
  )
}

function AnalyticsView({
  clubId,
  selectedClub,
  clubs,
  runId,
  status,
  seasonSyncProgress,
  localResults,
  onStatusChange,
  onOpenMatch,
}: {
  clubId: string
  selectedClub: Club
  clubs: Club[]
  runId: string
  status: AnalyticsSyncStatus
  seasonSyncProgress?: SeasonSyncProgress
  localResults: MatchResult[]
  onStatusChange: (status: AnalyticsSyncStatus) => void
  onOpenMatch: (matchId: string) => void
}) {
  const [summary, setSummary] = useState<AnalyticsSummary>()
  const [actionInsights, setActionInsights] = useState<ActionInsights>()
  const [timeline, setTimeline] = useState<AnalyticsTimeline>()
  const [runInfo, setRunInfo] = useState<AnalyticsRun>()
  const [seasonComparison, setSeasonComparison] = useState<AnalyticsSeasonComparison[]>([])
  const [tacticalMatchups, setTacticalMatchups] = useState<TacticalMatchup[]>([])
  const [icebergHistory, setIcebergHistory] = useState<IcebergHistory>()
  const [loadedKey, setLoadedKey] = useState('')
  const [error, setError] = useState(false)
  const requestKey = `${clubId}:${runId}:${localResults.length}`
  const loading = loadedKey !== requestKey && !summary
  const points = timeline?.points ?? []
  const [selectedRoundIndex, setSelectedRoundIndex] = useState(0)
  const currentIndex = points.length ? Math.min(selectedRoundIndex, points.length - 1) : 0
  const selectedPoint = points[currentIndex]
  const clubsById = useMemo(() => new Map(clubs.map((club) => [club.id, club])), [clubs])
  const selectedTable = useMemo(
    () => (selectedPoint ? (timeline?.table ?? []).filter((standing) => standing.round === selectedPoint.round).sort((left, right) => left.rank - right.rank) : []),
    [selectedPoint, timeline?.table],
  )
  const selectedMatches = useMemo(
    () => (selectedPoint ? localResults.filter((result) => result.round === selectedPoint.round && (result.homeId === clubId || result.awayId === clubId)) : []),
    [clubId, localResults, selectedPoint],
  )
  const playerNames = useMemo(() => new Map(selectedClub.squad.map((player) => [player.id, player.name])), [selectedClub.squad])
  const comparisonCareerId = runInfo?.careerId ?? runId.split(':season:')[0]

  useEffect(() => {
    let active = true
    if (status === 'syncing') return () => {
      active = false
    }
    void Promise.all([fetchAnalyticsSummary(clubId, runId), fetchAnalyticsTimeline(clubId, runId), fetchActionInsights(runId, clubId)])
      .then(([nextSummary, nextTimeline, nextActionInsights]) => {
        if (!active) return
        setSummary(nextSummary)
        setActionInsights(nextActionInsights)
        setTimeline(nextTimeline)
        setError(false)
        setLoadedKey(requestKey)
        setSelectedRoundIndex(Math.max(0, nextTimeline.points.length - 1))
        onStatusChange('online')
      })
      .catch(() => {
        if (!active) return
        setError(true)
        setLoadedKey(requestKey)
        onStatusChange('offline')
      })

    return () => {
      active = false
    }
  }, [clubId, onStatusChange, requestKey, runId, status])

  useEffect(() => {
    let active = true
    void fetchAnalyticsRuns()
      .then((runs) => {
        if (!active) return
        setRunInfo(runs.find((run) => run.runId === runId))
      })
      .catch(() => {
        if (active) setRunInfo(undefined)
      })

    return () => {
      active = false
    }
  }, [localResults.length, runId, status])

  useEffect(() => {
    let active = true
    void fetchSeasonComparison(comparisonCareerId, clubId)
      .then((comparison) => {
        if (active) setSeasonComparison(comparison)
      })
      .catch(() => {
        if (active) setSeasonComparison([])
      })

    return () => {
      active = false
    }
  }, [clubId, comparisonCareerId, localResults.length, status])

  useEffect(() => {
    let active = true
    void fetchTacticalMatchups(runId, clubId)
      .then((matchups) => {
        if (active) setTacticalMatchups(matchups)
      })
      .catch(() => {
        if (active) setTacticalMatchups([])
      })

    return () => {
      active = false
    }
  }, [clubId, localResults.length, runId, status])

  useEffect(() => {
    let active = true
    void fetchIcebergHistory()
      .then((history) => {
        if (active) setIcebergHistory(history)
      })
      .catch(() => {
        if (active) setIcebergHistory(undefined)
      })

    return () => {
      active = false
    }
  }, [localResults.length, status])

  const statusLabel =
    status === 'syncing'
      ? seasonSyncProgress
        ? `Syncing season ${seasonSyncProgress.completed}/${seasonSyncProgress.total}`
        : 'Syncing matchday'
      : status === 'online'
        ? 'ClickHouse connected'
        : status === 'offline'
          ? 'Analytics offline'
          : 'Waiting for first sync'

  return (
    <section className="single-view analytics-view">
      <section className="panel wide-panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Analytics Lab</p>
            <h2>Touchline Data Room</h2>
          </div>
          <a className="secondary-action" href={clickhouseUiUrl} target="_blank" rel="noreferrer">
            <ExternalLink size={17} />
            <span>Open ClickStack</span>
          </a>
        </div>
        <div className={`analytics-status ${status}`}>
          <span className="status-dot" />
          <strong>{statusLabel}</strong>
          <span>Arrow batches land in ClickHouse for live queries and Iceberg for replay history.</span>
        </div>
        <div className="analytics-run-id">
          <span>Season run</span>
          <code>{runId}</code>
          {runInfo && (
            <span className="analytics-run-meta">
              {runInfo.status === 'complete' ? 'Season complete' : `Round ${runInfo.roundsCompleted} synced`} · schema v{runInfo.schemaVersion}
            </span>
          )}
        </div>
        {icebergHistory && (
          <div className="analytics-run-id">
            <span>Iceberg history</span>
            <span className="analytics-run-meta">
              {icebergHistory.snapshots.length} snapshot{icebergHistory.snapshots.length === 1 ? '' : 's'} · {icebergHistory.table}
            </span>
          </div>
        )}
      </section>

      <RealDataExplorerPanel />
      <WorldCup2026Panel />

      {loading && <section className="panel"><p className="empty-state">Reading the latest match facts…</p></section>}

      {!loading && error && (
        <section className="panel wide-panel">
          <div className="empty-state">
            <strong>Analytics service not reachable.</strong>
            <p>{localResults.length ? 'The game has local match data. Start the stack with docker compose up, then open this tab again.' : 'Play a matchday after starting the local analytics stack to populate this view.'}</p>
          </div>
        </section>
      )}

      {!loading && !error && summary && timeline && (
        <>
          <section className="panel analytics-hero">
            <div className="analytics-hero-club">
              <ClubBadge club={selectedClub} size="lg" />
              <div>
                <p className="eyebrow">Season Command Center</p>
                <h2>{selectedClub.name} performance arc</h2>
                <p className="analytics-hero-copy">A round-by-round view built from ClickHouse match facts, standings snapshots, and player development.</p>
              </div>
            </div>
            {selectedPoint ? (
              <div className="analytics-hero-score">
                <span>Round {selectedPoint.round + 1}</span>
                <strong>#{selectedPoint.rank}</strong>
                <small>{selectedPoint.points} points</small>
              </div>
            ) : (
              <div className="analytics-hero-score muted-score">
                <span>Preseason</span>
                <strong>—</strong>
                <small>Awaiting match facts</small>
              </div>
            )}
          </section>

          <section className="analytics-grid">
            <Metric icon={<CircleDot size={17} />} label="Matches" value={summary.matches} />
            <Metric icon={<ClipboardList size={17} />} label="Events" value={summary.events} />
            <Metric icon={<Activity size={17} />} label="Replay frames" value={summary.frames} />
            <Metric icon={<BarChart3 size={17} />} label="Avg xG" value={summary.averageXg.toFixed(2)} />
          </section>

          {actionInsights && (
            <section className="panel wide-panel action-insights-panel">
              <div className="section-heading compact">
                <div>
                  <p className="eyebrow">Action intelligence</p>
                  <h2>What the team actually did</h2>
                </div>
                <Activity size={20} />
              </div>
              <div className="action-insights-metrics">
                <Metric icon={<ClipboardList size={16} />} label="Actions" value={actionInsights.actions} />
                <Metric icon={<CircleDot size={16} />} label="Possessions" value={actionInsights.possessions} />
                <Metric icon={<CheckCircle2 size={16} />} label="Pass completion" value={`${actionInsights.passCompletion.toFixed(1)}%`} />
                <Metric icon={<Target size={16} />} label="Action xG" value={actionInsights.xg.toFixed(2)} />
              </div>
              {actionInsights.actions > 0 ? (
                <>
                  <div className="action-insights-columns">
                    <div>
                      <p className="eyebrow">Action mix</p>
                      <ActionMixList rows={actionInsights.actionMix} />
                    </div>
                    <div>
                      <p className="eyebrow">Passing connections</p>
                      <PassNetworkTable links={actionInsights.passNetwork} playerNames={playerNames} />
                    </div>
                  </div>
                  <div className="player-role-section">
                    <p className="eyebrow">Player role map</p>
                    <PlayerRoleTable profiles={actionInsights.playerProfiles} playerNames={playerNames} />
                  </div>
                </>
              ) : (
                <p className="empty-state">No structured actions have landed for this run yet. Play a matchday to populate this lab.</p>
              )}
              <div className="analytics-insight-note">
                <AlertTriangle size={16} />
                <span>{actionInsights.analystNote}</span>
              </div>
            </section>
          )}

          {points.length > 0 && selectedPoint && (
            <>
              <section className="panel wide-panel analytics-story-panel">
                <div className="section-heading compact">
                  <div>
                    <p className="eyebrow">Season story</p>
                    <h2>How the campaign moved</h2>
                  </div>
                  <div className="analytics-round-label">
                    <span>Round</span>
                    <strong>{selectedPoint.round + 1}</strong>
                    <span>of {points.length}</span>
                  </div>
                </div>
                <input
                  className="analytics-round-slider"
                  type="range"
                  min="0"
                  max={Math.max(0, points.length - 1)}
                  value={currentIndex}
                  onChange={(event) => setSelectedRoundIndex(Number(event.target.value))}
                  aria-label="Select season round"
                />
                <div className="analytics-chart-grid">
                  <TimelineChart
                    points={points}
                    selectedIndex={currentIndex}
                    title="Points collected"
                    series={[{ key: 'points', label: 'Points', color: 'var(--club-primary)' }]}
                  />
                  <TimelineChart
                    points={points}
                    selectedIndex={currentIndex}
                    title="Expected goals"
                    series={[
                      { key: 'xgFor', label: 'For', color: 'var(--club-primary)' },
                      { key: 'xgAgainst', label: 'Against', color: 'var(--negative)' },
                    ]}
                  />
                </div>
                <div className="analytics-round-summary">
                  <Metric icon={<Trophy size={16} />} label="Position" value={`#${selectedPoint.rank}`} />
                  <Metric icon={<BarChart3 size={16} />} label="Points" value={selectedPoint.points} />
                  <Metric icon={<Target size={16} />} label="xG" value={`${selectedPoint.xgFor.toFixed(2)} / ${selectedPoint.xgAgainst.toFixed(2)}`} />
                  <Metric icon={<Gauge size={16} />} label="Possession" value={`${selectedPoint.possession.toFixed(1)}%`} />
                </div>
              </section>

              <section className="panel wide-panel">
                <div className="section-heading compact">
                  <div>
                    <p className="eyebrow">League table</p>
                    <h2>Round {selectedPoint.round + 1} snapshot</h2>
                  </div>
                  <Trophy size={20} />
                </div>
                <LeagueSnapshotTable standings={selectedTable} clubsById={clubsById} selectedClubId={clubId} />
              </section>

              <section className="analytics-lower-grid">
                <section className="panel">
                  <div className="section-heading compact">
                    <div>
                      <p className="eyebrow">Matchday {selectedPoint.round + 1}</p>
                      <h2>Selected club fixtures</h2>
                    </div>
                    <CircleDot size={20} />
                  </div>
                  <MatchdayCards matches={selectedMatches} clubsById={clubsById} onOpenMatch={onOpenMatch} />
                </section>
                <section className="panel">
                  <div className="section-heading compact">
                    <div>
                      <p className="eyebrow">Player development</p>
                      <h2>Who moved the needle?</h2>
                    </div>
                    <Users size={20} />
                  </div>
                  <DevelopmentCards players={timeline.players} />
                </section>
              </section>
            </>
          )}

          <section className="panel wide-panel">
            <div className="section-heading compact">
              <div>
                <p className="eyebrow">Selected Club</p>
                <h2>Performance footprint</h2>
              </div>
              <BarChart3 size={20} />
            </div>
            <div className="tempo-row">
              <ValueBar label="Average possession" value={summary.averagePossession} />
              <ValueBar label="Stored matches" value={Math.min(100, summary.matches * 10)} suffix={String(summary.matches)} />
              <ValueBar label="Event density" value={summary.matches ? Math.min(100, summary.events / summary.matches * 4) : 0} suffix={summary.matches ? `${(summary.events / summary.matches).toFixed(1)} / match` : '0 / match'} />
            </div>
          </section>

          <section className="panel wide-panel">
            <div className="section-heading compact">
              <div>
                <p className="eyebrow">Player Form</p>
                <h2>Ratings from the fact table</h2>
              </div>
              <Users size={20} />
            </div>
            <PlayerFormTable players={summary.players} />
          </section>

          {seasonComparison.length > 0 && (
            <section className="panel wide-panel">
              <div className="section-heading compact">
                <div>
                  <p className="eyebrow">Career archive</p>
                  <h2>Season-over-season comparison</h2>
                </div>
                <CalendarDays size={20} />
              </div>
              <SeasonComparisonTable seasons={seasonComparison} />
            </section>
          )}

          {tacticalMatchups.length > 0 && (
            <section className="panel wide-panel">
              <div className="section-heading compact">
                <div>
                  <p className="eyebrow">Tactical matchup lab</p>
                  <h2>How the game plan performed</h2>
                </div>
                <Shield size={20} />
              </div>
              <TacticalMatchupTable matchups={tacticalMatchups} clubsById={clubsById} />
            </section>
          )}
        </>
      )}
    </section>
  )
}

function RealDataExplorerPanel() {
  const [matches, setMatches] = useState<RealMatchSummary[]>([])
  const [selectedMatchId, setSelectedMatchId] = useState('')
  const [explorer, setExplorer] = useState<RealMatchExplorer>()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [refreshKey, setRefreshKey] = useState(0)

  function refreshMatches() {
    setLoading(true)
    setRefreshKey((value) => value + 1)
  }

  useEffect(() => {
    let active = true
    void fetchRealDataMatches('statsbomb', 2015)
      .then((nextMatches) => {
        if (!active) return
        setMatches(nextMatches)
        setSelectedMatchId((current) => nextMatches.some((match) => match.sourceMatchId === current) ? current : nextMatches[0]?.sourceMatchId ?? '')
        setError(false)
      })
      .catch(() => {
        if (!active) return
        setMatches([])
        setSelectedMatchId('')
        setExplorer(undefined)
        setError(true)
      })
      .finally(() => {
        if (active) setLoading(false)
      })

    return () => {
      active = false
    }
  }, [refreshKey])

  useEffect(() => {
    let active = true
    if (!selectedMatchId) return
    void fetchRealDataMatch('statsbomb', selectedMatchId)
      .then((nextExplorer) => {
        if (active) setExplorer(nextExplorer)
      })
      .catch(() => {
        if (active) setExplorer(undefined)
      })

    return () => {
      active = false
    }
  }, [selectedMatchId])

  const selectedMatch = explorer?.match ?? matches.find((match) => match.sourceMatchId === selectedMatchId)

  return (
    <section className="panel wide-panel real-data-panel">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Real event data</p>
          <h2>Real Data Explorer</h2>
          <p className="panel-copy">Explore normalized match actions from StatsBomb Open Data beside the ClickHouse season lab.</p>
        </div>
        <div className="real-data-source">
          <span className="source-badge">StatsBomb Open Data</span>
          <a href="https://github.com/hudl/open-data" target="_blank" rel="noreferrer">Source ↗</a>
        </div>
      </div>

      {loading ? (
        <p className="empty-state">Reading imported real matches…</p>
      ) : error ? (
        <div className="empty-state real-data-empty">
          <strong>Real-data API unavailable.</strong>
          <p>Start the local stack, then refresh this panel.</p>
          <button className="secondary-action" type="button" onClick={refreshMatches}><RotateCcw size={16} />Retry</button>
        </div>
      ) : !matches.length ? (
        <div className="empty-state real-data-empty">
          <strong>No real matches imported yet.</strong>
          <p>Run <code>node scripts/import-statsbomb.mjs --limit=20</code> from the project root, then refresh.</p>
          <button className="secondary-action" type="button" onClick={refreshMatches}><RotateCcw size={16} />Refresh matches</button>
        </div>
      ) : selectedMatch ? (
        <>
          <div className="real-data-controls">
            <label>
              <span>Imported match</span>
              <select className="real-data-select" value={selectedMatchId} onChange={(event) => setSelectedMatchId(event.target.value)}>
                {matches.map((match) => (
                  <option value={match.sourceMatchId} key={match.sourceMatchId}>
                    {match.matchDate} · {match.homeTeamName} {match.homeScore}–{match.awayScore} {match.awayTeamName}
                  </option>
                ))}
              </select>
            </label>
            <button className="secondary-action" type="button" onClick={refreshMatches}><RotateCcw size={16} />Refresh</button>
          </div>
          <div className="real-data-metrics">
            <Metric icon={<ClipboardList size={16} />} label="Actions" value={selectedMatch.actions} />
            <Metric icon={<CircleDot size={16} />} label="Possessions" value={selectedMatch.possessions} />
            <Metric icon={<Users size={16} />} label="Players" value={selectedMatch.players} />
            <Metric icon={<Target size={16} />} label="Shot xG" value={selectedMatch.xg.toFixed(2)} />
          </div>
          {explorer ? (
            <div className="real-data-grid">
              <RealShotMap shots={explorer.shots} homeTeamName={selectedMatch.homeTeamName} />
              <RealPassNetworkTable links={explorer.passNetwork} homeTeamName={selectedMatch.homeTeamName} />
              <RealPlayerProfileTable profiles={explorer.playerProfiles} />
            </div>
          ) : (
            <p className="empty-state">Loading match actions…</p>
          )}
        </>
      ) : null}
    </section>
  )
}

function RealShotMap({ shots, homeTeamName }: { shots: RealShot[]; homeTeamName: string }) {
  return (
    <div className="real-shot-map">
      <div className="real-data-subheading"><span><p className="eyebrow">Shot map</p><strong>Where chances came from</strong></span><span className="real-data-legend"><i className="home" />Home <i className="away" />Away</span></div>
      {shots.length ? (
        <svg viewBox="0 0 100 64" role="img" aria-label="Real match shot map">
          <rect className="real-shot-map-pitch" x="0" y="0" width="100" height="64" rx="2" />
          <line x1="50" x2="50" y1="0" y2="64" className="real-shot-map-line" />
          <circle cx="50" cy="32" r="8" className="real-shot-map-line" />
          <circle cx="50" cy="32" r="0.8" className="real-shot-map-mark" />
          <rect x="0" y="16" width="16" height="32" className="real-shot-map-line" />
          <rect x="84" y="16" width="16" height="32" className="real-shot-map-line" />
          <rect x="0" y="24" width="6" height="16" className="real-shot-map-line" />
          <rect x="94" y="24" width="6" height="16" className="real-shot-map-line" />
          {shots.map((shot, index) => {
            const isHome = shot.teamName === homeTeamName
            const y = shot.startY * 0.64
            const radius = Math.max(1.2, Math.min(3.2, 1.2 + shot.xg * 5))
            return (
              <circle className={isHome ? 'real-shot home' : 'real-shot away'} cx={shot.startX} cy={y} r={radius} key={`${shot.playerName}-${shot.second}-${index}`}>
                <title>{`${shot.playerName} · ${shot.xg.toFixed(2)} xG · ${shot.outcome}`}</title>
              </circle>
            )
          })}
        </svg>
      ) : <p className="empty-state">No shot events in this match.</p>}
    </div>
  )
}

function RealPassNetworkTable({ links, homeTeamName }: { links: RealPassNetworkLink[]; homeTeamName: string }) {
  if (!links.length) return <div className="real-data-table-card"><p className="eyebrow">Passing connections</p><p className="empty-state">No recipient links available.</p></div>

  return (
    <div className="real-data-table-card">
      <div className="real-data-subheading"><span><p className="eyebrow">Passing connections</p><strong>Top player links</strong></span><Activity size={18} /></div>
      <div className="table-wrap">
        <table className="data-table compact-table">
          <thead><tr><th>Team</th><th>Passer → receiver</th><th>Comp.</th></tr></thead>
          <tbody>
            {links.map((link) => (
              <tr key={`${link.teamName}-${link.passer}-${link.receiver}`}>
                <td><span className={link.teamName === homeTeamName ? 'positive-text' : 'negative-text'}>{link.teamName}</span></td>
                <td><strong>{link.passer}</strong><span className="table-subtext"> → {link.receiver}</span></td>
                <td>{link.completions}/{link.attempts} <span className="table-subtext">{link.completionRate.toFixed(0)}%</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function RealPlayerProfileTable({ profiles }: { profiles: RealPlayerProfile[] }) {
  if (!profiles.length) return <div className="real-data-table-card"><p className="eyebrow">Player profiles</p><p className="empty-state">No player actions available.</p></div>

  return (
    <div className="real-data-table-card real-player-table-card">
      <div className="real-data-subheading"><span><p className="eyebrow">Player profiles</p><strong>Match activity ledger</strong></span><UserCheck size={18} /></div>
      <div className="table-wrap">
        <table className="data-table compact-table">
          <thead><tr><th>Player</th><th>Passes</th><th>Carries</th><th>Shots</th><th>Def.</th><th>xG</th></tr></thead>
          <tbody>
            {profiles.map((profile) => (
              <tr key={profile.playerId}>
                <td><strong>{profile.playerName}</strong><span className="table-subtext">{profile.teamName}</span></td>
                <td>{profile.completedPasses}/{profile.passes}</td>
                <td>{profile.carries}</td>
                <td>{profile.shots}</td>
                <td>{profile.defensiveActions}</td>
                <td>{profile.xg.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function WorldCup2026Panel() {
  const [overview, setOverview] = useState<WorldCup2026Overview>()
  const [group, setGroup] = useState('Group A')
  const [round, setRound] = useState('all')
  const [error, setError] = useState(false)

  useEffect(() => {
    let active = true
    void fetchWorldCup2026Overview()
      .then((nextOverview) => {
        if (!active) return
        setOverview(nextOverview)
        setError(false)
      })
      .catch(() => {
        if (active) setError(true)
      })
    return () => {
      active = false
    }
  }, [])

  const groups = useMemo(() => [...new Set((overview?.teams ?? []).map((team) => team.groupName).filter(Boolean))].sort(), [overview?.teams])
  const knockoutRounds = ['Round of 32', 'Round of 16', 'Quarter-final', 'Semi-final', 'Match for third place', 'Final']
  const groupTeams = overview?.teams.filter((team) => team.groupName === group) ?? []
  const bracketMatches = overview?.matches.filter((match) => !match.groupName && (round === 'all' || match.round === round)) ?? []
  const finalMatch = overview?.matches.find((match) => match.round === 'Final')
  const maxScorerGoals = Math.max(...(overview?.topScorers ?? []).map((scorer) => scorer.goals), 1)
  const maxTimingGoals = Math.max(...(overview?.goalTiming ?? []).map((bucket) => bucket.goals), 1)

  return (
    <section className="panel wide-panel worldcup-panel">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Tournament intelligence</p>
          <h2>FIFA World Cup 2026</h2>
          <p className="panel-copy">A ClickHouse-backed view of the 104-match tournament, from group tables to the final goal.</p>
        </div>
        <div className="real-data-source">
          <span className="source-badge">Openfootball · CC0</span>
          <a href="https://www.fifa.com/en/tournaments/mens/worldcup/canadamexicousa2026/statistics" target="_blank" rel="noreferrer">FIFA stats ↗</a>
        </div>
      </div>

      {error ? (
        <p className="empty-state">World Cup data is not imported yet. Run <code>node scripts/import-worldcup2026.mjs</code> after starting the analytics stack.</p>
      ) : !overview ? (
        <p className="empty-state">Crunching World Cup matches from ClickHouse…</p>
      ) : (
        <>
          <div className="worldcup-summary-grid">
            <Metric icon={<CalendarDays size={16} />} label="Matches" value={overview.summary.matches} />
            <Metric icon={<Users size={16} />} label="Teams" value={overview.summary.teams} />
            <Metric icon={<Target size={16} />} label="Goals" value={overview.summary.goals} />
            <Metric icon={<BarChart3 size={16} />} label="Goals / match" value={overview.summary.averageGoals.toFixed(2)} />
            <Metric icon={<CircleDot size={16} />} label="Venues" value={overview.summary.venues} />
          </div>

          <div className="worldcup-champion-card">
            <div>
              <p className="eyebrow">Final</p>
              <h3>{overview.summary.champion} are world champions</h3>
              <p>{overview.summary.runnerUp} · {finalMatch?.homeTeam} {finalMatch?.homeGoals}-{finalMatch?.awayGoals} {finalMatch?.awayTeam} after extra time</p>
            </div>
            <Trophy size={28} />
          </div>

          <div className="worldcup-main-grid">
            <section className="worldcup-card">
              <div className="worldcup-card-heading">
                <div><p className="eyebrow">Group stage</p><strong>Standings explorer</strong></div>
                <select value={group} onChange={(event) => setGroup(event.target.value)} aria-label="Select World Cup group">
                  {groups.map((groupName) => <option value={groupName} key={groupName}>{groupName}</option>)}
                </select>
              </div>
              <div className="table-wrap">
                <table className="data-table compact-table worldcup-table">
                  <thead><tr><th>#</th><th>Team</th><th>Pl</th><th>W-D-L</th><th>GD</th><th>Pts</th></tr></thead>
                  <tbody>
                    {groupTeams.map((team) => (
                      <tr key={team.teamName}>
                        <td><strong>{team.rank}</strong></td>
                        <td><strong>{team.teamName}</strong><span className="table-subtext">{team.stage}</span></td>
                        <td>{team.played}</td>
                        <td>{team.won}-{team.drawn}-{team.lost}</td>
                        <td className={team.goalDifference >= 0 ? 'positive-text' : 'negative-text'}>{team.goalDifference > 0 ? '+' : ''}{team.goalDifference}</td>
                        <td><strong>{team.points}</strong></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>

            <section className="worldcup-card">
              <div className="worldcup-card-heading"><div><p className="eyebrow">Golden boot race</p><strong>Top scorers</strong></div><Target size={18} /></div>
              <div className="worldcup-bars">
                {overview.topScorers.slice(0, 8).map((scorer) => (
                  <div className="worldcup-bar-row" key={`${scorer.teamName}-${scorer.playerName}`}>
                    <div className="worldcup-bar-label"><strong>{scorer.playerName}</strong><span>{scorer.teamName} · {scorer.penaltyGoals} pens</span></div>
                    <div className="worldcup-bar-track"><i style={{ width: `${scorer.goals / maxScorerGoals * 100}%` }} /></div>
                    <strong>{scorer.goals}</strong>
                  </div>
                ))}
              </div>
            </section>
          </div>

          <div className="worldcup-lower-grid">
            <section className="worldcup-card">
              <div className="worldcup-card-heading"><div><p className="eyebrow">When goals arrived</p><strong>Goal timing</strong></div><Activity size={18} /></div>
              <div className="worldcup-timing-chart">
                {overview.goalTiming.map((bucket) => (
                  <div className="worldcup-timing-column" key={bucket.label}>
                    <span>{bucket.goals}</span><i style={{ height: `${bucket.goals / maxTimingGoals * 100}%` }} /><small>{bucket.label}</small>
                  </div>
                ))}
              </div>
            </section>
            <section className="worldcup-card">
              <div className="worldcup-card-heading"><div><p className="eyebrow">Host footprint</p><strong>Highest-scoring venues</strong></div><MapPin size={18} /></div>
              <div className="worldcup-venue-list">
                {overview.venues.slice(0, 6).map((venue) => <div className="worldcup-venue-row" key={venue.venue}><strong>{venue.venue}</strong><span>{venue.matches} matches · {venue.goals} goals · {venue.averageGoals.toFixed(2)} / match</span></div>)}
              </div>
            </section>
          </div>

          <section className="worldcup-card worldcup-bracket-card">
            <div className="worldcup-card-heading">
              <div><p className="eyebrow">Knockout path</p><strong>Bracket explorer</strong></div>
              <select value={round} onChange={(event) => setRound(event.target.value)} aria-label="Select World Cup knockout round">
                <option value="all">All knockout rounds</option>
                {knockoutRounds.map((roundName) => <option value={roundName} key={roundName}>{roundName}</option>)}
              </select>
            </div>
            <WorldCupBracket matches={bracketMatches} />
          </section>
        </>
      )}
    </section>
  )
}

function WorldCupBracket({ matches }: { matches: WorldCup2026MatchSummary[] }) {
  if (!matches.length) return <p className="empty-state">No knockout matches match this filter.</p>
  const roundOrder = ['Round of 32', 'Round of 16', 'Quarter-final', 'Semi-final', 'Match for third place', 'Final']
  return (
    <div className="worldcup-bracket">
      {roundOrder.map((round) => {
        const roundMatches = matches.filter((match) => match.round === round)
        if (!roundMatches.length) return null
        return (
          <div className="worldcup-bracket-column" key={round}>
            <p className="eyebrow">{round}</p>
            {roundMatches.map((match) => (
              <div className={match.round === 'Final' ? 'worldcup-match-card final' : 'worldcup-match-card'} key={match.matchNumber}>
                <span>#{match.matchNumber} · {match.venue}</span>
                <strong>{match.homeTeam} <b>{match.homeGoals}</b></strong>
                <strong>{match.awayTeam} <b>{match.awayGoals}</b></strong>
                {match.penaltyShootout && <small>Pens {match.shootoutHomeGoals}-{match.shootoutAwayGoals}</small>}
              </div>
            ))}
          </div>
        )
      })}
    </div>
  )
}

type TimelineSeries = {
  key: keyof AnalyticsTimelinePoint
  label: string
  color: string
}

function TimelineChart({
  points,
  selectedIndex,
  title,
  series,
  invertPosition = false,
}: {
  points: AnalyticsTimelinePoint[]
  selectedIndex: number
  title: string
  series: TimelineSeries[]
  invertPosition?: boolean
}) {
  const width = 760
  const height = 220
  const padding = { top: 18, right: 18, bottom: 28, left: 38 }
  const plotWidth = width - padding.left - padding.right
  const plotHeight = height - padding.top - padding.bottom
  const values = series.flatMap(({ key }) => points.map((point) => Number(point[key])))
  const min = Math.min(...values, 0)
  const max = Math.max(...values, 1)
  const range = max - min || 1
  const x = (index: number) => padding.left + (points.length === 1 ? plotWidth / 2 : (index / (points.length - 1)) * plotWidth)
  const y = (value: number) => {
    const normalized = (value - min) / range
    return invertPosition ? padding.top + normalized * plotHeight : padding.top + (1 - normalized) * plotHeight
  }

  return (
    <div className="analytics-chart">
      <div className="analytics-chart-heading">
        <strong>{title}</strong>
        <div className="analytics-chart-legend">
          {series.map((item) => (
            <span key={item.label}><i style={{ background: item.color }} />{item.label}</span>
          ))}
        </div>
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label={title}>
        {[0, 0.5, 1].map((tick) => {
          const tickValue = max - range * tick
          return (
            <g key={tick}>
              <line x1={padding.left} x2={width - padding.right} y1={padding.top + plotHeight * tick} y2={padding.top + plotHeight * tick} className="chart-gridline" />
              <text x={padding.left - 8} y={padding.top + plotHeight * tick + 4} textAnchor="end" className="chart-label">{tickValue.toFixed(title === 'Expected goals' ? 1 : 0)}</text>
            </g>
          )
        })}
        {series.map((item) => {
          const path = points.map((point, index) => `${index ? 'L' : 'M'} ${x(index)} ${y(Number(point[item.key]))}`).join(' ')
          return <path d={path} fill="none" stroke={item.color} strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" key={item.label} />
        })}
        {points.map((point, index) => (
          <g key={point.round}>
            <line x1={x(index)} x2={x(index)} y1={padding.top} y2={height - padding.bottom} className={index === selectedIndex ? 'chart-focus-line active' : 'chart-focus-line'} />
            {series.map((item) => (
              <circle cx={x(index)} cy={y(Number(point[item.key]))} r={index === selectedIndex ? 5 : 3} fill={item.color} stroke="var(--surface)" strokeWidth="2" key={`${point.round}-${item.label}`}>
                <title>{`Round ${point.round + 1}: ${item.label} ${Number(point[item.key]).toFixed(2)}`}</title>
              </circle>
            ))}
            {(index === 0 || index === points.length - 1 || index === selectedIndex) && <text x={x(index)} y={height - 8} textAnchor="middle" className="chart-label">R{point.round + 1}</text>}
          </g>
        ))}
      </svg>
    </div>
  )
}

function LeagueSnapshotTable({
  standings,
  clubsById,
  selectedClubId,
}: {
  standings: AnalyticsTimelineStanding[]
  clubsById: Map<string, Club>
  selectedClubId: string
}) {
  if (!standings.length) return <p className="empty-state">No standings snapshot has landed for this round.</p>

  return (
    <div className="table-wrap">
      <table className="data-table analytics-table">
        <thead>
          <tr><th>#</th><th>Club</th><th>Pl</th><th>W-D-L</th><th>GD</th><th>Pts</th><th>Form</th></tr>
        </thead>
        <tbody>
          {standings.map((standing) => {
            const club = clubsById.get(standing.clubId)
            return (
              <tr className={standing.clubId === selectedClubId ? 'selected-row' : ''} key={standing.clubId}>
                <td><strong>{standing.rank}</strong></td>
                <td><span className="fixture-club">{club && <ClubBadge club={club} size="sm" />}<strong>{club?.name ?? standing.clubId}</strong></span></td>
                <td>{standing.played}</td>
                <td>{standing.won}-{standing.drawn}-{standing.lost}</td>
                <td className={standing.goalDifference >= 0 ? 'positive-text' : 'negative-text'}>{standing.goalDifference > 0 ? '+' : ''}{standing.goalDifference}</td>
                <td><strong>{standing.points}</strong></td>
                <td><span className="form-inline">{standing.form || '—'}</span></td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function MatchdayCards({
  matches,
  clubsById,
  onOpenMatch,
}: {
  matches: MatchResult[]
  clubsById: Map<string, Club>
  onOpenMatch: (matchId: string) => void
}) {
  if (!matches.length) return <p className="empty-state">No local match report is available for this round.</p>

  return (
    <div className="analytics-match-list">
      {matches.map((match) => {
        const home = clubsById.get(match.homeId)
        const away = clubsById.get(match.awayId)
        return (
          <button className="analytics-match-card" type="button" onClick={() => onOpenMatch(match.fixtureId)} key={match.fixtureId}>
            <span className="analytics-match-meta">Round {match.round + 1} <b>{match.metrics.homeXg.toFixed(2)} xG · {match.events.length} events</b></span>
            <span className="analytics-match-teams"><span>{home?.shortName ?? match.homeId}</span><strong>{match.homeGoals} - {match.awayGoals}</strong><span>{away?.shortName ?? match.awayId}</span></span>
            <span className="analytics-match-action">Open replay →</span>
          </button>
        )
      })}
    </div>
  )
}

function DevelopmentCards({ players }: { players: AnalyticsDevelopmentPlayer[] }) {
  if (!players.length) return <p className="empty-state">Player snapshots will appear after the first matchday.</p>

  return (
    <div className="development-card-list">
      {players.slice(0, 6).map((player) => (
        <div className="development-card" key={player.playerId}>
          <div>
            <strong>{player.playerName}</strong>
            <span>{player.position} · {player.form} form · {player.fitness} fitness</span>
          </div>
          <div className="development-rating">
            <strong>{player.overall}</strong>
            <span className={player.change >= 0 ? 'positive-text' : 'negative-text'}>{player.change > 0 ? '+' : ''}{player.change}</span>
          </div>
        </div>
      ))}
    </div>
  )
}

function PlayerFormTable({ players }: { players: AnalyticsPlayer[] }) {
  if (!players.length) return <p className="empty-state">No player ratings have landed yet.</p>

  return (
    <div className="table-wrap">
      <table className="data-table analytics-table">
        <thead>
          <tr>
            <th>Player</th>
            <th>Matches</th>
            <th>Average rating</th>
          </tr>
        </thead>
        <tbody>
          {players.map((player) => (
            <tr key={player.playerId}>
              <td><strong>{player.playerName}</strong></td>
              <td>{player.matches}</td>
              <td><InlineMeter value={Math.round(player.averageRating * 10)} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ActionMixList({ rows }: { rows: ActionInsights['actionMix'] }) {
  const maximum = Math.max(...rows.map((row) => row.actions), 1)

  return (
    <div className="action-mix-list">
      {rows.map((row) => (
        <div className="action-mix-row" key={row.actionType}>
          <div className="action-mix-label">
            <strong>{row.actionType.replaceAll('-', ' ')}</strong>
            <span>{row.successRate.toFixed(0)}% successful</span>
          </div>
          <i><b style={{ inlineSize: `${(row.actions / maximum) * 100}%` }} /></i>
          <strong>{row.actions}</strong>
        </div>
      ))}
    </div>
  )
}

function PassNetworkTable({ links, playerNames }: { links: ActionInsights['passNetwork']; playerNames: Map<string, string> }) {
  if (!links.length) return <p className="empty-state">No repeat passing connections yet.</p>

  return (
    <div className="table-wrap">
      <table className="data-table analytics-table">
        <thead>
          <tr><th>Passer</th><th>Receiver</th><th>Comp.</th><th>Progressive</th></tr>
        </thead>
        <tbody>
          {links.map((link) => (
            <tr key={`${link.passerId}-${link.receiverId}`}>
              <td><strong>{playerNames.get(link.passerId) ?? link.passerId}</strong></td>
              <td>{playerNames.get(link.receiverId) ?? link.receiverId}</td>
              <td>{link.completions}/{link.attempts} <span className="table-subtext">{link.completionRate.toFixed(0)}%</span></td>
              <td>{link.progressivePasses}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function PlayerRoleTable({ profiles, playerNames }: { profiles: ActionInsights['playerProfiles']; playerNames: Map<string, string> }) {
  if (!profiles.length) return <p className="empty-state">Player roles will appear after the action feed has landed.</p>

  return (
    <div className="table-wrap">
      <table className="data-table analytics-table">
        <thead>
          <tr><th>Player</th><th>Primary role</th><th>Actions</th><th>Passing</th><th>Progression</th><th>Evidence</th></tr>
        </thead>
        <tbody>
          {profiles.map((profile) => (
            <tr key={profile.playerId}>
              <td><strong>{playerNames.get(profile.playerId) ?? profile.playerId}</strong></td>
              <td><span className="role-pill">{profile.primaryRole}</span></td>
              <td>{profile.actions}</td>
              <td>{profile.passes ? `${profile.completedPasses}/${profile.passes}` : '—'}{profile.passes > 0 && <span className="table-subtext">{profile.completionRate.toFixed(0)}%</span>}</td>
              <td>{profile.progressiveActions}</td>
              <td>{profile.shots > 0 ? `${profile.shots} shots · ${profile.xg.toFixed(2)} xG` : `${profile.defensiveActions} defensive`}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function SeasonComparisonTable({ seasons }: { seasons: AnalyticsSeasonComparison[] }) {
  return (
    <div className="table-wrap">
      <table className="data-table analytics-table">
        <thead>
          <tr><th>Season</th><th>Finish</th><th>Points</th><th>Record</th><th>xG</th><th>Avg rating</th><th>Press wins</th></tr>
        </thead>
        <tbody>
          {seasons.map((season) => (
            <tr key={season.runId}>
              <td><strong>{season.season}</strong><span className="table-subtext">{season.status === 'complete' ? 'Complete' : `Round ${season.lastRound + 1}`}</span></td>
              <td>{season.rank ? `#${season.rank}` : '—'}</td>
              <td><strong>{season.points}</strong></td>
              <td>{season.won}-{season.drawn}-{season.lost}</td>
              <td>{season.xgFor.toFixed(2)} / {season.xgAgainst.toFixed(2)}</td>
              <td>{season.averageRating ? season.averageRating.toFixed(2) : '—'}</td>
              <td>{season.averagePressWins.toFixed(1)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function TacticalMatchupTable({ matchups, clubsById }: { matchups: TacticalMatchup[]; clubsById: Map<string, Club> }) {
  return (
    <div className="table-wrap">
      <table className="data-table analytics-table">
        <thead>
          <tr><th>Opponent</th><th>Shape</th><th>Plan</th><th>Matches</th><th>xG</th><th>Possession</th><th>Press wins</th><th>Box entries</th><th>Counters</th></tr>
        </thead>
        <tbody>
          {matchups.map((matchup) => {
            const opponent = clubsById.get(matchup.opponentId)
            return (
              <tr key={`${matchup.opponentId}-${matchup.clubFormation}-${matchup.opponentFormation}`}>
                <td><strong>{opponent?.name ?? matchup.opponentId}</strong><span>{matchup.opponentFormation} · {matchup.opponentMentality}</span></td>
                <td><strong>{matchup.clubFormation}</strong><span>{matchup.clubMentality}</span></td>
                <td>{matchup.clubPressing.toFixed(0)} press · {matchup.clubTempo.toFixed(0)} tempo</td>
                <td>{matchup.matches}</td>
                <td>{matchup.xgFor.toFixed(2)} / {matchup.xgAgainst.toFixed(2)}</td>
                <td>{matchup.possession.toFixed(1)}%</td>
                <td>{matchup.pressWins.toFixed(1)} / {matchup.opponentPressWins.toFixed(1)}</td>
                <td>{matchup.boxEntries.toFixed(1)} / {matchup.opponentBoxEntries.toFixed(1)}</td>
                <td>{matchup.counters.toFixed(1)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function IconButton({ label, children, onClick }: { label: string; children: ReactNode; onClick: () => void }) {
  return (
    <button className="icon-button" type="button" title={label} aria-label={label} onClick={onClick}>
      {children}
    </button>
  )
}

function Metric({ icon, label, value }: { icon: ReactNode; label: string; value: ReactNode }) {
  return (
    <div className="metric">
      {icon}
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function RangeControl({ label, value, onChange }: { label: string; value: number; onChange: (value: number) => void }) {
  return (
    <label className="range-control">
      <span>{label}</span>
      <strong>{value}</strong>
      <input type="range" min="25" max="90" value={value} onChange={(event) => onChange(Number(event.target.value))} />
    </label>
  )
}

function ValueBar({ label, value, suffix }: { label: string; value: number; suffix?: string }) {
  return (
    <div className="value-bar">
      <div>
        <span>{label}</span>
        <strong>{suffix ?? `${Math.round(value)}%`}</strong>
      </div>
      <i style={{ inlineSize: `${Math.min(100, Math.max(0, value))}%` }} />
    </div>
  )
}

function InlineMeter({ value }: { value: number }) {
  return (
    <span className="inline-meter">
      <i style={{ inlineSize: `${value}%` }} />
      <b>{value}</b>
    </span>
  )
}

function StatDuel({ label, left, right, suffix = '' }: { label: string; left: number; right: number; suffix?: string }) {
  const total = Number(left) + Number(right) || 1
  const leftShare = (Number(left) / total) * 100
  return (
    <div className="stat-duel">
      <div>
        <strong>
          {left}
          {suffix}
        </strong>
        <span>{label}</span>
        <strong>
          {right}
          {suffix}
        </strong>
      </div>
      <i>
        <b style={{ inlineSize: `${leftShare}%` }} />
      </i>
    </div>
  )
}

function fixtureLabel(state: CareerState, homeId: string, awayId: string) {
  return `${getClub(state, homeId).shortName} v ${getClub(state, awayId).shortName}`
}

function resultLine(state: CareerState, result: MatchResult) {
  return `${getClub(state, result.homeId).shortName} ${result.homeGoals}-${result.awayGoals} ${getClub(state, result.awayId).shortName}`
}

export default App
