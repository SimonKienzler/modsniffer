package config

type ModSnifferConfig struct {
	PreferredGoVersion string            `yaml:"preferredGoVersion"`
	RelevantPackages   []RelevantPackage `yaml:"relevantPackages"`
}

type RelevantPackage struct {
	Name             string `yaml:"name"`
	PreferredVersion string `yaml:"preferredVersion"`
}
