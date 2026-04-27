package bitwarden

import "testing"

func TestMockCreateAndGet(t *testing.T) {
	client := NewMockClient()

	created, err := client.CreateSecret("test-key", "test-value", "note", "org1", []string{"proj1"})
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	got, err := client.GetSecret(created.ID)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if got.Value != "test-value" {
		t.Errorf("expected test-value, got %s", got.Value)
	}
}

func TestMockListAndDelete(t *testing.T) {
	client := NewMockClient()
	client.CreateSecret("key1", "val1", "", "org1", nil)
	client.CreateSecret("key2", "val2", "", "org1", nil)

	secrets, _ := client.ListSecrets("org1")
	if len(secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(secrets))
	}

	client.DeleteSecrets([]string{"mock-key1"})

	secrets, _ = client.ListSecrets("org1")
	if len(secrets) != 1 {
		t.Errorf("expected 1 secret after delete, got %d", len(secrets))
	}
}

func TestMockGetSecretByKey(t *testing.T) {
	client := NewMockClient()
	client.CreateSecret("my-ref-name", "secret-val", "", "org1", nil)

	got, err := client.GetSecretByKey("my-ref-name")
	if err != nil {
		t.Fatalf("GetSecretByKey failed: %v", err)
	}

	if got.Value != "secret-val" {
		t.Errorf("expected secret-val, got %s", got.Value)
	}

	_, err = client.GetSecretByKey("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent key")
	}
}

func TestMockListProjects(t *testing.T) {
	client := NewMockClient()
	client.Projects["p1"] = &Project{ID: "p1", Name: "credentials"}
	client.Projects["p2"] = &Project{ID: "p2", Name: "testing"}

	projects, err := client.ListProjects("org1")
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}
