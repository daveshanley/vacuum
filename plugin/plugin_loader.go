package plugin

import (
	"github.com/pterm/pterm"
	"io/ioutil"
	"path/filepath"
	"plugin"
	"strings"
)

// LoadFunctions will load custom functions found in the supplied path
func LoadFunctions(path string) (*Manager, error) {

	dirEntries, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	pm := createPluginManager()

	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") {
			fPath := filepath.Join(path, entry.Name())

			// found something
			pterm.Info.Printf("Located custom function plugin: %s\n", fPath)

			// let's try and open it.
			p, e := plugin.Open(fPath)
			if e != nil {
				return nil, e
			}

			// look up the Boot function and store as a Symbol
			var bootFunc plugin.Symbol
			bootFunc, e = p.Lookup("Boot")
			if err != nil {
				return nil, err
			}

			// lets go pedro!
			if bootFunc != nil {
				bootFunc.(func(*Manager))(pm)
			} else {
				pterm.Error.Printf("Unable to boot plugin")
			}
		}
	}

	return pm, nil
}
