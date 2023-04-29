package scorer

import (
	"bytes"
	"fmt"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/simonkienzler/modsniffer/pkg/api"
	"github.com/simonkienzler/modsniffer/pkg/config"
	"go.uber.org/zap"
	"golang.org/x/mod/modfile"
	"k8s.io/utils/pointer"
)

// TODO this is evil when used as input for other CLI tools. Use proper library
// for colored output.
var (
	FontReset  = "\033[0m"
	FontRed    = "\033[31m"
	FontGreen  = "\033[32m"
	FontBlue   = "\033[34m"
	FontPurple = "\033[35m"
	FontCyan   = "\033[36m"
	FontGray   = "\033[37m"
	FontBold   = "\033[1m"
)

type Service struct {
	Logger             *zap.Logger
	PreferredGoVersion semver.Version
	GoModFileList      []string
	RelevantPackages   []config.RelevantPackage
}

func (s *Service) PerformGoModAnalysis(verbose bool) error {
	var buffer bytes.Buffer

	for i := range s.GoModFileList {
		modFileRaw := s.GoModFileList[i]
		content, err := GetFileContent(modFileRaw)
		if err != nil {
			return err
		}

		modFile, err := modfile.Parse("go.mod", content, nil)
		if err != nil {
			panic(err)
		}

		pkgs := []*api.Package{}

		for _, pkg := range s.RelevantPackages {
			pkg, err := GetPackage(&pkg, modFile)
			if err != nil {
				continue
			}
			pkgs = append(pkgs, pkg)
		}

		// TODO seperate the actual analysis from the printing, so that the CLI
		// can support arbirtary output formats (e.g. pretty, JSON, YAML, TSV)
		analysis, finalScore, err := s.PrintPackageAnalysis(modFile, pkgs)
		if err != nil {
			return err
		}
		if verbose {
			buffer.WriteString(formatHeader(modFile.Module.Mod.Path))
			buffer.WriteString(analysis)
			buffer.WriteString(formatFinalScore(finalScore))
		} else {
			buffer.WriteString(fmt.Sprintf("{\"name\": \"%s\", \"score\": %d},", modFile.Module.Mod.Path, finalScore))
		}
	}
	fmt.Printf("%s\n", buffer.String())
	return nil
}

func GetPackage(pkg *config.RelevantPackage, modFile *modfile.File) (*api.Package, error) {
	if modFile == nil {
		return nil, fmt.Errorf("no modfile provided")
	}

	if pkg == nil {
		return nil, fmt.Errorf("no pkg provided")
	}

	if modFile.Replace != nil {
		for i := range modFile.Replace {
			if modFile.Replace[i].Old.Path == pkg.Name {
				version, err := semver.NewVersion(modFile.Replace[i].New.Version)
				if err != nil {
					return nil, err
				}
				prefVersion, err := semver.NewVersion(pkg.PreferredVersion)
				if err != nil {
					return nil, err
				}
				return &api.Package{
					Name:             modFile.Replace[i].Old.Path,
					Version:          *version,
					PreferredVersion: *prefVersion,
					Replaced:         true,
					Replacement:      pointer.String(modFile.Replace[i].New.Path),
				}, nil
			}
		}
	}

	if modFile.Require != nil {
		for i := range modFile.Require {
			if modFile.Require[i].Mod.Path == pkg.Name {
				version, err := semver.NewVersion(modFile.Require[i].Mod.Version)
				if err != nil {
					return nil, err
				}
				prefVersion, err := semver.NewVersion(pkg.PreferredVersion)
				if err != nil {
					return nil, err
				}
				return &api.Package{
					Name:             modFile.Require[i].Mod.Path,
					Version:          *version,
					PreferredVersion: *prefVersion,
					Replaced:         false,
					Replacement:      nil,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("no pkg found with name %s", pkg.Name)
}

func (s *Service) PrintPackageAnalysis(modFile *modfile.File, pkgs []*api.Package) (string, uint64, error) {
	goVersion, err := semver.NewVersion(modFile.Go.Version)
	if err != nil {
		return "", 0, err
	}

	goScore := scoreVersionDiff(*goVersion, s.PreferredGoVersion)
	finalScore := goScore

	var buffer bytes.Buffer

	buffer.WriteString(formatPackage("Go Version", "", goVersion.String(), s.PreferredGoVersion.String(), goScore))

	for _, pkg := range pkgs {
		if pkg != nil {
			score := scoreVersionDiff(pkg.Version, pkg.PreferredVersion)
			finalScore += score
			if pkg.Replaced {
				buffer.WriteString(formatPackage(pkg.Name, *pkg.Replacement, pkg.Version.String(), pkg.PreferredVersion.String(), score))
				continue
			}
			buffer.WriteString(formatPackage(pkg.Name, "", pkg.Version.String(), pkg.PreferredVersion.String(), score))
		}
	}

	return buffer.String(), finalScore, nil
}

func formatHeader(moduleName string) string {
	return fmt.Sprintf("\n  "+FontPurple+FontBold+"%s"+FontReset+"\n", moduleName)
}

func formatPackage(name, replacement, actual, preferred string, score uint64) string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("\n  » "+FontBlue+"%s"+FontReset+" ", name))
	if replacement != "" {
		buffer.WriteString(fmt.Sprintf("(replaced by "+FontCyan+"%s"+FontReset+")\n", replacement))
	} else {
		buffer.WriteString("\n")
	}
	buffer.WriteString(fmt.Sprintf("    └─ actual:    "+FontRed+"%s"+FontReset+"\n", actual))
	buffer.WriteString(fmt.Sprintf("    └─ preferred: "+FontGreen+"%s"+FontReset+"\n", preferred))
	buffer.WriteString(fmt.Sprintf("    └─ Score:     "+FontCyan+"%d"+FontReset, score))

	return buffer.String()
}

func formatFinalScore(score uint64) string {
	return fmt.Sprintf("\n\n  "+FontPurple+FontBold+"Final Score: %d"+FontReset+"\n", score)
}

func scoreVersionDiff(actual, preferred semver.Version) uint64 {
	// the pkg in use is newer than the recommendation, great
	if actual.Major() > preferred.Major() {
		return 0
	}

	if actual.Major() < preferred.Major() {
		return (preferred.Major() - actual.Major()) * 100
	}

	// major version matches with the preferred one, check minor
	if actual.Minor() > preferred.Minor() {
		return 0
	}

	if actual.Minor() < preferred.Minor() {
		return (preferred.Minor() - actual.Minor()) * 10
	}

	// minor version matches with preferred, let's check patch if given
	if actual.Patch() < preferred.Patch() {
		return preferred.Patch() - actual.Patch()
	}

	return 0
}

func GetFileContent(filePath string) ([]byte, error) {
	f, err := os.ReadFile(filePath)
	if err != nil {
		return []byte(""), err
	}

	return f, nil
}
