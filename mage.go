// +build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	// Packages is the list of packages.
	Packages = []string{
		"tuntap",
		"fscp",
	}

	// Default is the default target.
	Default = All
)

// All the targets are run.
func All() {
	mg.Deps(Build)
}

// Build the code.
func Build() error {
	for _, pkg := range Packages {
		if err := sh.Run("go", "build", "./"+pkg); err != nil {
			return fmt.Errorf("building package `%s`: %v", pkg, err)
		}
	}

	mg.Deps(Test)

	return nil
}

// Test the code.
func Test() error {
	for _, pkg := range Packages {
		args := []string{"test", "./" + pkg}

		if mg.Verbose() {
			args = append(args, "-v")
		}

		if err := sh.RunV("go", args...); err != nil {
			return fmt.Errorf("building package `%s`: %v", pkg, err)
		}
	}

	return nil
}
