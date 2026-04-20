package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Dyuzhovsergey/gophkeeper/internal/cli/commands"
	"github.com/Dyuzhovsergey/gophkeeper/internal/config"
	"github.com/Dyuzhovsergey/gophkeeper/internal/logger"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configArgs, commandArgs := splitClientArgs(os.Args[1:])

	cfg, err := config.LoadClient(configArgs)
	if err != nil {
		return fmt.Errorf("load client config: %w", err)
	}

	log, err := logger.Init(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("init client logger: %w", err)
	}
	defer func() {
		_ = log.Sync()
	}()

	if len(commandArgs) == 0 {
		return fmt.Errorf("client command is required")
	}

	if err := commands.Run(commandArgs, os.Stdout); err != nil {
		return err
	}

	return nil
}

// splitClientArgs разделяет аргументы клиента на глобальные флаги и аргументы команды.
func splitClientArgs(args []string) ([]string, []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--":
			return args[:i], args[i+1:]

		case arg == "-server-address" || arg == "-log-level":
			if i+1 < len(args) {
				i++
				continue
			}
			return args, nil

		case strings.HasPrefix(arg, "-server-address="),
			strings.HasPrefix(arg, "-log-level="):
			continue

		case strings.HasPrefix(arg, "-"):
			continue

		default:
			return args[:i], args[i:]
		}
	}

	return args, nil
}
