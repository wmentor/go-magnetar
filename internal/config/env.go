package config

import (
	"os"
	"regexp"
)

var envVarPattern = regexp.MustCompile(`\$env:([A-Za-z_][A-Za-z0-9_]*)`)
var fileVarPattern = regexp.MustCompile(`\$file:([^$\s]+)`)

func ResolveEnvVars(value string) string {
	value = envVarPattern.ReplaceAllStringFunc(value, func(match string) string {
		varName := envVarPattern.ReplaceAllString(match, "$1")
		return os.Getenv(varName)
	})

	return fileVarPattern.ReplaceAllStringFunc(value, func(match string) string {
		filename := fileVarPattern.ReplaceAllString(match, "$1")
		content, err := os.ReadFile(filename)
		if err != nil {
			return ""
		}
		return string(content)
	})
}
