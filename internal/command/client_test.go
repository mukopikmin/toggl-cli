package command

import (
	"bytes"
	"context"
	stdcsv "encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestClientListFormats(t *testing.T) {
	clients := []model.Client{{ID: 2, Name: "Beta"}, {ID: 1, Name: "取引先 A"}}
	for _, format := range []string{"csv", "tsv", "json", "table"} {
		t.Run(format, func(t *testing.T) {
			client := &fakeClient{clients: clients}
			app, out := configuredApp(t, client, "")
			path := config.Path(os.Getenv("HOME"))
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := app.Run(context.Background(), []string{"client", "list", "--format", format}); err != nil {
				t.Fatal(err)
			}
			switch format {
			case "json":
				var got []model.Client
				if err := json.Unmarshal(out.Bytes(), &got); err != nil || !reflect.DeepEqual(got, clients) {
					t.Fatalf("output=%q error=%v", out.String(), err)
				}
			case "table":
				if !strings.Contains(out.String(), "│ Client") || !strings.Contains(out.String(), "Beta") || !strings.Contains(out.String(), "取引先 A") {
					t.Fatal(out.String())
				}
			default:
				want := "id\tname\n2\tBeta\n1\t取引先 A\n"
				if format == "csv" {
					want = strings.ReplaceAll(want, "\t", ",")
				}
				if out.String() != want {
					t.Fatalf("%q", out.String())
				}
			}
			if client.clientCalls != 1 || client.projectCalls != 0 || client.entryCalls != 0 {
				t.Fatalf("unexpected API calls: %+v", client)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("configuration changed: %v", err)
			}
		})
	}
}

func TestClientListDefaultAndEmptyOutput(t *testing.T) {
	tests := []struct {
		name    string
		clients []model.Client
		options []string
		want    string
	}{
		{"default", []model.Client{{ID: 1, Name: "Client A"}}, nil, "id\tname\n1\tClient A\n"},
		{"empty default", nil, nil, "id\tname\n"},
		{"CSV", []model.Client{{ID: 1, Name: "Client A"}}, []string{"-fcsv"}, "id,name\n1,Client A\n"},
		{"empty CSV", nil, []string{"--format", "csv"}, "id,name\n"},
		{"nil JSON", nil, []string{"-fjson"}, "[]\n"},
		{"empty JSON", []model.Client{}, []string{"--format=json"}, "[]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out := configuredApp(t, &fakeClient{clients: tt.clients}, "")
			if err := app.Run(context.Background(), append([]string{"client", "list"}, tt.options...)); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("got %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestClientUsageBeforeConfiguration(t *testing.T) {
	tests := [][]string{
		{"client"}, {"client", "get"}, {"client", "list", "extra"},
		{"client", "list", "--format", "xml"}, {"client", "list", "--format"},
		{"client", "list", "--separator", ""}, {"client", "list", "--separator"},
		{"client", "list", "--format", "table", "--separator", ","},
		{"client", "list", "--clipboard"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			client := &fakeClient{}
			app := App{Client: client, Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), args)
			var usageError UsageError
			if !errors.As(err, &usageError) || client.clientCalls != 0 {
				t.Fatalf("expected usage error without API access, got %v", err)
			}
		})
	}
}

func TestClientListDelimitedOutputEscapesNames(t *testing.T) {
	clients := []model.Client{
		{ID: 1, Name: "Client, A"}, {ID: 2, Name: "Client\tB"},
		{ID: 3, Name: "Client \"C\""}, {ID: 4, Name: "Client\nD"},
	}
	want := [][]string{{"id", "name"}, {"1", clients[0].Name}, {"2", clients[1].Name}, {"3", clients[2].Name}, {"4", clients[3].Name}}
	for _, format := range []string{"csv", "tsv"} {
		t.Run(format, func(t *testing.T) {
			app, out := configuredApp(t, &fakeClient{clients: clients}, "")
			if err := app.Run(context.Background(), []string{"client", "list", "--format", format}); err != nil {
				t.Fatal(err)
			}
			reader := stdcsv.NewReader(out)
			if format == "tsv" {
				reader.Comma = '\t'
			}
			got, err := reader.ReadAll()
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("got %q, %v; want %q", got, err, want)
			}
		})
	}
}

func TestClientListErrors(t *testing.T) {
	t.Run("API", func(t *testing.T) {
		want := errors.New("fetch clients failed")
		app, out := configuredApp(t, &fakeClient{err: want}, "")
		if err := app.Run(context.Background(), []string{"client", "list"}); !errors.Is(err, want) || out.Len() != 0 {
			t.Fatalf("error=%v output=%q", err, out.String())
		}
	})
	t.Run("missing configuration", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		client := &fakeClient{}
		app := App{Client: client, Out: &bytes.Buffer{}}
		var missing config.FileNotFoundError
		if err := app.Run(context.Background(), []string{"client", "list"}); !errors.As(err, &missing) || client.clientCalls != 0 {
			t.Fatalf("expected configuration error without API access, got %v", err)
		}
	})
}
