package output

// SkillSummary describes a bundled skill and its install status.
type SkillSummary struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Targets     []string            `json:"targets"`
	Status      []SkillTargetStatus `json:"status,omitempty"`
}

type SkillTargetStatus struct {
	Target   string `json:"target"`
	Path     string `json:"path"`
	Present  bool   `json:"present"`
	Matches  bool   `json:"matches,omitempty"`
	ErrorMsg string `json:"error,omitempty"`
}

type SkillsListResponse struct {
	Skills []SkillSummary `json:"skills"`
}

type SkillsInstallRecord struct {
	SkillID string `json:"skill_id"`
	Target  string `json:"target"`
	Path    string `json:"path,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type SkillsInstallResponse struct {
	Records []SkillsInstallRecord `json:"records"`
}

type SkillsUninstallResponse struct {
	Records []SkillsInstallRecord `json:"records"`
}
