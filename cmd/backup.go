package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/secrets"
	"github.com/y0anfa/rhino/internal/store"
)

var backupCmd = &cobra.Command{
	Use:   "backup [output-path]",
	Short: "Create a backup of workflows, config, secrets, and history",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		output := fmt.Sprintf("rhino-backup-%s.tar.gz", time.Now().Format("20060102-150405"))
		if len(args) > 0 {
			output = args[0]
		}

		f, err := os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating backup file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		gw := gzip.NewWriter(f)
		defer gw.Close()
		tw := tar.NewWriter(gw)
		defer tw.Close()

		var count int

		// Backup config
		configPath := config.GetConfigPath()
		if err := addFileToTar(tw, configPath, "config.yaml"); err == nil {
			count++
		}

		// Backup workflows
		wfDir := config.GetString("workflows-dir")
		if wfDir != "" {
			entries, _ := os.ReadDir(wfDir)
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
					if err := addFileToTar(tw, filepath.Join(wfDir, e.Name()), filepath.Join("workflows", e.Name())); err == nil {
						count++
					}
				}
			}
		}

		// Backup secrets
		secretsPath := secrets.DefaultStorePath()
		if err := addFileToTar(tw, secretsPath, "secrets.enc"); err == nil {
			count++
		}

		// Backup history
		dbPath := store.DefaultDBPath()
		if err := addFileToTar(tw, dbPath, "history.db"); err == nil {
			count++
		}

		fmt.Printf("Backup created: %s (%d files)\n", output, count)
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <backup-path>",
	Short: "Restore from a backup archive",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening backup: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		gr, err := gzip.NewReader(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading backup: %v\n", err)
			os.Exit(1)
		}
		defer gr.Close()

		tr := tar.NewReader(gr)
		var count int
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading archive: %v\n", err)
				os.Exit(1)
			}

			var destPath string
			switch {
			case header.Name == "config.yaml":
				destPath = config.GetConfigPath()
			case strings.HasPrefix(header.Name, "workflows/"):
				wfDir := config.GetString("workflows-dir")
				if wfDir == "" {
					wfDir = "workflows"
				}
				destPath = filepath.Join(wfDir, filepath.Base(header.Name))
			case header.Name == "secrets.enc":
				destPath = secrets.DefaultStorePath()
			case header.Name == "history.db":
				destPath = store.DefaultDBPath()
			default:
				continue
			}

			dir := filepath.Dir(destPath)
			os.MkdirAll(dir, 0755)

			out, err := os.Create(destPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", destPath, err)
				continue
			}
			io.Copy(out, tr)
			out.Close()
			count++
			fmt.Printf("  Restored: %s\n", destPath)
		}

		fmt.Printf("Restore complete: %d files\n", count)
	},
}

func addFileToTar(tw *tar.Writer, srcPath, archiveName string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = archiveName

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(tw, f)
	return err
}

func init() {
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}
