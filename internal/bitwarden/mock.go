package bitwarden

import "fmt"

// MockClient is an in-memory Bitwarden client for testing.
type MockClient struct {
	Secrets  map[string]*Secret // keyed by Secret.Key
	Projects map[string]*Project
}

func NewMockClient() *MockClient {
	return &MockClient{
		Secrets:  make(map[string]*Secret),
		Projects: make(map[string]*Project),
	}
}

func (m *MockClient) ListProjects(_ string) ([]Project, error) {
	var out []Project
	for _, p := range m.Projects {
		out = append(out, *p)
	}
	return out, nil
}

func (m *MockClient) CreateProject(_ string, name string) (*Project, error) {
	p := &Project{ID: "mock-proj-" + name, Name: name}
	m.Projects[name] = p
	return p, nil
}

func (m *MockClient) ListSecrets(_ string) ([]Secret, error) {
	var out []Secret
	for _, s := range m.Secrets {
		out = append(out, *s)
	}
	return out, nil
}

func (m *MockClient) GetSecret(id string) (*Secret, error) {
	for _, s := range m.Secrets {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, fmt.Errorf("secret not found: %s", id)
}

func (m *MockClient) GetSecretByKey(key string) (*Secret, error) {
	if s, ok := m.Secrets[key]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("secret not found: %s", key)
}

func (m *MockClient) CreateSecret(key, value, note, orgID string, projectIDs []string) (*Secret, error) {
	pid := ""
	if len(projectIDs) > 0 {
		pid = projectIDs[0]
	}
	s := &Secret{ID: "mock-" + key, Key: key, Value: value, Note: note, ProjectID: pid}
	m.Secrets[key] = s
	return s, nil
}

func (m *MockClient) DeleteSecrets(ids []string) error {
	for _, id := range ids {
		for k, s := range m.Secrets {
			if s.ID == id {
				delete(m.Secrets, k)
			}
		}
	}
	return nil
}

func (m *MockClient) Close() {}
