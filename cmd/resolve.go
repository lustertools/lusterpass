package cmd

import (
	"fmt"
	"strings"

	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/config"
)

// resolveAccount determines which account to use.
// Priority: config file account field > global active account > error.
func resolveAccount(cfg *config.Config) (string, error) {
	// 1. YAML account field
	if cfg != nil && cfg.Account != "" {
		if err := auth.ValidateAccountName(cfg.Account); err != nil {
			return "", fmt.Errorf("invalid account in config: %w", err)
		}
		return cfg.Account, nil
	}

	// 2. Global active account
	account, err := auth.LoadActiveAccount()
	if err == nil {
		return account, nil
	}

	// 3. Error with helpful message
	accounts, _ := auth.ListAccounts()
	if len(accounts) > 0 {
		return "", fmt.Errorf("no active account. Available: %s\nRun: lusterpass account use <name>", strings.Join(accounts, ", "))
	}
	return "", fmt.Errorf("no accounts configured. Run: lusterpass login --account <name>")
}
