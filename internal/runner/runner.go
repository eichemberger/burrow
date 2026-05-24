package runner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/eichemberger/burrow/internal/apictx"
	"github.com/eichemberger/burrow/internal/awsconfig"
	"github.com/eichemberger/burrow/internal/netutil"
	"github.com/eichemberger/burrow/internal/session"
	"github.com/eichemberger/burrow/internal/ssmexec"
	"github.com/eichemberger/burrow/internal/targetstore"
)

type ConnectOptions struct {
	LocalPort int  // 0 keeps the saved target's local port
	Print     bool // print aws command instead of running it
}

func Connect(dir, alias string, connectOpts ConnectOptions) error {
	target, opts, err := loadTarget(dir, alias)
	if err != nil {
		return err
	}

	opts, err = applyLocalPortOverride(opts, connectOpts.LocalPort)
	if err != nil {
		return err
	}

	if connectOpts.Print {
		return printCommand(opts)
	}

	if err := runPreflight(target, opts); err != nil {
		return formatConnectError(err)
	}

	cmd, err := ssmexec.BuildCommand(opts)
	if err != nil {
		return err
	}

	fmt.Println(ssmexec.Summary(opts))
	return runSessionCommand(opts.TargetInstanceID, cmd)
}

func ConnectBackground(dir, alias string, connectOpts ConnectOptions) error {
	if runtime.GOOS == "windows" {
		return session.ErrUnsupported
	}

	target, opts, err := loadTarget(dir, alias)
	if err != nil {
		return err
	}

	opts, err = applyLocalPortOverride(opts, connectOpts.LocalPort)
	if err != nil {
		return err
	}

	if connectOpts.Print {
		return printCommand(opts)
	}

	if err := runPreflight(target, opts); err != nil {
		return formatConnectError(err)
	}

	cmd, err := ssmexec.BuildCommand(opts)
	if err != nil {
		return err
	}

	rec, err := session.SpawnDetached(cmd, session.SpawnInput{
		BurrowDir:  dir,
		Alias:      alias,
		LocalPort:  opts.LocalPort,
		Host:       opts.Host,
		RemotePort: opts.RemotePort,
		BastionID:  opts.TargetInstanceID,
		Region:     opts.Region,
		Profile:    opts.Profile,
		UseEnv:     opts.UseEnv,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Session %q running on localhost:%d (pid %d, id %s).\n",
		rec.Alias, rec.LocalPort, rec.PID, rec.ID)
	fmt.Printf("Stop with: burrow stop %s\n", rec.Alias)
	return nil
}

func applyLocalPortOverride(opts ssmexec.Options, localPort int) (ssmexec.Options, error) {
	if localPort == 0 {
		return opts, nil
	}
	if localPort < 1 || localPort > 65535 {
		return ssmexec.Options{}, fmt.Errorf("invalid local port: %d", localPort)
	}
	opts.LocalPort = localPort
	return opts, nil
}

func printCommand(opts ssmexec.Options) error {
	cmd, err := ssmexec.FormatCommand(opts)
	if err != nil {
		return err
	}
	fmt.Println(cmd)
	return nil
}

func loadTarget(dir, alias string) (targetstore.Target, ssmexec.Options, error) {
	store, err := targetstore.Load(dir)
	if err != nil {
		return targetstore.Target{}, ssmexec.Options{}, err
	}

	target, err := store.Get(alias)
	if err != nil {
		return targetstore.Target{}, ssmexec.Options{}, err
	}

	return target, target.ToSSMExec(), nil
}

func runPreflight(target targetstore.Target, opts ssmexec.Options) error {
	authCtx, authCancel := apictx.AuthBackground()
	defer authCancel()

	cfg, err := awsconfig.Load(authCtx, awsconfig.Options{
		Profile: target.AWSProfile,
		Region:  target.Region,
		UseEnv:  target.UseEnv,
	})
	if err != nil {
		return apictx.WrapDeadline(err, "load AWS configuration")
	}

	ctx, cancel := apictx.Background()
	defer cancel()

	if err := ssmexec.VerifyInstanceOnline(ctx, cfg, opts.TargetInstanceID); err != nil {
		return err
	}

	return netutil.LocalPortAvailable(opts.LocalPort)
}

func runSessionCommand(instanceID string, cmd *exec.Cmd) error {
	var stderr bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if err := cmd.Run(); err != nil {
		return formatConnectError(ssmexec.ClassifyRunError(instanceID, stderr.String(), err))
	}
	return nil
}

func formatConnectError(err error) error {
	var failure ssmexec.RunFailure
	if errors.As(err, &failure) {
		return fmt.Errorf("%w\n\nRun burrow without --target to create a new connection.", err)
	}
	return err
}
