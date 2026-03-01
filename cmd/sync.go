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

var syncInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise sync directory",
	RunE:  runSyncInit,
}

var syncMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate changeset files",
	RunE:  runSyncMigrate,
}

func init() {
	syncCmd.PersistentFlags().String("sync-dir", "",
		"Sync directory path (default: $JOT_SYNC_DIR or XDG data dir)")
	syncInitCmd.Flags().Bool("encrypt", false, "Enable encryption for sync")
	syncMigrateCmd.Flags().Bool("encrypt", false, "Encrypt existing plain changeset files")
	syncCmd.AddCommand(syncStatusCmd, syncPushCmd, syncPullCmd, syncInitCmd, syncMigrateCmd)
	rootCmd.AddCommand(syncCmd)
}

func newSyncer(cmd *cobra.Command) (*sync.Syncer, error) {
	syncDir, _ := cmd.Flags().GetString("sync-dir")
	if syncDir == "" {
		syncDir = cfg.SyncDir
	}

	encrypted, _ := db.GetSyncMeta("encrypt")
	if encrypted == "true" {
		passphrase, err := sync.LoadIdentity(cfg.IdentityPath)
		if err != nil {
			return nil, fmt.Errorf("loading sync identity: %w", err)
		}
		return sync.NewEncrypted(db, syncDir, passphrase), nil
	}

	return sync.New(db, syncDir), nil
}

func runSync(cmd *cobra.Command, args []string) error {
	s, err := newSyncer(cmd)
	if err != nil {
		return err
	}
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
	s, err := newSyncer(cmd)
	if err != nil {
		return err
	}
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

	encrypted, _ := db.GetSyncMeta("encrypt")
	if encrypted == "true" {
		fmt.Fprintln(os.Stderr, "Encryption: enabled")
	}

	return nil
}

func runSyncPush(cmd *cobra.Command, args []string) error {
	s, err := newSyncer(cmd)
	if err != nil {
		return err
	}
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
	s, err := newSyncer(cmd)
	if err != nil {
		return err
	}
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

func runSyncInit(cmd *cobra.Command, args []string) error {
	encrypt, _ := cmd.Flags().GetBool("encrypt")

	syncDir, _ := cmd.Flags().GetString("sync-dir")
	if syncDir == "" {
		syncDir = cfg.SyncDir
	}

	// Ensure the sync directory exists.
	s := sync.New(db, syncDir)
	if _, err := s.Status(); err != nil {
		return err
	}

	if encrypt {
		if err := sync.GenerateIdentity(cfg.IdentityPath); err != nil {
			return fmt.Errorf("generating identity: %w", err)
		}
		if err := db.SetSyncMeta("encrypt", "true"); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Encryption enabled. Identity saved to %s\n", cfg.IdentityPath)
		fmt.Fprintln(os.Stderr, "Keep this file safe — it is required to decrypt your synced notes.")
	} else {
		fmt.Fprintln(os.Stderr, "Sync initialised (unencrypted).")
	}

	return nil
}

func runSyncMigrate(cmd *cobra.Command, args []string) error {
	encrypt, _ := cmd.Flags().GetBool("encrypt")
	if !encrypt {
		return fmt.Errorf("specify --encrypt to migrate existing changesets to encrypted format")
	}

	s, err := newSyncer(cmd)
	if err != nil {
		return err
	}

	if !s.Encrypted() {
		return fmt.Errorf("encryption is not enabled — run 'jot sync init --encrypt' first")
	}

	migrated, err := s.MigrateEncrypt()
	if err != nil {
		return err
	}

	if flagJSON {
		return render.JSON(os.Stdout, map[string]int{"migrated": migrated})
	}

	fmt.Fprintf(os.Stderr, "Migrated %d changeset files to encrypted format\n", migrated)
	return nil
}
