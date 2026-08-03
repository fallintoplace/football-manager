-- Replace the placeholders before running this tactical matchup lab.
-- The same fact join powers /api/analytics/tactical-matchup.

SELECT
    cf.run_id,
    cf.club_id,
    cf.opponent_id,
    cf.formation AS club_formation,
    cf.mentality AS club_mentality,
    round(avg(cf.pressing), 1) AS club_pressing,
    round(avg(cf.tempo), 1) AS club_tempo,
    round(avg(cf.defensive_line), 1) AS club_defensive_line,
    opp.formation AS opponent_formation,
    opp.mentality AS opponent_mentality,
    round(avg(opp.pressing), 1) AS opponent_pressing,
    round(avg(opp.tempo), 1) AS opponent_tempo,
    round(avg(opp.defensive_line), 1) AS opponent_defensive_line,
    count() AS matches,
    sum(cf.goals_for) AS goals_for,
    sum(cf.goals_against) AS goals_against,
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
GROUP BY
    cf.run_id,
    cf.club_id,
    cf.opponent_id,
    cf.formation,
    cf.mentality,
    opp.formation,
    opp.mentality
ORDER BY matches DESC, xg_for DESC;
