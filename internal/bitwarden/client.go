package bitwarden

// Secret represents a secret from Bitwarden.
type Secret struct {
	ID        string
	Key       string
	Value     string
	Note      string
	ProjectID string
}

// Project represents a Bitwarden project.
type Project struct {
	ID   string
	Name string
}

// Client is the interface for Bitwarden operations.
type Client interface {
	ListProjects(orgID string) ([]Project, error)
	CreateProject(orgID, name string) (*Project, error)
	ListSecrets(orgID string) ([]Secret, error)
	GetSecret(id string) (*Secret, error)
	CreateSecret(key, value, note, orgID string, projectIDs []string) (*Secret, error)
	DeleteSecrets(ids []string) error
	Close()
}
