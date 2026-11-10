package retention

import "context"

func (j *Job) pruneFleet(ctx context.Context) error {
	findings := j.db.WithContext(ctx).Exec(
		`DELETE FROM fleet_findings WHERE resolved_at IS NOT NULL AND resolved_at < now() - make_interval(days => ?)`,
		FleetRetentionDays)
	j.log("pruned resolved fleet findings", findings)

	runs := j.db.WithContext(ctx).Exec(
		`DELETE FROM fleet_runs WHERE created_at < now() - make_interval(days => ?)`,
		FleetRetentionDays)
	j.log("pruned fleet runs", runs)

	return firstError(findings.Error, runs.Error)
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
