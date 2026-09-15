package common

import (
	"os/user"
	"strings"
)

func ExpandHome(path string) string {
	if path == "~" {
		usr, err := user.Current()
		if err == nil {
			return usr.HomeDir
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		usr, err := user.Current()
		if err == nil {
			return usr.HomeDir + path[1:]
		}
	}
	return path
}
