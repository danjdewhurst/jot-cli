package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/danjdewhurst/jot-cli/internal/sync"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronise notes with a shared directory",
	Long: `Push local changes and pull remote changes from a sync directory.

The sync directory can be any shared folder (Dropbox, Syncthing, iCloud Drive,
a mounted network share, etc.). Set it with --sync-dir or JOT_SYNC_DIR.`,
	RunE: runSync,
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status",
	RunE:  runSyncStatus,
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push local changes to sync directory",
	RunE:  runSyncPush,
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull remote changes from sync directory",
	RunE:  runSyncPull,
}

func init() {
	syncCmd.PersistentFlags().String("sync-dir", "",
		"Sync directory path (default: $JOT_SYNC_DIR or XDG data dir)")
	syncCmd.AddCommand(syncStatusCmd, syncPushCmd, syncPullCmd)
	rootCmd.AddCommand(syncCmd)
}

func newSyncer(cmd *cobra.Command) *sync.Syncer {
	syncDir, _ := cmd.Flags().GetString("sync-dir")
	if syncDir == "" {
		syncDir = cfg.SyncDir
	}
	return sync.New(db, syncDir)
}

func runSync(cmd *cobra.Command, args []string) error {
	s := newSyncer(cmd)
	result, err := s.Sync()
	if err != nil {
		return err
	}

	if flagJSON {
		return render.JSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stderr, "Synced: pushed %d, pulled %d", result.Pushed, result.Pulled)
	if result.Conflicts > 0 {
		fmt.Fprintf(os.Stderr, " (%d conflicts — local version kept)", result.Conflicts)
	}
	fmt.Fprintln(os.Stderr)
	return nil
}

func runSyncStatus(cmd *cobra.Command, args []string) error {
	s := newSyncer(cmd)
	status, err := s.Status()
	if err != nil {
		return err
	}

	if flagJSON {
		return render.JSON(os.Stdout, status)
	}

	fmt.Fprintf(os.Stderr, "Pending changes: %d\n", status.Pending)
	if status.LastSync.IsZero() {
		fmt.Fprintln(os.Stderr, "Last sync: never")
	} else {
		fmt.Fprintf(os.Stderr, "Last sync: %s\n", status.LastSync.Format("2006-01-02 15:04:05 UTC"))
	}
	return nil
}

func runSyncPush(cmd *cobra.Command, args []string) error {
	s := newSyncer(cmd)
	pushed, err := s.Push()
	if err != nil {
		return err
	}

	if flagJSON {
		return render.JSON(os.Stdout, map[string]int{"pushed": pushed})
	}

	fmt.Fprintf(os.Stderr, "Pushed %d changes\n", pushed)
	return nil
}

func runSyncPull(cmd *cobra.Command, args []string) error {
	s := newSyncer(cmd)
	pulled, conflicts, err := s.Pull()
	if err != nil {
		return err
	}

	if flagJSON {
		return render.JSON(os.Stdout, map[string]int{"pulled": pulled, "conflicts": conflicts})
	}

	fmt.Fprintf(os.Stderr, "Pulled %d changes", pulled)
	if conflicts > 0 {
		fmt.Fprintf(os.Stderr, " (%d conflicts — local version kept)", conflicts)
	}
	fmt.Fprintln(os.Stderr)
	return nil
}
