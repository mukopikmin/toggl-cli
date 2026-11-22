package command

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// Public Sigstore trust root from sigstore/root-signing at 25b222a48cda4aa88e1af85605bdd193f5619a60.
//
//go:embed sigstore-root.json
var sigstoreRoot []byte

func verifyReleaseOrigin(ctx context.Context, client *http.Client, plan updatePlan, archive []byte) error {
	tag, workflow := "v"+plan.TargetVersion, "release"
	ref := "refs/tags/" + tag
	if plan.Channel == "nightly" {
		tag, workflow, ref = "nightly", "nightly", "refs/heads/main"
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if err := getJSON(ctx, client, updateAPI+"/commits/"+tag, &commit); err != nil {
		return err
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(commit.SHA) {
		return errors.New("invalid release commit")
	}
	dir, err := os.MkdirTemp("", "toggl-provenance-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	archivePath := filepath.Join(dir, plan.Archive)
	if err = os.WriteFile(archivePath, archive, 0600); err != nil {
		return err
	}
	bundleURL, err := url.JoinPath(plan.DownloadURL, "..", "provenance.json")
	if err != nil {
		return err
	}
	if tag == "v0.5.2" {
		bundleURL = "https://github.com/mukopikmin/toggl-cli/attestations/58847128/download"
	}
	bundle, err := download(ctx, client, bundleURL)
	if err != nil {
		return fmt.Errorf("download build provenance: %w", err)
	}
	bundlePath := filepath.Join(dir, "provenance.json")
	if err = os.WriteFile(bundlePath, bundle, 0600); err != nil {
		return err
	}
	verifier, err := exec.LookPath("cosign")
	if errors.Is(err, exec.ErrNotFound) {
		verifier, err = downloadVerifier(ctx, client, dir, plan.Target)
	}
	if err != nil {
		return err
	}
	rootPath := filepath.Join(dir, "trusted_root.json")
	if err = os.WriteFile(rootPath, sigstoreRoot, 0600); err != nil {
		return err
	}
	identity := "https://github.com/mukopikmin/toggl-cli/.github/workflows/" + workflow + ".yml@" + ref
	output, err := exec.CommandContext(ctx, verifier, "verify-blob-attestation", "--trusted-root", rootPath, "--bundle", bundlePath, "--type", "slsaprovenance1", "--certificate-identity", identity, "--certificate-oidc-issuer", "https://token.actions.githubusercontent.com", "--certificate-github-workflow-sha", commit.SHA, archivePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("release provenance verification failed: %w: %s", err, output)
	}
	return nil
}

func downloadVerifier(ctx context.Context, client *http.Client, dir, target string) (string, error) {
	var asset, checksum string
	switch target {
	case "linux-x64":
		asset, checksum = "cosign-linux-amd64", "c956e5dfcac53d52bcf058360d579472f0c1d2d9b69f55209e256fe7783f4c74"
	case "darwin-arm64":
		asset, checksum = "cosign-darwin-arm64", "5fadd012ae6381a6a29ff86a7d39aa873878852f1073fc90b15995961ecfb084"
	case "windows-x64":
		asset, checksum = "cosign-windows-amd64.exe", "9b85a88ebff2d9dd30ff4984a6f61f2cedc232dd87d81fa7f2ff3c0ed96c241c"
	default:
		return "", fmt.Errorf("unsupported verifier target %q", target)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://github.com/sigstore/cosign/releases/download/v3.0.6/"+asset, nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download verifier: %s", res.Status)
	}
	path := filepath.Join(dir, asset)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	digest := sha256.New()
	_, err = io.Copy(io.MultiWriter(f, digest), io.LimitReader(res.Body, 256<<20))
	closeErr := f.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if hex.EncodeToString(digest.Sum(nil)) != checksum {
		return "", errors.New("verifier checksum mismatch")
	}
	return path, os.Chmod(path, 0700)
}
