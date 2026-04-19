// Package buildinfo предоставляет информацию о сборке приложения.
package buildinfo

import "fmt"

// Version содержит версию приложения.
var Version = "N/A"

// Date содержит дату сборки приложения.
var Date = "N/A"

// Commit содержит идентификатор коммита, из которого собрано приложение.
var Commit = "N/A"

// Info описывает информацию о сборке приложения.
type Info struct {
	// Version — версия приложения.
	Version string

	// Date — дата сборки приложения.
	Date string

	// Commit — идентификатор коммита.
	Commit string
}

// Current возвращает текущую информацию о сборке.
func Current() Info {
	return Info{
		Version: Version,
		Date:    Date,
		Commit:  Commit,
	}
}

// String возвращает человекочитаемое представление информации о сборке.
func (i Info) String() string {
	return fmt.Sprintf(
		"version: %s\ndate: %s\ncommit: %s",
		i.Version,
		i.Date,
		i.Commit,
	)
}
