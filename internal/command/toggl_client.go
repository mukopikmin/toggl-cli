package command

import (
	"context"

	"github.com/mukopikmin/toggl-cli/internal/model"
	"github.com/mukopikmin/toggl-cli/internal/usecase"
)

// TogglClient is the set of operations the CLI uses on an authenticated client.
type TogglClient interface {
	usecase.TimeEntryReader
	usecase.ProjectReader
	Clients(context.Context) ([]model.Client, error)
}
