package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Kashuab/claimenv/internal/lockstore"
	"github.com/spf13/cobra"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status <pool>",
	Short: "Show the status of all slots in a pool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		poolName := args[0]

		statuses, err := eng.Status(cmd.Context(), poolName)
		if err != nil {
			return err
		}

		if statusJSON {
			return printStatusJSON(statuses)
		}

		return printStatusTable(statuses)
	},
}

func printStatusJSON(statuses []lockstore.SlotStatus) error {
	data, err := json.MarshalIndent(statuses, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printStatusTable(statuses []lockstore.SlotStatus) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLOT\tSTATUS\tHOLDER\tEXPIRES")

	for _, s := range statuses {
		status := "free"
		holder := "-"
		expires := "-"

		if s.Claimed && s.Claim != nil {
			status = "claimed"
			holder = s.Claim.Holder
			expires = s.Claim.ExpiresAt.Format("2006-01-02 15:04:05")
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.SlotIndex, status, holder, expires)
	}

	return w.Flush()
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(statusCmd)
}
