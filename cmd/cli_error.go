package cmd

import (
	"errors"
	"fmt"
)

type CLIError struct {
	Message  string
	Err      error
	ExitCode int
}

func (e *CLIError) Error() string {
	switch {
	case e == nil:
		return ""
	case e.Message != "" && e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	case e.Message != "":
		return e.Message
	case e.Err != nil:
		return e.Err.Error()
	default:
		return "command failed"
	}
}

func (e *CLIError) Unwrap() error { return e.Err }

func commandError(message string, err error) error {
	return &CLIError{Message: message, Err: err, ExitCode: 1}
}

func usageError(message string) error {
	return &CLIError{Message: message, ExitCode: 2}
}

func formatCLIError(err error) string {
	if err == nil {
		return ""
	}
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return "❌ " + cliErr.Error()
	}
	return err.Error()
}

func cliExitCode(err error) int {
	var cliErr *CLIError
	if errors.As(err, &cliErr) && cliErr.ExitCode != 0 {
		return cliErr.ExitCode
	}
	return 1
}
