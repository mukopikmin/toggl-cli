//go:build !windows

package command

func enableTableTerminal(uintptr) (func() error, error) {
	return func() error { return nil }, nil
}
