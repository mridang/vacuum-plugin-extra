// Command plugin is the vacuum Go plugin that registers every extra rule. It
// must be built with -buildmode=plugin and the same toolchain and vacuum
// version as the vacuum binary that loads it:
//
//	go build -buildmode=plugin -o extra.so ./plugin
//
// Place extra.so in a directory and pass that directory to vacuum with -f.
package main

import (
	"github.com/daveshanley/vacuum/plugin"

	extra "github.com/mridang/vacuum-plugin-extra"
)

// Boot is called by vacuum when the plugin is located. It registers every
// custom function of the catalogue.
func Boot(manager *plugin.Manager) {
	extra.Register(manager)
}

func main() {
	// A Go plugin must be a main package; the entry point is Boot.
}
