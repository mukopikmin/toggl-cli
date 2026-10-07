package migrate

import (
	"github.com/mukopikmin/toggl-cli/internal/config"
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	got, err := Convert("# old\nWORKSPACE = 123\nTOKEN=secret\nPROJECT_NAME_9 = Client A\n")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Workspace != "123" || cfg.Token != "secret" || cfg.Projects[9].DisplayName != "Client A" {
		t.Fatalf("%+v\n%s", cfg, got)
	}
	if !strings.Contains(got, "[projects.9]") {
		t.Fatal(got)
	}
}
