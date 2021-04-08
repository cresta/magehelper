// +build mage

package main

import (
	// mage:import go
	_ "github.com/cresta/magehelper/gobuild"
	// mage:import docker
	_ "github.com/cresta/magehelper/docker"
	// mage:import shell
	_ "github.com/cresta/magehelper/shellcheck"
	// mage:import ecr
	_ "github.com/cresta/magehelper/docker/registry/ecr"
)
