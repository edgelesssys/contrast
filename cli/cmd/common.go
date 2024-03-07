package cmd

import (
	_ "embed"
	"os"
	"path/filepath"
)

const (
	coordHashFilename      = "coordinator-policy.sha256"
	coordRootPEMFilename   = "coordinator-root.pem"
	coordIntermPEMFilename = "mesh-root.pem"
	workloadOwnerPEM       = "workload-owner.pem"
	manifestFilename       = "manifest.json"
	settingsFilename       = "settings.json"
	rulesFilename          = "rules.rego"
	verifyDir              = "./verify"
	cacheDirEnv            = "CONTRAST_CACHE_DIR"
)

var (
	//go:embed assets/genpolicy
	genpolicyBin []byte
	//go:embed assets/genpolicy-settings.json
	defaultGenpolicySettings []byte
	//go:embed assets/genpolicy-rules.rego
	defaultRules []byte
)

func cachedir(subdir string) (string, error) {
	dir := os.Getenv(cacheDirEnv)
	if dir == "" {
		cachedir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(cachedir, "contrast")
	}
	return filepath.Join(dir, subdir), nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
