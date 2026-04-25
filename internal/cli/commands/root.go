// Package commands предоставляет базовые CLI-команды клиента.
package commands

import (
	"fmt"
	"io"
)

// Run выполняет CLI-команду клиента.
func Run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("client command is required")
	}

	switch args[0] {
	case "version":
		return runVersion(stdout, args[1:])
	default:
		return fmt.Errorf("unknown client command: %s", args[0])
	}
}