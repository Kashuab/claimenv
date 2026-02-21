package cmd

import (
	"fmt"

	"github.com/Kashuab/claimenv/internal/lease"
	"github.com/spf13/cobra"
)

var readFormat string

var readCmd = &cobra.Command{
	Use:   "read <KEY>",
	Short: "Read a single env var from the claimed slot",
	Long:  `Reads a value or secret name. Use --format=name to get the GCP Secret Manager secret name instead of the value.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		lf, err := lease.Load(eng.LeaseFile)
		if err != nil {
			return err
		}

		switch readFormat {
		case "name":
			name, err := eng.SecretName(lf, key)
			if err != nil {
				return err
			}
			fmt.Print(name)
		default: // "value"
			val, err := eng.ReadKey(cmd.Context(), lf, key)
			if err != nil {
				return err
			}
			fmt.Print(val)
		}

		return nil
	},
}

func init() {
	readCmd.Flags().StringVar(&readFormat, "format", "value", "output format: value, name")
	rootCmd.AddCommand(readCmd)
}
