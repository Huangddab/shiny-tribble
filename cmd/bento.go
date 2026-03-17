package cmd

import (
	"context"
	"os"

	_ "data-server/internal/bento"

	"github.com/spf13/cobra"
	"github.com/warpstreamlabs/bento/public/service"
)

var (
	bentoCmd = &cobra.Command{
		Use:                "bento",
		Short:              "Bento stream processing service commands",
		Long:               "Commands for managing the Bento stream processing service",
		DisableFlagParsing: true,
		Run:                runBentoCommand,
	}
	// Version version set at compile time.
	Version string
	// DateBuilt date built set at compile time.
	DateBuilt string
	// BinaryName binary name.
	BinaryName string = "bento"
)

func initBentoCommands() {
	// Add bento command to root
	rootCmd.AddCommand(bentoCmd)
}

func runBentoCommand(cmd *cobra.Command, args []string) {
	// Placeholder for actual bento command logic
	os.Args = append([]string{os.Args[0]}, args...)

	// Run the Bento CLI with custom options
	service.RunCLI(
		context.Background(),
		service.CLIOptSetVersion(Version, DateBuilt),
		service.CLIOptSetBinaryName(BinaryName),
		service.CLIOptSetProductName("Bento"),
		service.CLIOptSetDocumentationURL("https://warpstreamlabs.github.io/bento/docs"),
		service.CLIOptSetShowRunCommand(true),
	)
}
