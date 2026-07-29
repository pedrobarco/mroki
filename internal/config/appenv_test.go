package config

import "testing"

func TestAppEnv_IsDevelopment(t *testing.T) {
	tests := []struct {
		env      AppEnv
		expected bool
	}{
		{appEnvDevelopment, true},
		{appEnvProduction, false},
		{AppEnv("staging"), false},
		{AppEnv(""), false},
	}

	for _, tt := range tests {
		result := tt.env.IsDevelopment()
		if result != tt.expected {
			t.Errorf("env=%q: expected IsDevelopment()=%v, got %v", tt.env, tt.expected, result)
		}
	}
}

func TestAppEnv_IsProduction(t *testing.T) {
	tests := []struct {
		env      AppEnv
		expected bool
	}{
		{appEnvProduction, true},
		{appEnvDevelopment, false},
		{AppEnv("staging"), false},
		{AppEnv(""), false},
	}

	for _, tt := range tests {
		result := tt.env.IsProduction()
		if result != tt.expected {
			t.Errorf("env=%q: expected IsProduction()=%v, got %v", tt.env, tt.expected, result)
		}
	}
}

func TestAppEnv_IsValid(t *testing.T) {
	tests := []struct {
		env      AppEnv
		expected bool
	}{
		{appEnvDevelopment, true},
		{appEnvProduction, true},
		{AppEnv("staging"), false},
		{AppEnv("prod"), false},
		{AppEnv(""), false},
	}

	for _, tt := range tests {
		result := tt.env.IsValid()
		if result != tt.expected {
			t.Errorf("env=%q: expected IsValid()=%v, got %v", tt.env, tt.expected, result)
		}
	}
}
