SELECT
    s.run_id,
    s.season,
    s.round,
    s.club_id,
    s.rank,
    s.played,
    s.points,
    s.goals_for,
    s.goals_against,
    s.goal_difference,
    round(if(m.home_id = s.club_id, m.home_xg, m.away_xg), 2) AS xg_for,
    round(if(m.home_id = s.club_id, m.away_xg, m.home_xg), 2) AS xg_against,
    round(if(m.home_id = s.club_id, m.home_possession, 100 - m.home_possession), 1) AS possession,
    if(m.home_id = s.club_id, m.home_shots, m.away_shots) AS shots_for,
    if(m.home_id = s.club_id, m.away_shots, m.home_shots) AS shots_against,
    if(m.home_id = s.club_id, m.home_pressure, m.away_pressure) AS pressure,
    if(m.home_id = s.club_id, m.home_territory, 100 - m.home_territory) AS territory
FROM (SELECT * FROM touchline_standings FINAL) AS s
LEFT JOIN touchline_matches_v2 AS m
    ON m.run_id = s.run_id
   AND m.season = s.season
   AND m.round = s.round
   AND (m.home_id = s.club_id OR m.away_id = s.club_id)
WHERE s.run_id = 'replace-with-run-id'
  AND s.club_id = 'replace-with-club-id'
ORDER BY s.round;

SELECT
    run_id,
    season,
    round,
    rank,
    club_id,
    points,
    won,
    drawn,
    lost,
    goals_for,
    goals_against,
    goal_difference,
    form
FROM touchline_standings FINAL
WHERE run_id = 'replace-with-run-id'
  AND round = (
      SELECT max(round)
      FROM touchline_standings
      WHERE run_id = 'replace-with-run-id'
  )
ORDER BY rank;

SELECT
    run_id,
    season,
    round,
    club_id,
    rank,
    points,
    goal_difference,
    goals_for
FROM touchline_standings FINAL
WHERE run_id = 'replace-with-run-id'
ORDER BY round, rank;

WITH latest_clubs AS
(
    SELECT
        run_id,
        season,
        club_id,
        argMax(short_name, ingested_at) AS club_name
    FROM touchline_clubs_v2
    WHERE run_id = 'replace-with-run-id'
    GROUP BY run_id, season, club_id
), club_xg AS
(
    SELECT
        m.run_id,
        m.season,
        m.round,
        m.home_id AS club_id,
        m.home_xg AS xg
    FROM touchline_matches_v2 AS m
    WHERE m.run_id = 'replace-with-run-id'
    UNION ALL
    SELECT
        m.run_id,
        m.season,
        m.round,
        m.away_id AS club_id,
        m.away_xg AS xg
    FROM touchline_matches_v2 AS m
    WHERE m.run_id = 'replace-with-run-id'
)
SELECT
    x.run_id,
    x.season,
    x.round,
    x.club_id,
    coalesce(any(c.club_name), x.club_id) AS club_name,
    round(avg(x.xg), 2) AS average_xg
FROM club_xg AS x
LEFT JOIN latest_clubs AS c
    ON c.run_id = x.run_id
   AND c.season = x.season
   AND c.club_id = x.club_id
GROUP BY x.run_id, x.season, x.round, x.club_id
ORDER BY x.round, club_name;

SELECT
    run_id,
    season,
    club_id,
    player_id,
    any(player_name) AS player_name,
    count() AS matches,
    round(avg(rating), 2) AS average_rating,
    round(max(rating), 2) AS best_rating
FROM touchline_player_ratings_v2
WHERE run_id = 'replace-with-run-id'
GROUP BY run_id, season, club_id, player_id
ORDER BY average_rating DESC, matches DESC
LIMIT 50;

SELECT
    run_id,
    season,
    player_id,
    any(player_name) AS player_name,
    any(position) AS position,
    argMin(overall, round) AS opening_overall,
    argMax(overall, round) AS closing_overall,
    closing_overall - opening_overall AS overall_change,
    argMin(form, round) AS opening_form,
    argMax(form, round) AS closing_form,
    argMin(fitness, round) AS opening_fitness,
    argMax(fitness, round) AS closing_fitness,
    argMax(average_rating, round) AS last_match_rating
FROM touchline_player_snapshots FINAL
WHERE run_id = 'replace-with-run-id'
GROUP BY run_id, season, player_id
ORDER BY overall_change DESC, closing_overall DESC
LIMIT 50;

SELECT
    run_id,
    season,
    round,
    event_type,
    team_id,
    count() AS events,
    round(sum(xg), 3) AS event_xg
FROM touchline_match_events_v2
WHERE run_id = 'replace-with-run-id'
GROUP BY run_id, season, round, event_type, team_id
ORDER BY round, events DESC;

SELECT
    run_id,
    season,
    round,
    match_id,
    team_id,
    player_id,
    any(player_name) AS player_name,
    intDiv(toInt32(player_x), 5) * 5 AS x_bin,
    intDiv(toInt32(player_y), 5) * 5 AS y_bin,
    count() AS intensity
FROM touchline_player_frames_v2
WHERE run_id = 'replace-with-run-id'
  AND team_id = 'arsenal'
GROUP BY run_id, season, round, match_id, team_id, player_id, x_bin, y_bin
ORDER BY round, match_id, intensity DESC;

SELECT
    run_id,
    season,
    round,
    club_id,
    sum(matches) AS matches,
    sum(goals_for) AS goals_for,
    sum(goals_against) AS goals_against,
    round(sum(xg_for), 2) AS xg_for,
    round(sum(xg_against), 2) AS xg_against,
    round(avg(possession), 1) AS possession,
    round(avg(pressure), 1) AS pressure,
    round(avg(territory), 1) AS territory
FROM touchline_club_round_metrics
WHERE run_id = 'replace-with-run-id'
  AND club_id = 'replace-with-club-id'
GROUP BY run_id, season, round, club_id
ORDER BY round;

SELECT
    run_id,
    career_id,
    season,
    last_round,
    rounds_completed,
    matches_in_round,
    clubs_expected,
    status,
    schema_version,
    updated_at
FROM touchline_runs_v2
WHERE run_id = 'replace-with-run-id'
ORDER BY updated_at DESC;

SELECT
    run_id,
    season,
    round,
    club_id,
    opponent_id,
    if(is_home, 'home', 'away') AS venue,
    formation,
    mentality,
    pressing,
    tempo,
    defensive_line,
    possessions,
    final_third_entries,
    box_entries,
    press_wins,
    build_up_fails,
    midfield_wins,
    line_breaks,
    balls_behind,
    counters,
    late_fatigue_losses,
    round(xg_for, 2) AS xg_for
FROM touchline_match_club_facts_v2
WHERE run_id = 'replace-with-run-id'
  AND club_id = 'replace-with-club-id'
ORDER BY round, match_id;

SELECT
    run_id,
    season,
    round,
    player_id,
    player_name,
    position,
    opponent_id,
    started,
    minutes_played,
    round(rating, 2) AS rating,
    goals,
    shots,
    round(xg, 3) AS xg,
    round(xg / greatest(minutes_played, 1) * 90, 3) AS xg_per_90
FROM touchline_player_match_facts_v2
WHERE run_id = 'replace-with-run-id'
  AND club_id = 'replace-with-club-id'
ORDER BY round, rating DESC;

SELECT
    cf.opponent_id,
    cf.formation AS club_formation,
    cf.mentality AS club_mentality,
    round(avg(cf.pressing), 1) AS club_pressing,
    opp.formation AS opponent_formation,
    opp.mentality AS opponent_mentality,
    count() AS matches,
    round(avg(cf.xg_for), 2) AS xg_for,
    round(avg(opp.xg_for), 2) AS xg_against,
    round(avg(cf.possession), 1) AS possession,
    round(avg(cf.press_wins), 1) AS press_wins,
    round(avg(opp.press_wins), 1) AS opponent_press_wins,
    round(avg(cf.box_entries), 1) AS box_entries,
    round(avg(opp.box_entries), 1) AS opponent_box_entries,
    round(avg(cf.counters), 1) AS counters,
    round(avg(cf.build_up_fails), 1) AS build_up_fails,
    round(avg(opp.build_up_fails), 1) AS opponent_build_up_fails
FROM touchline_match_club_facts_v2 AS cf
INNER JOIN touchline_match_club_facts_v2 AS opp
    ON opp.run_id = cf.run_id
   AND opp.match_id = cf.match_id
   AND opp.club_id = cf.opponent_id
WHERE cf.run_id = 'replace-with-run-id'
  AND cf.club_id = 'replace-with-club-id'
  AND ('replace-with-opponent-id' = '' OR cf.opponent_id = 'replace-with-opponent-id')
GROUP BY cf.opponent_id, cf.formation, cf.mentality, opp.formation, opp.mentality
ORDER BY matches DESC, xg_for DESC;

SELECT
    action_type,
    outcome,
    count() AS actions,
    uniqExact(possession_id) AS possessions,
    countIf(action_type = 'pass' AND outcome = 'successful') AS completed_passes,
    round(avgIf(end_x - start_x, end_x > start_x), 1) AS average_forward_distance
FROM touchline_match_actions_v2
WHERE run_id = 'replace-with-run-id'
  AND team_id = 'replace-with-club-id'
GROUP BY action_type, outcome
ORDER BY actions DESC;

SELECT
    player_id AS passer_id,
    recipient_player_id AS receiver_id,
    count() AS attempts,
    countIf(outcome = 'successful') AS completions,
    round(completions / greatest(attempts, 1) * 100, 1) AS completion_rate
FROM touchline_match_actions_v2
WHERE run_id = 'replace-with-run-id'
  AND team_id = 'replace-with-club-id'
  AND action_type = 'pass'
  AND recipient_player_id != ''
GROUP BY passer_id, receiver_id
ORDER BY completions DESC, attempts DESC
LIMIT 40;
