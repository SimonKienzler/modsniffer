package api

import (
	semver "github.com/Masterminds/semver/v3"
)

type Package struct {
	Name             string
	Version          semver.Version
	PreferredVersion semver.Version
	Replaced         bool
	Replacement      *string
}
