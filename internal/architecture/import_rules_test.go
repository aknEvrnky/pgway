package architecture_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

type pkg struct {
	ImportPath string   `json:"ImportPath"`
	Imports    []string `json:"Imports"`
}

func TestImportRules(t *testing.T) {
	cmd := exec.Command("go", "list", "-json", "github.com/aknEvrnky/pgway/internal/...", "github.com/aknEvrnky/pgway/cmd/...")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}

	dec := json.NewDecoder(strings.NewReader(string(out)))
	var pkgs []pkg
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			t.Fatal(err)
		}
		pkgs = append(pkgs, p)
	}

	const (
		dataplanePrefix    = "github.com/aknEvrnky/pgway/internal/application/dataplane"
		controlplanePrefix = "github.com/aknEvrnky/pgway/internal/application/controlplane"
		portsPkg           = "github.com/aknEvrnky/pgway/internal/ports"
		domainPkg          = "github.com/aknEvrnky/pgway/internal/application/core/domain"
		applicationPrefix  = "github.com/aknEvrnky/pgway/internal/application/"
		cliAdapterPrefix   = "github.com/aknEvrnky/pgway/internal/adapters/cli"
		cmdPgctlPkg        = "github.com/aknEvrnky/pgway/cmd/pgctl"
		platformConfigPref = "github.com/aknEvrnky/pgway/internal/platform/config"
	)

	for _, p := range pkgs {
		for _, imp := range p.Imports {
			if strings.HasPrefix(p.ImportPath, dataplanePrefix) {
				if strings.HasPrefix(imp, controlplanePrefix) {
					t.Errorf("%s must not import %s", p.ImportPath, imp)
				}
			}
			if p.ImportPath == portsPkg || strings.HasPrefix(p.ImportPath, portsPkg+"/") {
				if strings.HasPrefix(imp, applicationPrefix) && imp != domainPkg {
					t.Errorf("ports must not import %s (only core/domain allowed)", imp)
				}
			}
			// pgctl is config-file free: address comes from --host/--port and
			// the token from --token-path. A config import here would silently
			// reintroduce the config.toml dependency.
			if strings.HasPrefix(p.ImportPath, cliAdapterPrefix) || p.ImportPath == cmdPgctlPkg {
				if strings.HasPrefix(imp, platformConfigPref) {
					t.Errorf("%s must not import %s (pgctl is config-file free)", p.ImportPath, imp)
				}
			}
		}
	}
}
