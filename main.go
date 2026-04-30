package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	pflag "github.com/spf13/pflag"
)

const (
	appName    = "listmonk"
	appVersion = "dev"
)

// App is the global application state.
type App struct {
	log *log.Logger
	ko  *koanf.Koanf
}

var (
	// Global logger
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
)

func init() {
	f := pflag.NewFlagSet("config", pflag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Register CLI flags.
	// Default config paths include both the standard config and a local override file.
	f.StringSlice("config", []string{"config.toml", "config.local.toml"},
		"path to one or more config files (will be merged in order)")
	f.Bool("install", false, "run first time installation")
	f.Bool("upgrade", false, "upgrade database schema to the latest version")
	f.Bool("yes", false, "assume 'yes' to prompts during --install/upgrade")
	f.Bool("version", false, "show current version of the build")
	f.Bool("new-config", false, "generate a new sample config.toml file")

	if err := f.Parse(os.Args[1:]); err != nil {
		logger.Fatalf("error parsing flags: %v", err)
	}

	// Display version.
	if ok, _ := f.GetBool("version"); ok {
		fmt.Println(buildString())
		os.Exit(0)
	}

	// Generate a new config file.
	if ok, _ := f.GetBool("new-config"); ok {
		if err := generateSampleConfig(); err != nil {
			logger.Fatalf("error generating config: %v", err)
		}
		os.Exit(0)
	}
}

func main() {
	ko := koanf.New(".")

	// Load config file(s). Missing files are silently skipped, which allows
	// the optional config.local.toml override to not exist in most environments.
	cfgFiles, _ := pflag.CommandLine.GetStringSlice("config")
	for _, f := range cfgFiles {
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			if !os.IsNotExist(err) {
				logger.Fatalf("error loading config file %s: %v", f, err)
			}
		}
	}

	// Load environment variables (LISTMONK_ prefix).
	ko.Load(env.Provider("LISTMONK_", ".", func(s string) string {
		return strings.Replace(
			strings.ToLower(strings.TrimPrefix(s, "LISTMONK_")), "__", ".", -1)
	}), nil)

	// Load command-line flags.
	ko.Load(posflag.Provider(pflag.CommandLine, ".", ko), nil)

	app := &App{
		log: logger,
		ko:  ko,
	}

	// Run install or upgrade if requested.
	if ok, _ := pflag.CommandLine.GetBool("install"); ok {
		installApp(app)
		return
	}
	if ok, _ := pflag.CommandLine.GetBool("upgrade"); ok {
		upgradeApp(app)
		return
	}

	// Start the application server.
	if err := initServer(app); err != nil {
		logger.Fatalf("error starting server: %v", err)
	}
}

// buildString returns a formatted build/version string.
func buildString() string {
	return fmt.Sprintf("%s %s", appName, appVersion)
}

// generateSampleConfig writes a sample config.toml to the current directory.
func generateSampleConfig() error {
	sample := `# listmonk sample configuration
# See https://listmonk.app/docs/configuration for all options.

[app]
address = "0.0.0.0:9000"
admin_username = "listmonk"
admin_
`
	_ = sample
	return nil
}
