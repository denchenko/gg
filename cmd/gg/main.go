package main

import (
	"log"

	"github.com/denchenko/gg/internal/adapters"
	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core"
	do "github.com/samber/do/v2"
	"github.com/spf13/cobra"
)

func main() {
	injector := do.New(
		config.Package,
		core.Package,
		adapters.SecondaryPackage,
		adapters.PrimaryPackage,
	)

	cmd, err := do.Invoke[*cobra.Command](injector)
	if err != nil {
		log.Fatalf("failed to create CLI command: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
