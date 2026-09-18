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
	cmd := exec.Command("go", "list", "-json", "github.com/aknEvrnky/pgway/internal/...")
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
		dataplanePrefix   = "github.com/aknEvrnky/pgway/internal/application/dataplane"
		controlplanePkg   = "github.com/aknEvrnky/pgway/internal/application/controlplane"
		authPkg           = "github.com/aknEvrnky/pgway/internal/application/auth"
		agentPkg          = "github.com/aknEvrnky/pgway/internal/application/agent"
		portsPkg          = "github.com/aknEvrnky/pgway/internal/ports"
		domainPkg         = "github.com/aknEvrnky/pgway/internal/application/core/domain"
		applicationPrefix = "github.com/aknEvrnky/pgway/internal/application/"
	)

	for _, p := range pkgs {
		for _, imp := range p.Imports {
			if strings.HasPrefix(p.ImportPath, dataplanePrefix) {
				if imp == controlplanePkg || imp == authPkg || imp == agentPkg {
					t.Errorf("%s must not import %s", p.ImportPath, imp)
				}
			}
			if p.ImportPath == portsPkg || strings.HasPrefix(p.ImportPath, portsPkg+"/") {
				if strings.HasPrefix(imp, applicationPrefix) && imp != domainPkg {
					t.Errorf("ports must not import %s (only core/domain allowed)", imp)
				}
			}
		}
	}
}
