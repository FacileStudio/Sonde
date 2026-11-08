package fleet

import (
	"cmp"
	"slices"
	"time"
)

// Pin is one project's dependency on a tracked package.
type Pin struct {
	ProjectID   int64
	ProjectName string
	Package     string
	Resolved    string
}

// Policy is a deliberate hold. ProjectID nil holds the package everywhere.
type Policy struct {
	ID          int64      `json:"id"`
	Package     string     `json:"package"`
	ProjectID   *int64     `json:"project_id"`
	HeldVersion string     `json:"held_version"`
	Reason      string     `json:"reason"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

// DriftProject is a project that is not on the latest version of a package.
type DriftProject struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Held      bool       `json:"held"`
	Reason    string     `json:"reason,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// DriftVersion groups the projects that sit on one version.
type DriftVersion struct {
	Resolved string         `json:"resolved"`
	Projects []DriftProject `json:"projects"`
}

// DriftRow is the state of one tracked package across the fleet.
type DriftRow struct {
	Package string         `json:"package"`
	Latest  string         `json:"latest"`
	Current int            `json:"current"`
	Behind  []DriftVersion `json:"behind"`
}

const unpinned = "unpinned"

func holdFor(policies []Policy, pin Pin, now time.Time) *Policy {
	for i := range policies {
		policy := policies[i]
		if policy.Package != pin.Package || !policy.ExpiresAt.After(now) || policy.HeldVersion != pin.Resolved {
			continue
		}
		if policy.ProjectID == nil || *policy.ProjectID == pin.ProjectID {
			return &policy
		}
	}
	return nil
}

// BuildDrift compares every pin with the latest version of its package. A pin
// that matches an unexpired policy is shown as held rather than hidden, so a
// hold stays visible until it lapses and the project reappears as behind.
func BuildDrift(pins []Pin, latest map[string]string, policies []Policy, now time.Time) []DriftRow {
	rows := map[string]*DriftRow{}
	versions := map[string]map[string][]DriftProject{}
	for name, version := range latest {
		rows[name] = &DriftRow{Package: name, Latest: version}
		versions[name] = map[string][]DriftProject{}
	}
	for _, pin := range pins {
		if row, tracked := rows[pin.Package]; tracked {
			place(row, versions[pin.Package], pin, policies, now)
		}
	}
	out := make([]DriftRow, 0, len(rows))
	for name, row := range rows {
		row.Behind = groupBehind(versions[name])
		out = append(out, *row)
	}
	slices.SortFunc(out, func(a, b DriftRow) int { return cmp.Compare(a.Package, b.Package) })
	return out
}

func place(row *DriftRow, versions map[string][]DriftProject, pin Pin, policies []Policy, now time.Time) {
	if pin.Resolved == row.Latest {
		row.Current++
		return
	}
	resolved := pin.Resolved
	if resolved == "" {
		resolved = unpinned
	}
	entry := DriftProject{ID: pin.ProjectID, Name: pin.ProjectName}
	if hold := holdFor(policies, pin, now); hold != nil {
		expires := hold.ExpiresAt
		entry.Held, entry.Reason, entry.ExpiresAt = true, hold.Reason, &expires
	}
	versions[resolved] = append(versions[resolved], entry)
}

func groupBehind(versions map[string][]DriftProject) []DriftVersion {
	var behind []DriftVersion
	for resolved, projects := range versions {
		slices.SortFunc(projects, func(a, b DriftProject) int { return cmp.Compare(a.Name, b.Name) })
		behind = append(behind, DriftVersion{Resolved: resolved, Projects: projects})
	}
	slices.SortFunc(behind, func(a, b DriftVersion) int {
		if c := cmp.Compare(len(b.Projects), len(a.Projects)); c != 0 {
			return c
		}
		return cmp.Compare(a.Resolved, b.Resolved)
	})
	return behind
}
