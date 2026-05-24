package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/eichemberger/burrow/internal/session"
	"github.com/eichemberger/burrow/internal/targetstore"
)

func RunStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	burrowDir := fs.String("burrow-dir", targetstore.DefaultDir(), "Path to burrow config directory")
	asJSON := fs.Bool("json", false, "Output JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	reg, err := session.Open(*burrowDir)
	if err != nil {
		return err
	}

	entries, err := reg.List()
	if err != nil {
		return err
	}

	if *asJSON {
		return writeStatusJSON(os.Stdout, entries)
	}
	return writeStatusTable(os.Stdout, entries)
}

func RunStop(args []string) error {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	burrowDir := fs.String("burrow-dir", targetstore.DefaultDir(), "Path to burrow config directory")
	stopAll := fs.Bool("all", false, "Stop all active sessions")
	if err := fs.Parse(args); err != nil {
		return err
	}

	reg, err := session.Open(*burrowDir)
	if err != nil {
		return err
	}

	if *stopAll {
		n, err := session.StopAll(reg)
		if err != nil {
			return err
		}
		if n == 0 {
			fmt.Println("No active burrow sessions.")
		} else {
			fmt.Printf("Stopped %d session(s).\n", n)
		}
		return nil
	}

	ref := strings.TrimSpace(fs.Arg(0))
	if ref == "" {
		return fmt.Errorf("usage: burrow stop <alias|id> (or --all)")
	}

	rec, err := reg.Resolve(ref)
	if err != nil {
		return err
	}

	wasAlive := session.IsAlive(rec)
	if err := session.Stop(reg, rec); err != nil {
		return err
	}

	if wasAlive {
		fmt.Printf("Stopped session %q (%s).\n", rec.Alias, rec.ID)
	} else {
		fmt.Printf("Removed stale session record %q (%s).\n", rec.Alias, rec.ID)
	}
	return nil
}

func writeStatusJSON(w io.Writer, entries []session.Entry) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

func writeStatusTable(w io.Writer, entries []session.Entry) error {
	if len(entries) == 0 {
		fmt.Fprintln(w, "No active burrow sessions.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ALIAS\tID\tPID\tLOCAL\tREMOTE\tBASTION\tREGION\tUPTIME\tSTATE")
	for _, e := range entries {
		fmt.Fprintf(tw, "%s\t%s\t%d\tlocalhost:%d\t%s:%d\t%s\t%s\t%s\t%s\n",
			e.Alias,
			e.ID,
			e.PID,
			e.LocalPort,
			e.Host,
			e.RemotePort,
			e.BastionID,
			e.Region,
			session.FormatDuration(time.Duration(e.UptimeSeconds)*time.Second),
			e.State,
		)
	}
	return tw.Flush()
}
