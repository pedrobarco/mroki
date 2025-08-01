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
