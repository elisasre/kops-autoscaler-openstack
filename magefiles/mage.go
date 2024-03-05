//go:build mage

package main

import (
	"os"

	"github.com/magefile/mage/mg"

	//mage:import
	_ "github.com/elisasre/mageutil/git/target"
	//mage:import
	_ "github.com/elisasre/mageutil/golangcilint/target"
	//mage:import
	_ "github.com/elisasre/mageutil/govulncheck/target"
	//mage:import
	_ "github.com/elisasre/mageutil/golicenses/target"
	//mage:import
	docker "github.com/elisasre/mageutil/docker/target"
	//mage:import
	golang "github.com/elisasre/mageutil/golang/target"
)

// Configure imported targets
func init() {
	os.Setenv(mg.VerboseEnv, "1")
	os.Setenv("CGO_ENABLED", "0")

	golang.BuildTarget = "./cmd/kops-autoscaler-openstack"
	golang.RunArgs = []string{"--log-level=4", "--sleep=10"}
	docker.ImageName = "europe-north1-docker.pkg.dev/sose-sre-5737/sre-public/kops-autoscaler-openstack"
	docker.ProjectUrl = "https://github.com/elisasre/kops-autoscaler-openstack"
}
