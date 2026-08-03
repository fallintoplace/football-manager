WITH club_round AS
(
    SELECT
        run_id,
        season,
        round,
        club_id,
        sum(goals_for) AS goals_for,
        sum(xg_for) AS xg_for
    FROM touchline_club_round_metrics
    WHERE run_id = 'replace-with-run-id'
      AND club_id = 'replace-with-club-id'
    GROUP BY run_id, season, round, club_id
)
SELECT
    run_id,
    season,
    round,
    club_id,
    goals_for,
    round(xg_for, 2) AS xg_for,
    round(xg_for - goals_for, 2) AS xg_underperformance
FROM club_round
WHERE goals_for < xg_for - 0.75
ORDER BY round DESC;
