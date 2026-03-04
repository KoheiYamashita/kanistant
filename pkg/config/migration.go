package config

// ConfigVersion is the current config schema version.
// Increment this when adding new fields that must appear in existing config files.
const ConfigVersion = 1

// migrateConfig runs version-based migrations on cfg.
// Returns true if migrations were applied and config should be re-saved.
func migrateConfig(cfg *Config) bool {
	if cfg.Version >= ConfigVersion {
		return false
	}

	migrations := []func(*Config){
		migrateV0ToV1,
	}

	for i := cfg.Version; i < ConfigVersion && i < len(migrations); i++ {
		migrations[i](cfg)
	}

	cfg.Version = ConfigVersion
	return true
}

func migrateV0ToV1(cfg *Config) {
	// queue_messages: Go zero value (false) is the correct default.
	// Version bump + re-save writes the new field into config.json.
}
