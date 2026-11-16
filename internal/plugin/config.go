package plugin

import "github.com/wmentor/go-magnetar/internal/config"

// Config defines the configuration access methods required by plugins.
type Config interface { //nolint:interfacebloat // Config interface mirrors configuration access methods required across plugins.
	Bool(key string) bool
	Copy() *config.Config
	Float64(key string) float64
	Int(key string) int
	Profile() string
	ProfileParamBool(key string) bool
	ProfileParamFloat64(key string) float64
	ProfileParamInt(key string) int
	ProfileParamString(key string) string
	Profiles() []string
	SetProfile(profile string) error
	String(key string) string
	Strings(key string) []string
}
