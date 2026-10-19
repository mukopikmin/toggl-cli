package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckThirdPartyNotices(t *testing.T) {
	script, err := filepath.Abs("check_third_party_notices.sh")
	if err != nil {
		t.Fatal(err)
	}
	const covered = "# Notices\n\n## example.com/module\n\n> Copyright Example.\n> Permission to use\n> and distribute.\n"
	const license = "Copyright Example.\n\nPermission to use and distribute.\n"
	tests := []struct {
		name, notices, module, license, licenseName, goFailure, wantError string
	}{
		{name: "wrapped license", notices: covered, module: "example.com/module", license: license, licenseName: "LICENSE"},
		{name: "updated version and COPYING", notices: covered, module: "example.com/module", license: license, licenseName: "COPYING"},
		{name: "missing notices", module: "example.com/module", license: license, wantError: "Missing or empty"},
		{name: "missing module", notices: covered, module: "example.com/new", license: license, licenseName: "LICENSE", wantError: "Missing third-party notice"},
		{name: "missing license", notices: covered, module: "example.com/module", wantError: "Cannot find a license"},
		{name: "blank license", notices: covered, module: "example.com/module", license: " \n", licenseName: "LICENSE", wantError: "Empty license"},
		{name: "changed license", notices: covered, module: "example.com/module", license: "Copyright Another Author.\n", licenseName: "LICENSE", wantError: "License text"},
		{name: "download failure", notices: covered, goFailure: "mod", wantError: "go failed"},
		{name: "list failure", notices: covered, goFailure: "list", wantError: "go failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			moduleDir := filepath.Join(dir, "module@v2.0.0")
			binDir := filepath.Join(dir, "bin")
			for _, path := range []string{moduleDir, binDir} {
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			}
			write := func(path, data string, mode os.FileMode) {
				t.Helper()
				if err := os.WriteFile(path, []byte(data), mode); err != nil {
					t.Fatal(err)
				}
			}
			if tt.notices != "" {
				write(filepath.Join(dir, "THIRD_PARTY_NOTICES.md"), tt.notices, 0644)
			}
			if tt.licenseName != "" {
				write(filepath.Join(moduleDir, tt.licenseName), tt.license, 0644)
			}
			write(filepath.Join(binDir, "go"), `#!/bin/sh
if [ "$1" = "$NOTICE_TEST_FAILURE" ]; then
  echo 'go failed' >&2
  exit 1
fi
if [ "$1" = list ]; then
  printf '\n%s %s\n' "$NOTICE_TEST_MODULE" "$NOTICE_TEST_DIR"
fi
`, 0755)
			cmd := exec.Command("sh", script)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(),
				"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"NOTICE_TEST_MODULE="+tt.module,
				"NOTICE_TEST_DIR="+moduleDir,
				"NOTICE_TEST_FAILURE="+tt.goFailure,
			)
			output, err := cmd.CombinedOutput()
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("check failed: %v\n%s", err, output)
				}
			} else if err == nil || !strings.Contains(string(output), tt.wantError) {
				t.Fatalf("expected %q, got %v\n%s", tt.wantError, err, output)
			}
		})
	}
}
