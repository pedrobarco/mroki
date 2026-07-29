package config

type AppEnv string

const (
	appEnvDevelopment AppEnv = "development"
	appEnvProduction  AppEnv = "production"
)

func (e AppEnv) IsDevelopment() bool {
	return e == appEnvDevelopment
}

func (e AppEnv) IsProduction() bool {
	return e == appEnvProduction
}

// EffectiveLogLevel returns the log level to use. A non-empty explicit value
// always wins; otherwise the level is derived from the environment (production:
// info, development: debug).
func (e AppEnv) EffectiveLogLevel(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if e.IsProduction() {
		return "info"
	}
	return "debug"
}

// EffectiveLogFormat returns the log format to use. A non-empty explicit value
// always wins; otherwise the format is derived from the environment (production:
// json, development: text).
func (e AppEnv) EffectiveLogFormat(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if e.IsProduction() {
		return "json"
	}
	return "text"
}
