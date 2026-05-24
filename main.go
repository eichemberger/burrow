package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/awsconfig"
	"github.com/eichemberger/burrow/internal/cli"
	"github.com/eichemberger/burrow/internal/debuglog"
	"github.com/eichemberger/burrow/internal/runner"
	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/tui"

	_ "github.com/eichemberger/burrow/internal/services/elasticache"
	_ "github.com/eichemberger/burrow/internal/services/opensearch"
	_ "github.com/eichemberger/burrow/internal/services/rds"
)

// Set at link time: go build -ldflags="-X main.version=v1.2.3"
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "status":
			if err := cli.RunStatus(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			return
		case "stop":
			if err := cli.RunStop(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	awsDir := flag.String("aws-dir", awsconfig.DefaultAWSDir(), "Path to AWS shared config directory")
	burrowDir := flag.String("burrow-dir", targetstore.DefaultDir(), "Path to burrow config directory")
	profile := flag.String("profile", "", "AWS profile to use (skips profile picker)")
	region := flag.String("region", "", "AWS region (skips region picker)")
	target := flag.String("target", "", "Saved target alias to connect (skips TUI)")
	localPort := flag.Int("local-port", 0, "Override local port for --target (1-65535)")
	printCmd := flag.Bool("print", false, "Print the aws ssm start-session command instead of running it (requires --target)")
	background := flag.Bool("background", false, "Run the session in the background and return to the shell (requires --target)")
	listTargets := flag.Bool("list-targets", false, "List saved targets and exit")
	showTarget := flag.String("show-target", "", "Show a saved target and exit")
	deleteTarget := flag.String("delete-target", "", "Delete a saved target and exit")
	manage := flag.Bool("manage", false, "Open the connection manager TUI")
	debug := flag.Bool("debug", false, "Enable debug logging")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		writeVersion(os.Stdout)
		return
	}

	closeDebug, err := initDebugLogging(*debug, *burrowDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not open debug log: %v\n", err)
	}
	defer closeDebug()

	if *listTargets {
		if err := printTargets(*burrowDir); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *showTarget != "" {
		if err := showTargetConfig(*burrowDir, *showTarget); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *deleteTarget != "" {
		if err := deleteTargetConfig(*burrowDir, *deleteTarget); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted target %q\n", *deleteTarget)
		return
	}

	if *localPort != 0 && *target == "" {
		fmt.Fprintln(os.Stderr, "error: --local-port requires --target")
		os.Exit(1)
	}

	if *printCmd && *target == "" {
		fmt.Fprintln(os.Stderr, "error: --print requires --target")
		os.Exit(1)
	}

	if *printCmd && *background {
		fmt.Fprintln(os.Stderr, "error: --print cannot be used with --background")
		os.Exit(1)
	}

	connectOpts := runner.ConnectOptions{
		LocalPort: *localPort,
		Print:     *printCmd,
	}

	if *target != "" {
		if *background {
			if err := runner.ConnectBackground(*burrowDir, *target, connectOpts); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if err := runner.Connect(*burrowDir, *target, connectOpts); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *background {
		fmt.Fprintln(os.Stderr, "error: --background requires --target")
		os.Exit(1)
	}

	opts := tui.Options{
		AWSDir:    *awsDir,
		BurrowDir: *burrowDir,
		Profile:   *profile,
		Region:    *region,
		Manage:    *manage,
	}

	if err := tui.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printTargets(dir string) error {
	store, err := targetstore.Load(dir)
	if err != nil {
		return err
	}
	aliases := store.Aliases()
	if len(aliases) == 0 {
		fmt.Println("No saved targets.")
		return nil
	}
	all := store.All()
	for _, alias := range aliases {
		fmt.Println(all[alias].Summary(alias))
	}
	return nil
}

func showTargetConfig(dir, alias string) error {
	store, err := targetstore.Load(dir)
	if err != nil {
		return err
	}
	target, err := store.Get(alias)
	if err != nil {
		return err
	}
	fmt.Println(target.Summary(alias))
	return nil
}

func setupDebugLogging(burrowDir string) (*os.File, error) {
	if err := os.MkdirAll(burrowDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(burrowDir, "burrow-debug.log")
	return tea.LogToFile(path, "burrow")
}

func initDebugLogging(debug bool, burrowDir string) (func(), error) {
	log.SetOutput(io.Discard)
	if !debug {
		return func() {}, nil
	}

	f, err := setupDebugLogging(burrowDir)
	if err != nil {
		return func() {}, err
	}

	debuglog.SetEnabled(true)
	return func() { f.Close() }, nil
}

func deleteTargetConfig(dir, alias string) error {
	store, err := targetstore.Load(dir)
	if err != nil {
		return err
	}
	return store.Delete(alias)
}

func writeVersion(w io.Writer) {
	fmt.Fprintf(w, "burrow %s\n", version)
}
