package command

import (
	"context"
	"strings"
	"testing"
)

func TestConfigFormatsHideToken(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "json", "table"} {
		t.Run(format, func(t *testing.T) {
			app, out := configuredApp(t, &fakeClient{}, "")
			if err := app.Run(context.Background(), []string{"config", "--format", format}); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "secret") || !strings.Contains(out.String(), "WORKSPACE") {
				t.Fatal(out.String())
			}
		})
	}
}
