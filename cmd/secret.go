package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/secrets"
	"golang.org/x/term"
)

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage encrypted secrets",
	Long:  `Manage secrets stored in an encrypted local store. Set RHINO_SECRET_KEY or provide master key when prompted.`,
}

var secretSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Set a secret value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		store, err := secrets.NewStore(secrets.DefaultStorePath(), "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print("Enter secret value: ")
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}

		if err := store.Set(args[0], string(value)); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting secret: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Secret '%s' saved.\n", args[0])
	},
}

var secretGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a secret value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		store, err := secrets.NewStore(secrets.DefaultStorePath(), "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		val, ok := store.Get(args[0])
		if !ok {
			fmt.Fprintf(os.Stderr, "Secret '%s' not found.\n", args[0])
			os.Exit(1)
		}
		fmt.Println(val)
	},
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secret keys",
	Run: func(cmd *cobra.Command, args []string) {
		store, err := secrets.NewStore(secrets.DefaultStorePath(), "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		keys := store.List()
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Println(k)
		}
	},
}

var secretDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a secret",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		store, err := secrets.NewStore(secrets.DefaultStorePath(), "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := store.Delete(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting secret: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Secret '%s' deleted.\n", args[0])
	},
}

func init() {
	secretCmd.AddCommand(secretSetCmd)
	secretCmd.AddCommand(secretGetCmd)
	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretDeleteCmd)
	rootCmd.AddCommand(secretCmd)
}
