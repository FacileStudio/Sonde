package fleet

import (
	stderrors "errors"
	"log/slog"

	"github.com/FacileStudio/tronc/errors"
)

// logDelivery records what an accepted report changed. A run that listed far
// fewer projects than are covered is called out, since nothing was removed
// for the ones it no longer sees, and the projects a complete run removed are
// named.
func (h *Handler) logDelivery(result IngestResult) {
	h.logger.Info("fleet report stored",
		slog.Int64("run_id", result.RunID),
		slog.Int("projects", result.Projects),
		slog.Int("covered", result.Covered),
		slog.Int("new_findings", result.NewFindings),
		slog.Int("resolved", result.Resolved),
		slog.Int("findings_open", result.FindingsOpen),
		slog.Bool("complete", result.Complete))
	if result.CoverageDropped {
		h.logger.Warn("fleet report listed far fewer projects than are covered; none were resolved",
			slog.Int64("run_id", result.RunID), slog.Int("projects", result.Projects))
	}
	if len(result.Removed) > 0 {
		h.logger.Info("fleet report removed projects it no longer lists",
			slog.Int64("run_id", result.RunID), slog.Any("projects", result.Removed))
	}
}

// logRefusal records why a report was not stored. A refusal is logged with its
// code and message; an internal failure also carries the cause, which the
// response never shows. Neither logs the report itself.
func (h *Handler) logRefusal(err error) {
	var refusal *errors.Error
	if !stderrors.As(err, &refusal) {
		h.logger.Error("fleet report not stored", slog.Any("error", err))
		return
	}
	if refusal.Code == "internal" {
		h.logger.Error("fleet report not stored", slog.String("code", refusal.Code), slog.Any("cause", refusal.Cause))
		return
	}
	h.logger.Warn("fleet report refused", slog.String("code", refusal.Code), slog.String("reason", refusal.Message))
}
