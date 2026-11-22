package scripts_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerCompletions(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh is not installed")
	}
	for _, tc := range []struct {
		name                                   string
		legacy, fallback, blocked, badChecksum bool
	}{
		{name: "default directories"},
		{name: "zsh fallback", fallback: true},
		{name: "completion failure preserves binary", blocked: true},
		{name: "older archive", legacy: true},
		{name: "checksum failure", badChecksum: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "test home")
			mockBin := filepath.Join(root, "bin")
			zshDir := filepath.Join(home, "zsh", "site-functions")
			for _, dir := range []string{home, mockBin, zshDir} {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			write := func(path, text string, mode os.FileMode) {
				t.Helper()
				if err := os.WriteFile(path, []byte(text), mode); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{".bashrc", ".zshrc", ".config.fish"} {
				write(filepath.Join(home, name), "# keep my settings\n", 0600)
			}
			archive := filepath.Join(root, "release.tar.gz")
			files := map[string]string{"toggl": "#!/bin/sh\necho fixture\n"}
			if !tc.legacy {
				installer, err := os.ReadFile("install-completions.sh")
				if err != nil {
					t.Fatal(err)
				}
				files["install-completions.sh"] = string(installer)
				for _, shell := range []string{"bash", "zsh", "fish"} {
					files["completions/"+shell] = "# " + shell + " fixture\n"
				}
			}
			var data bytes.Buffer
			gz := gzip.NewWriter(&data)
			tw := tar.NewWriter(gz)
			for name, content := range files {
				if err := tw.WriteHeader(&tar.Header{Name: "toggl-cli-nightly-linux-x64/" + name, Mode: 0755, Size: int64(len(content))}); err != nil {
					t.Fatal(err)
				}
				if _, err := tw.Write([]byte(content)); err != nil {
					t.Fatal(err)
				}
			}
			if err := tw.Close(); err != nil {
				t.Fatal(err)
			}
			if err := gz.Close(); err != nil {
				t.Fatal(err)
			}
			write(archive, data.String(), 0600)
			checksum := fmt.Sprintf("%x", sha256.Sum256(data.Bytes()))
			if tc.badChecksum {
				checksum = strings.Repeat("0", 64)
			}
			write(archive+".sha256", checksum+"\n", 0600)
			write(filepath.Join(mockBin, "cosign"), "#!/bin/sh\ntest \"$1\" = verify-blob-attestation\n", 0700)
			write(filepath.Join(mockBin, "uname"), "#!/bin/sh\ncase $1 in -s) echo Linux;; -m) echo x86_64;; esac\n", 0700)
			write(filepath.Join(mockBin, "zsh"), "#!/bin/sh\nprintf '%s\\n' \"$TEST_ZSH_FPATH\"\n", 0700)
			write(filepath.Join(mockBin, "curl"), `#!/bin/sh
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) output=$2; shift 2;;
    -*) shift;;
    *) url=$1; shift;;
  esac
done
case "$url" in
  */commits/nightly) echo 0123456789012345678901234567890123456789;;
  *.sha256) cp "$TEST_ARCHIVE.sha256" "$output";;
  *) cp "$TEST_ARCHIVE" "$output";;
esac
`, 0700)
			bashDir := filepath.Join(home, "bash completion")
			if tc.blocked {
				write(bashDir, "blocked", 0600)
			}
			fpath := zshDir
			if tc.fallback {
				fpath = filepath.Join(home, "missing", "site-functions")
			}
			cmd := exec.Command(sh, "../install.sh", "--nightly")
			// Strip inherited XDG and completion overrides so tests cannot touch
			// real user directories, including when run on a developer's machine.
			for _, entry := range os.Environ() {
				key, _, _ := strings.Cut(entry, "=")
				switch key {
				case "HOME", "PATH", "XDG_DATA_HOME", "XDG_CONFIG_HOME", "BASH_COMPLETION_USER_DIR", "TMPDIR":
					continue
				}
				cmd.Env = append(cmd.Env, entry)
			}
			cmd.Env = append(cmd.Env, "HOME="+home, "PATH="+mockBin+string(os.PathListSeparator)+os.Getenv("PATH"), "TMPDIR="+root,
				"XDG_DATA_HOME="+filepath.Join(home, "data"), "XDG_CONFIG_HOME="+filepath.Join(home, "config"),
				"BASH_COMPLETION_USER_DIR="+bashDir, "TEST_ZSH_FPATH="+fpath, "TEST_ARCHIVE="+archive)
			output, err := cmd.CombinedOutput()
			binary := filepath.Join(home, ".local", "bin", "toggl")
			if tc.badChecksum {
				if err == nil || !strings.Contains(string(output), "checksum mismatch") {
					t.Fatalf("err=%v output=%s", err, output)
				}
				if _, err := os.Stat(binary); !os.IsNotExist(err) {
					t.Fatal("installed an unverified binary")
				}
				return
			}
			if err != nil {
				t.Fatalf("%v: %s", err, output)
			}
			if b, err := os.ReadFile(binary); err != nil || string(b) != files["toggl"] {
				t.Fatalf("binary=%q err=%v", b, err)
			}
			zshPath := filepath.Join(zshDir, "_toggl")
			if tc.fallback {
				zshPath = filepath.Join(home, "data", "zsh", "site-functions", "_toggl")
				if !strings.Contains(string(output), "No writable default site-functions") {
					t.Fatal(string(output))
				}
			}
			for shell, path := range map[string]string{
				"bash": filepath.Join(bashDir, "completions", "toggl"),
				"fish": filepath.Join(home, "config", "fish", "completions", "toggl.fish"),
				"zsh":  zshPath,
			} {
				b, err := os.ReadFile(path)
				if tc.legacy || (tc.blocked && shell == "bash") {
					if err == nil {
						t.Fatalf("unexpected completion file: %s", path)
					}
				} else if err != nil || string(b) != files["completions/"+shell] {
					t.Fatalf("%s: %q %v", shell, b, err)
				}
			}
			if tc.blocked && !strings.Contains(string(output), "Could not install bash completion") {
				t.Fatal(string(output))
			}
			if !tc.legacy {
				// Manual installation uses the same script; it is repeatable and
				// reports failures even though the main installer is best-effort.
				completionDir := filepath.Join(root, "manual completions")
				if err := os.Mkdir(completionDir, 0700); err != nil {
					t.Fatal(err)
				}
				for _, shell := range []string{"bash", "zsh", "fish"} {
					write(filepath.Join(completionDir, shell), files["completions/"+shell], 0600)
				}
				manual := exec.Command(sh, "install-completions.sh", completionDir)
				manual.Env = cmd.Env
				b, err := manual.CombinedOutput()
				if (err != nil) != tc.blocked {
					t.Fatalf("manual install err=%v output=%s", err, b)
				}
			}
			for _, name := range []string{".bashrc", ".zshrc", ".config.fish"} {
				if b, _ := os.ReadFile(filepath.Join(home, name)); string(b) != "# keep my settings\n" {
					t.Fatalf("modified %s", name)
				}
			}
		})
	}
}
