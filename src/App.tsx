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
  Gauge,
  ListChecks,
  Play,
  RotateCcw,
  Save,
  Settings2,
  Shield,
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
  teamAverage,
  toggleStarter,
  updateCaptain,
  updateTactic,
  updateTraining,
} from './game/career'
import { buildAnalyticsPayload, clickhouseUiUrl, fetchAnalyticsSummary, ingestMatchday } from './game/analytics'
import { createAiTactic, createInitialCareer } from './game/data'
import { playerScore, validateLineup } from './game/lineup'
import type { AnalyticsPlayer, AnalyticsSummary, AnalyticsSyncStatus } from './game/analytics'
import type { CareerState, Club, Formation, MatchResult, Mentality, PlayerPosition, Tactic, TrainingFocus } from './game/types'

type Tab = 'club' | 'squad' | 'tactics' | 'match' | 'league' | 'analytics'

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
      setAnalyticsStatus('syncing')
      void ingestMatchday(buildAnalyticsPayload(advanced.state, advanced.results))
        .then(() => setAnalyticsStatus('online'))
        .catch(() => setAnalyticsStatus('offline'))
    }
    setActiveTab('match')
  }

  function save() {
    setState((current) => saveCareer(current))
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
          onPlay={playMatchday}
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

      {activeTab === 'analytics' && <AnalyticsView clubId={selectedClub.id} status={analyticsStatus} localResults={state.results} />}
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
  onPlay,
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
  onPlay: () => void
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
          <button className="primary-action" type="button" onClick={onPlay} disabled={!canPlay}>
            <Play size={18} />
            <span>{seasonComplete ? 'Start Season' : 'Play Matchday'}</span>
          </button>
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
                <small>{player.personality}</small>
              </span>
              <span className="player-score">{Math.round(playerScore(player))}</span>
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
                <th>Score</th>
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
                    <td>{Math.round(playerScore(player))}</td>
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
  status,
  localResults,
}: {
  clubId: string
  status: AnalyticsSyncStatus
  localResults: MatchResult[]
}) {
  const [summary, setSummary] = useState<AnalyticsSummary>()
  const [loadedKey, setLoadedKey] = useState('')
  const [error, setError] = useState(false)
  const requestKey = `${clubId}:${localResults.length}`
  const loading = loadedKey !== requestKey

  useEffect(() => {
    let active = true
    void fetchAnalyticsSummary(clubId)
      .then((nextSummary) => {
        if (!active) return
        setSummary(nextSummary)
        setError(false)
        setLoadedKey(requestKey)
      })
      .catch(() => {
        if (!active) return
        setError(true)
        setLoadedKey(requestKey)
      })

    return () => {
      active = false
    }
  }, [clubId, requestKey])

  const statusLabel = status === 'syncing' ? 'Syncing matchday' : status === 'online' ? 'ClickHouse connected' : status === 'offline' ? 'Analytics offline' : 'Waiting for first sync'

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
      </section>

      {loading && <section className="panel"><p className="empty-state">Reading the latest match facts…</p></section>}

      {!loading && error && (
        <section className="panel wide-panel">
          <div className="empty-state">
            <strong>Analytics service not reachable.</strong>
            <p>{localResults.length ? 'The game has local match data. Start the stack with docker compose up, then open this tab again.' : 'Play a matchday after starting the local analytics stack to populate this view.'}</p>
          </div>
        </section>
      )}

      {!loading && !error && summary && (
        <>
          <section className="analytics-grid">
            <Metric icon={<CircleDot size={17} />} label="Matches" value={summary.matches} />
            <Metric icon={<ClipboardList size={17} />} label="Events" value={summary.events} />
            <Metric icon={<Activity size={17} />} label="Replay frames" value={summary.frames} />
            <Metric icon={<BarChart3 size={17} />} label="Avg xG" value={summary.averageXg.toFixed(2)} />
          </section>

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
        </>
      )}
    </section>
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
