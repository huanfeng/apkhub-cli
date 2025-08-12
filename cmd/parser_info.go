package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/huanfeng/apkhub-cli/pkg/apk"
	"github.com/spf13/cobra"
)

var parserInfoCmd = &cobra.Command{
	Use:   "parser-info",
	Short: "Show information about available APK parsers",
	Long:  `Display information about all available APK parsers and their capabilities.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create parser to get info
		parser := apk.NewParser(".")
		parsers := parser.GetParserInfo()

		if len(parsers) == 0 {
			fmt.Println("No parsers available")
			return nil
		}

		// Display parser information
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tAVAILABLE\tPRIORITY\tCAPABILITIES")
		fmt.Fprintln(w, "----\t-------\t---------\t--------\t------------")

		for _, info := range parsers {
			available := "No"
			if info.Available {
				available = "Yes"
			}

			capabilities := ""
			if len(info.Capabilities) > 0 {
				capabilities = fmt.Sprintf("%v", info.Capabilities)
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				info.Name,
				info.Version,
				available,
				info.Priority,
				capabilities,
			)
		}

		w.Flush()

		return nil
	},
}

func init() {
	repoCmd.AddCommand(parserInfoCmd)
}