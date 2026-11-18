package command

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Only exact regular-file entries from the checksum-verified release are used.
// Never unpack arbitrary archive paths or links into the user's filesystem.
func installUpdateCompletions(ctx context.Context, archive []byte, name string, out, errOut io.Writer) error {
	root := strings.TrimSuffix(name, ".tar.gz") + "/"
	wanted := map[string]bool{"install-completions.sh": true, "completions/bash": true, "completions/zsh": true, "completions/fish": true}
	files := map[string][]byte{}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(h.Name, root)
		if h.Name == rel || !wanted[rel] {
			continue
		}
		if h.Typeflag != tar.TypeReg || h.Size > 1024*1024 || files[rel] != nil {
			return fmt.Errorf("invalid completion archive entry: %s", h.Name)
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		files[rel] = b
	}
	if _, ok := files["install-completions.sh"]; !ok {
		return nil
	} // Legacy release.
	if len(files) != len(wanted) {
		return fmt.Errorf("release archive has incomplete completion files")
	}
	dir, err := os.MkdirTemp("", "toggl-completions-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := os.Mkdir(filepath.Join(dir, "completions"), 0700); err != nil {
		return err
	}
	for rel, b := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), b, 0600); err != nil {
			return err
		}
	}
	cmd := exec.CommandContext(ctx, "sh", filepath.Join(dir, "install-completions.sh"))
	cmd.Stdout, cmd.Stderr = out, errOut
	return cmd.Run()
}
