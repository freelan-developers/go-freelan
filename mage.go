// +build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	Packages = []string{
		"tuntap",
	}

	Default = All
)

func All() {
	mg.Deps(Build)
}

func Build() error {
	for _, pkg := range Packages {
		if err := sh.Run("go", "build", "./"+pkg); err != nil {
			return fmt.Errorf("building package `%s`: %v", pkg, err)
		}
	}

	mg.Deps(Test)

	return nil
}

func Test() error {
	for _, pkg := range Packages {
		if err := sh.Run("go", "build", "./"+pkg); err != nil {
			return fmt.Errorf("building package `%s`: %v", pkg, err)
		}
	}

	return nil
}
