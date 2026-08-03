-- Replace the match ID in the fixture-level queries with one shown in the Real Data Explorer.

SELECT
    source,
    competition,
    season,
    season_label,
    source_match_id,
    match_date,
    home_team_name,
    home_score,
    away_team_name,
    away_score
FROM touchline_real_matches_v1
WHERE source = 'statsbomb'
ORDER BY match_date, source_match_id;

SELECT
    team_name,
    action_type,
    count() AS actions,
    countIf(outcome = 'successful') AS successful_actions,
    round(toFloat64(successful_actions) / greatest(toFloat64(actions), 1) * 100, 1) AS success_rate,
    uniqExact(possession_id) AS possessions
FROM touchline_real_actions_v1
WHERE source = 'statsbomb'
  AND source_match_id = '3754217'
GROUP BY team_name, action_type
ORDER BY team_name, actions DESC;

SELECT
    team_name,
    player_id,
    any(player_name) AS player_name,
    count() AS actions,
    countIf(action_type = 'pass') AS passes,
    countIf(action_type = 'pass' AND outcome = 'successful') AS completed_passes,
    round(toFloat64(completed_passes) / greatest(toFloat64(passes), 1) * 100, 1) AS pass_completion,
    countIf(action_type = 'carry') AS carries,
    countIf(action_type = 'shot') AS shots,
    round(sum(toFloat64OrZero(JSONExtractRaw(qualifiers, 'shotXg'))), 3) AS xg,
    countIf(action_type IN ('tackle', 'interception', 'duel', 'block', 'clearance', 'recovery')) AS defensive_actions
FROM touchline_real_actions_v1
WHERE source = 'statsbomb'
  AND source_match_id = '3754217'
GROUP BY team_name, player_id
ORDER BY actions DESC
LIMIT 50;

SELECT
    team_name,
    player_name,
    action_type,
    second,
    round(start_x, 1) AS start_x,
    round(start_y, 1) AS start_y,
    round(end_x, 1) AS end_x,
    round(end_y, 1) AS end_y,
    round(toFloat64OrZero(JSONExtractRaw(qualifiers, 'shotXg')), 3) AS shot_xg,
    outcome
FROM touchline_real_actions_v1
WHERE source = 'statsbomb'
  AND source_match_id = '3754217'
  AND action_type = 'shot'
ORDER BY second;

SELECT
    team_name,
    player_name AS passer,
    recipient_player_name AS receiver,
    count() AS attempts,
    countIf(outcome = 'successful') AS completions,
    round(toFloat64(completions) / greatest(toFloat64(attempts), 1) * 100, 1) AS completion_rate
FROM touchline_real_actions_v1
WHERE source = 'statsbomb'
  AND source_match_id = '3754217'
  AND action_type = 'pass'
  AND recipient_player_id != ''
GROUP BY team_name, passer, receiver
ORDER BY completions DESC, attempts DESC
LIMIT 40;
