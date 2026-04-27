package bitwarden

import (
	"fmt"

	sdk "github.com/bitwarden/sdk-go/v2"
)

type SDKClient struct {
	inner sdk.BitwardenClientInterface
}

func NewSDKClient(accessToken string) (*SDKClient, error) {
	apiURL := "https://api.bitwarden.com"
	identityURL := "https://identity.bitwarden.com"

	client, err := sdk.NewBitwardenClient(&apiURL, &identityURL)
	if err != nil {
		return nil, fmt.Errorf("creating BW client: %w", err)
	}

	if err := client.AccessTokenLogin(accessToken, nil); err != nil {
		client.Close()
		return nil, fmt.Errorf("BW login failed: %w", err)
	}

	return &SDKClient{inner: client}, nil
}

func (c *SDKClient) ListProjects(orgID string) ([]Project, error) {
	resp, err := c.inner.Projects().List(orgID)
	if err != nil {
		return nil, err
	}

	var projects []Project
	for _, p := range resp.Data {
		projects = append(projects, Project{ID: p.ID, Name: p.Name})
	}
	return projects, nil
}

func (c *SDKClient) CreateProject(orgID, name string) (*Project, error) {
	p, err := c.inner.Projects().Create(orgID, name)
	if err != nil {
		return nil, err
	}
	return &Project{ID: p.ID, Name: p.Name}, nil
}

func (c *SDKClient) ListSecrets(orgID string) ([]Secret, error) {
	resp, err := c.inner.Secrets().List(orgID)
	if err != nil {
		return nil, err
	}

	var secrets []Secret
	for _, s := range resp.Data {
		secrets = append(secrets, Secret{ID: s.ID, Key: s.Key})
	}
	return secrets, nil
}

func (c *SDKClient) GetSecret(id string) (*Secret, error) {
	s, err := c.inner.Secrets().Get(id)
	if err != nil {
		return nil, err
	}
	pid := ""
	if s.ProjectID != nil {
		pid = *s.ProjectID
	}
	return &Secret{ID: s.ID, Key: s.Key, Value: s.Value, Note: s.Note, ProjectID: pid}, nil
}

func (c *SDKClient) CreateSecret(key, value, note, orgID string, projectIDs []string) (*Secret, error) {
	s, err := c.inner.Secrets().Create(key, value, note, orgID, projectIDs)
	if err != nil {
		return nil, err
	}
	pid := ""
	if s.ProjectID != nil {
		pid = *s.ProjectID
	}
	return &Secret{ID: s.ID, Key: s.Key, Value: s.Value, Note: s.Note, ProjectID: pid}, nil
}

func (c *SDKClient) UpdateSecret(id, key, value, note, orgID string, projectIDs []string) (*Secret, error) {
	s, err := c.inner.Secrets().Update(id, key, value, note, orgID, projectIDs)
	if err != nil {
		return nil, err
	}
	pid := ""
	if s.ProjectID != nil {
		pid = *s.ProjectID
	}
	return &Secret{ID: s.ID, Key: s.Key, Value: s.Value, Note: s.Note, ProjectID: pid}, nil
}

func (c *SDKClient) DeleteSecrets(ids []string) error {
	_, err := c.inner.Secrets().Delete(ids)
	return err
}

func (c *SDKClient) Close() {
	c.inner.Close()
}
