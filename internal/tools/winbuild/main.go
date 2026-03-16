package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jonlabelle/nssm-redux/internal/versioninfo"
)

const versionVariable = "github.com/jonlabelle/nssm-redux/internal/cli.Version"

func main() {
	var (
		source          string
		output          string
		arch            string
		version         string
		versionInfoPath string
	)

	flag.StringVar(&source, "source", "./cmd/nssmr", "Go package path for the Windows build")
	flag.StringVar(&output, "out", "", "output executable path")
	flag.StringVar(&arch, "arch", "", "target Windows architecture")
	flag.StringVar(&version, "version", "dev", "application version to embed")
	flag.StringVar(&versionInfoPath, "versioninfo", "", "path to the VERSIONINFO JSON config")
	flag.Parse()

	if output == "" {
		fail("missing required -out argument")
	}
	if arch == "" {
		fail("missing required -arch argument")
	}

	machine, err := versioninfo.MachineForGOARCH(arch)
	if err != nil {
		fail(err.Error())
	}

	cfg, err := versioninfo.LoadConfig(versionInfoPath)
	if err != nil {
		fail(err.Error())
	}
	cfg.ApplyDefaults(version, filepath.Base(output))

	resourceObject, err := versioninfo.Generate(cfg, machine)
	if err != nil {
		fail(err.Error())
	}

	sourceDir, err := resolvePackageDir(source)
	if err != nil {
		fail(err.Error())
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		failf("create output directory: %v", err)
	}

	resourcePath := filepath.Join(sourceDir, fmt.Sprintf("zz_versioninfo_windows_%s.syso", arch))
	if err := os.WriteFile(resourcePath, resourceObject, 0o644); err != nil {
		failf("write generated version resource %q: %v", resourcePath, err)
	}
	defer func() {
		_ = os.Remove(resourcePath)
	}()

	ldflags := fmt.Sprintf("-s -w -X %s=%s", versionVariable, version)
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", output, source)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=windows",
		"GOARCH="+arch,
	)

	if err := cmd.Run(); err != nil {
		failf("go build failed: %v", err)
	}
}

func resolvePackageDir(source string) (string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", source)
	output, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exitError, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(exitError.Stderr))
		}
		if stderr != "" {
			return "", fmt.Errorf("resolve package directory for %q: %w\n%s", source, err, stderr)
		}
		return "", fmt.Errorf("resolve package directory for %q: %w", source, err)
	}

	dir := strings.TrimSpace(string(output))
	if dir == "" {
		return "", fmt.Errorf("resolve package directory for %q: empty result", source)
	}

	return dir, nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func failf(format string, args ...any) {
	fail(fmt.Sprintf(format, args...))
}
