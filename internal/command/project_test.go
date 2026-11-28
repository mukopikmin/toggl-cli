package command

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestProjectListAndSync(t *testing.T) {
	client := &fakeClient{projects: []model.Project{{ID: 2, Name: "Hidden", Active: true}, {ID: 1, Name: "Client", Active: true}}}
	app, out := configuredApp(t, client, "")
	appendProjectConfig(t, "\n[projects.1]\ndisplay_name = \"Shown\"\n[projects.2]\nhidden = true\n")
	if err := app.Run(context.Background(), []string{"project", "list", "--format", "tsv"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "name\tid\talias\tdisplayName\thidden\tactive\tdisplayOrder\nClient\t1\tShown\tShown\tfalse\ttrue\t\n" {
		t.Fatal(out.String())
	}
	out.Reset()
	client.projects = append(client.projects,
		model.Project{ID: 3, Name: "New", Active: true},
		model.Project{ID: 4, Name: "Archived"},
	)
	if err := app.Run(context.Background(), []string{"project", "sync"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(config.Path(os.Getenv("HOME")))
	if !strings.Contains(string(b), "[projects.\"3\"]") || strings.Contains(string(b), "[projects.\"4\"]") {
		t.Fatal(string(b))
	}
}
