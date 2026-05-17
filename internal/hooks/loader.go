package hooks

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/skeeeon/pb-wiki/internal/access"
)

// LoadRules returns all access_rules ordered by ascending priority (first
// matching rule wins). Limit 0 means no SQL LIMIT clause — we fetch all rules
// because the evaluator walks them in order; rule counts are expected to be
// small (tens at most for a real wiki).
func LoadRules(app core.App) ([]access.Rule, error) {
	records, err := app.FindRecordsByFilter("access_rules", "", "+priority", 0, 0)
	if err != nil {
		return nil, err
	}
	rules := make([]access.Rule, len(records))
	for i, r := range records {
		rules[i] = access.Rule{
			Pattern: r.GetString("pattern"),
			Access:  r.GetString("access"),
			Groups:  r.GetStringSlice("groups"),
		}
	}
	return rules, nil
}

// ConfigFlags is the bundle of wiki_config booleans the access evaluator
// needs. Loaded together because each access check needs both — saves a
// duplicate FindFirstRecordByFilter call per request.
type ConfigFlags struct {
	PrivateDefault bool
	RequireLogin   bool
}

// LoadConfigFlags returns the access-relevant flags from wiki_config.
func LoadConfigFlags(app core.App) (ConfigFlags, error) {
	cfg, err := app.FindFirstRecordByFilter("wiki_config", "")
	if err != nil {
		return ConfigFlags{}, err
	}
	return ConfigFlags{
		PrivateDefault: cfg.GetBool("private_default"),
		RequireLogin:   cfg.GetBool("require_login"),
	}, nil
}

// RecordToUser projects an auth record onto the minimal shape the access
// evaluator needs. Returns nil for anonymous requests.
func RecordToUser(r *core.Record) *access.User {
	if r == nil {
		return nil
	}
	return &access.User{
		Role:   r.GetString("role"),
		Groups: r.GetStringSlice("groups"),
	}
}
