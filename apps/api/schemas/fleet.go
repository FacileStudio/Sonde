package schemas

import "time"

// Project is one GitHub repository of the organisation, as the fleet scan
// lists it. Monitors link to it by project_id and maintenance hangs its
// packages, checks and edges off it. Tier, VersionURL and JournalApp are set
// by people and survive every scan.
type Project struct {
	ID            int64  `gorm:"column:id;primaryKey"`
	GithubRepoID  int64  `gorm:"column:github_repo_id;uniqueIndex"`
	Name          string `gorm:"column:name"`
	DefaultBranch string `gorm:"column:default_branch"`
	Tier          string `gorm:"column:tier"`
	VersionURL    string `gorm:"column:version_url"`
	JournalApp    string `gorm:"column:journal_app"`
	FleetScan
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// FleetScan is what the last scan learned about a project, embedded in Project.
type FleetScan struct {
	Covered         bool       `gorm:"column:covered"`
	NotCovered      string     `gorm:"column:not_covered"`
	LastScannedAt   *time.Time `gorm:"column:last_scanned_at"`
	CIConclusion    string     `gorm:"column:ci_conclusion"`
	FiletWired      bool       `gorm:"column:filet_wired"`
	FiletPin        string     `gorm:"column:filet_pin"`
	DeployedVersion string     `gorm:"column:deployed_version"`
	GoDirective     string     `gorm:"column:go_directive"`
}

func (Project) TableName() string { return "projects" }

// FleetRun is one report the scheduled runner delivered.
type FleetRun struct {
	ID            int64     `gorm:"column:id;primaryKey"`
	StartedAt     time.Time `gorm:"column:started_at"`
	FinishedAt    time.Time `gorm:"column:finished_at"`
	DouaneVersion string    `gorm:"column:douane_version"`
	ReposExpected int       `gorm:"column:repos_expected"`
	ReposScanned  int       `gorm:"column:repos_scanned"`
	Complete      bool      `gorm:"column:complete"`
	FindingsOpen  int       `gorm:"column:findings_open"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (FleetRun) TableName() string { return "fleet_runs" }

// FleetFinding is the current state of one flaw in one project. A finding the
// scan stops reporting keeps its row with ResolvedAt set, until retention
// prunes it.
type FleetFinding struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	ProjectID int64  `gorm:"column:project_id"`
	Key       string `gorm:"column:key"`
	FleetAdvisory
	FleetPackage
	Scope      string     `gorm:"column:scope"`
	Target     string     `gorm:"column:target"`
	FirstSeen  time.Time  `gorm:"column:first_seen"`
	LastSeen   time.Time  `gorm:"column:last_seen"`
	ResolvedAt *time.Time `gorm:"column:resolved_at"`
}

// FleetAdvisory is what is known about the flaw itself, embedded in FleetFinding.
type FleetAdvisory struct {
	Advisory string  `gorm:"column:advisory"`
	Aliases  string  `gorm:"column:aliases"`
	Summary  string  `gorm:"column:summary"`
	Severity string  `gorm:"column:severity"`
	KEV      bool    `gorm:"column:kev"`
	EPSS     float64 `gorm:"column:epss"`
}

// FleetPackage names the affected package and its fix, embedded in FleetFinding.
type FleetPackage struct {
	Package   string `gorm:"column:package"`
	Ecosystem string `gorm:"column:ecosystem"`
	Installed string `gorm:"column:installed"`
	FixedIn   string `gorm:"column:fixed_in"`
}

func (FleetFinding) TableName() string { return "fleet_findings" }

// FleetPin is one dependency a project declares on a package the suite tracks.
type FleetPin struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	ProjectID int64  `gorm:"column:project_id"`
	Path      string `gorm:"column:path"`
	Package   string `gorm:"column:package"`
	Spec      string `gorm:"column:spec"`
	Resolved  string `gorm:"column:resolved"`
}

func (FleetPin) TableName() string { return "fleet_pins" }

// FleetLatest is the newest published version of a tracked package.
type FleetLatest struct {
	Package   string    `gorm:"column:package;primaryKey"`
	Version   string    `gorm:"column:version"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (FleetLatest) TableName() string { return "fleet_latest" }

// FleetPolicy holds a package on a version on purpose. It always expires, so a
// hold cannot quietly become the permanent state.
type FleetPolicy struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	Package     string    `gorm:"column:package"`
	ProjectID   *int64    `gorm:"column:project_id"`
	HeldVersion string    `gorm:"column:held_version"`
	Reason      string    `gorm:"column:reason"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
	CreatedBy   string    `gorm:"column:created_by"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (FleetPolicy) TableName() string { return "fleet_policies" }

// FleetEdge is one dependency the last covering scan read from a project's
// files: a library pin, a call, an event, identity or log shipping.
type FleetEdge struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	ProjectID int64  `gorm:"column:project_id"`
	ToName    string `gorm:"column:to_name"`
	Kind      string `gorm:"column:kind"`
	Auth      string `gorm:"column:auth"`
	Evidence  string `gorm:"column:evidence"`
}

func (FleetEdge) TableName() string { return "fleet_edges" }

// FleetCheck is one conformance check the last covering scan asked of a project.
type FleetCheck struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	ProjectID int64  `gorm:"column:project_id"`
	CheckID   string `gorm:"column:check_id"`
	OK        bool   `gorm:"column:ok"`
	Detail    string `gorm:"column:detail"`
}

func (FleetCheck) TableName() string { return "fleet_checks" }
