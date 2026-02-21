package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Kashuab/claimenv/internal/lease"
	"github.com/spf13/cobra"
)

var envFormat string

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Dump all env vars from the claimed slot",
	Long:  `Outputs all environment variables. Use eval $(claimenv env) to source them.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		lf, err := lease.Load(eng.LeaseFile)
		if err != nil {
			return err
		}

		all, err := eng.ReadAll(cmd.Context(), lf)
		if err != nil {
			return err
		}

		// Sort keys for deterministic output
		keys := make([]string, 0, len(all))
		for k := range all {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		switch envFormat {
		case "json":
			data, err := json.MarshalIndent(all, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		case "dotenv":
			for _, k := range keys {
				fmt.Printf("%s=%s\n", k, all[k])
			}
		default: // "export"
			for _, k := range keys {
				fmt.Printf("export %s=%q\n", k, all[k])
			}
		}

		return nil
	},
}

func init() {
	envCmd.Flags().StringVar(&envFormat, "format", "export", "output format: export, dotenv, json")
	rootCmd.AddCommand(envCmd)
}
