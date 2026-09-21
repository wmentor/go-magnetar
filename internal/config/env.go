package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var envVarPattern = regexp.MustCompile(`\$env:([A-Za-z_][A-Za-z0-9_]*)`)
var fileVarPattern = regexp.MustCompile(`\$file:([^$\s]+)`)

func ResolveEnvVars(value string, baseDir ...string) string {
	value = envVarPattern.ReplaceAllStringFunc(value, func(match string) string {
		varName := envVarPattern.ReplaceAllString(match, "$1")
		return os.Getenv(varName)
	})

	return fileVarPattern.ReplaceAllStringFunc(value, func(match string) string {
		filename := fileVarPattern.ReplaceAllString(match, "$1")

		if strings.HasPrefix(filename, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				filename = filepath.Join(home, filename[2:])
			}
		}

		if len(baseDir) > 0 && baseDir[0] != "" && !filepath.IsAbs(filename) {
			filename = filepath.Join(baseDir[0], filename)
		}

		content, err := os.ReadFile(filename)
		if err != nil {
			return ""
		}
		return string(content)
	})
}
