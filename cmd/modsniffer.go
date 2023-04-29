package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/brumhard/alligotor"
	"github.com/simonkienzler/modsniffer/internal/log"
	"github.com/simonkienzler/modsniffer/pkg/config"
	"github.com/simonkienzler/modsniffer/pkg/scorer"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfgFile string
	verbose bool

	modsnifferCmd = &cobra.Command{
		Use:   "modsniffer [local path or remote git URL] [flags]",
		Short: "Analyses the contents of your go.mod",
		Long: `modsniffer helps you keeping your projects' dependencies up to date
by checking imported packages in your go.mod against a config file.`,
		Args: args,
		Run:  Modsniffer,
	}
)

func Execute() error {
	return modsnifferCmd.Execute()
}

func init() {
	home := os.Getenv("HOME")
	modsnifferCmd.PersistentFlags().StringVar(&cfgFile, "config", home+"/.modsniffer.yaml", "config file")
	modsnifferCmd.PersistentFlags().BoolVar(&verbose, "verbose", true, "print detailed scores per package")
}

func args(cmd *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}

	return validate(args[0])
}

func Modsniffer(cmd *cobra.Command, args []string) {
	conf := config.ModSnifferConfig{}
	// TODO if the given config file doesn't exist, we continue. Should abort instead
	alligotor.New(alligotor.NewFilesSource(cfgFile)).Get(&conf)

	logger, err := log.NewAtLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred: %s\n", err)
		os.Exit(1)
	}

	defer func() {
		err = logger.Sync()
	}()

	logger.Debug("Argument provided", zap.String("path", args[0]))

	goModFile, err := getGoModFileFromPathArgument(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred: %s\n", err)
		os.Exit(1)
	}

	goVersion, err := semver.NewVersion(conf.PreferredGoVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred: %s\n", err)
		os.Exit(1)
	}

	scorerSvc := scorer.Service{
		Logger:             logger,
		GoModFileList:      []string{goModFile},
		RelevantPackages:   conf.RelevantPackages,
		PreferredGoVersion: *goVersion,
	}

	scorerSvc.PerformGoModAnalysis(verbose)
}

func getGoModFileFromPathArgument(path string) (string, error) {
	if isRemoteGitPath(path) {
		// TODO handle remote repo path, this is currently caught by validate()
		return "", nil
	}

	// user convenience: it shouldn't matter if the path has a trailing `go.mod`
	if strings.HasSuffix(path, "/go.mod") {
		return path, nil
	}

	if strings.HasSuffix(path, "/") {
		return path + "go.mod", nil
	}

	return path + "/go.mod", nil
}

func validate(arg string) error {
	// we support remote git repos with ssh or https
	if isRemoteGitPath(arg) {
		return fmt.Errorf("remote repository paths are not yet supported")
	}

	// if no remote repo is given, we assume a local, accessible file
	path, err := getGoModFileFromPathArgument(arg)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	return fmt.Errorf(`The path either needs to be a remote git repository (starting with 'ssh://', 'git@', 'https://') or a path to an existing local directory containing a go.mod file`)
}

func isRemoteGitPath(path string) bool {
	return strings.HasPrefix(path, "ssh://") || strings.HasPrefix(path, "git@") || strings.HasPrefix(path, "https://")
}
