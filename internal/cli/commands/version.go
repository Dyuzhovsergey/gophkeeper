package commands

import (
	"fmt"
	"io"

	"github.com/Dyuzhovsergey/gophkeeper/internal/buildinfo"
)

// runVersion выводит информацию о сборке клиента.
func runVersion(stdout io.Writer, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("version command does not accept arguments")
	}

	_, err := fmt.Fprintln(stdout, buildinfo.Current().String())
	return err
}
