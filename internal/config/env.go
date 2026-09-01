package config

import (
	"os"
	"regexp"
)

var envVarPattern = regexp.MustCompile(`\$env:([A-Za-z_][A-Za-z0-9_]*)`)

func ResolveEnvVars(value string) string {
	return envVarPattern.ReplaceAllStringFunc(value, func(match string) string {
		varName := envVarPattern.ReplaceAllString(match, "$1")
		return os.Getenv(varName)
	})
}
