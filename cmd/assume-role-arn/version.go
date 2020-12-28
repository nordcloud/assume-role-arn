package main

import (
	"fmt"
	"strings"
)

// Revision contains the Git commit that was compiled.
var Revision string

// Version contains the main version number that is being
// run at the moment.
var Version string

func formattedVersion() string {
	var versionString strings.Builder

	fmt.Fprintf(&versionString, "%s", Version)
	if Revision != "" {
		fmt.Fprintf(&versionString, " (%s)", Revision)
	}

	return versionString.String()
}
