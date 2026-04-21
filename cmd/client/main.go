package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dyuzhovsergey/gophkeeper/internal/cli"
	"github.com/Dyuzhovsergey/gophkeeper/internal/clientapi"
	"github.com/Dyuzhovsergey/gophkeeper/internal/config"
	"github.com/Dyuzhovsergey/gophkeeper/internal/logger"
	"github.com/Dyuzhovsergey/gophkeeper/internal/storage/local"
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

	apiClient, err := clientapi.New(cfg.ServerAddress, nil)
	if err != nil {
		return fmt.Errorf("init client api: %w", err)
	}

	sessionPath, err := local.DefaultSessionPath()
	if err != nil {
		return fmt.Errorf("resolve local session path: %w", err)
	}

	store, err := local.NewFileStore(sessionPath)
	if err != nil {
		return fmt.Errorf("init local session store: %w", err)
	}

	model := cli.NewModel(apiClient, store, commandArgs)

	finalModel, err := tea.NewProgram(model).Run()
	if err != nil {
		return fmt.Errorf("run bubble tea program: %w", err)
	}

	resultModel, ok := finalModel.(*cli.Model)
	if !ok {
		return fmt.Errorf("unexpected bubble tea final model type: %T", finalModel)
	}

	if resultModel.Err() != nil {
		return resultModel.Err()
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
			// Неизвестный флаг считаем глобальным.
			// Ошибку вернёт config.LoadClient.
			continue

		default:
			return args[:i], args[i:]
		}
	}

	return args, nil
}
