-- Replace the run and club placeholders before running this action-feed lab.
-- Highlights remain presentation data; actions are the analytical source of truth.

SELECT
    action_type,
    outcome,
    count() AS actions,
    uniqExact(possession_id) AS possessions,
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
    round(completions / greatest(attempts, 1) * 100, 1) AS completion_rate,
    round(avgIf(end_x - start_x, end_x > start_x), 1) AS average_progression
FROM touchline_match_actions_v2
WHERE run_id = 'replace-with-run-id'
  AND team_id = 'replace-with-club-id'
  AND action_type = 'pass'
  AND recipient_player_id != ''
GROUP BY passer_id, receiver_id
ORDER BY completions DESC, attempts DESC
LIMIT 40;

SELECT
    possession_id,
    any(sequence_id) AS sequence_id,
    min(second) AS start_second,
    max(second) AS end_second,
    count() AS actions,
    uniqExact(player_id) AS players_involved,
    round(max(end_x) - min(start_x), 1) AS distance_progressed,
    countIf(action_type = 'pass' AND outcome = 'successful') AS completed_passes
FROM touchline_match_actions_v2
WHERE run_id = 'replace-with-run-id'
  AND team_id = 'replace-with-club-id'
GROUP BY possession_id
ORDER BY start_second;

SELECT
    match_id,
    second,
    player_id,
    round(start_x, 1) AS shot_x,
    round(start_y, 1) AS shot_y,
    round(toFloat64OrZero(JSONExtractRaw(qualifiers, 'xg')), 3) AS xg,
    JSONExtractString(qualifiers, 'shotContext') AS shot_context,
    JSONExtractString(qualifiers, 'bodyPart') AS body_part,
    JSONExtractBool(qualifiers, 'bigChance') AS big_chance,
    outcome
FROM touchline_match_actions_v2
WHERE run_id = 'replace-with-run-id'
  AND team_id = 'replace-with-club-id'
  AND action_type = 'shot'
ORDER BY second;

SELECT
    round(intDiv(start_x, 10) * 10) AS start_x_bin,
    round(intDiv(start_y, 10) * 10) AS start_y_bin,
    action_type,
    count() AS actions,
    countIf(outcome = 'successful') AS successful_actions
FROM touchline_match_actions_v2
WHERE run_id = 'replace-with-run-id'
  AND team_id = 'replace-with-club-id'
GROUP BY start_x_bin, start_y_bin, action_type
ORDER BY actions DESC;
