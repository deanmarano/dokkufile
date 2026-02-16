package state

import (
	"github.com/deanmarano/dokkufile/pkg/schema"
)

// Reader reads live state from a dokku server.
type Reader interface {
	Read() (*schema.Dokkufile, error)
}

// DokkuReader reads state by shelling out to dokku commands.
type DokkuReader struct{}

// Read shells out to dokku to build the current server state.
// TODO: implement by parsing output of dokku apps:list, dokku config:show, etc.
func (r *DokkuReader) Read() (*schema.Dokkufile, error) {
	return &schema.Dokkufile{
		Version:  "1",
		Services: map[string]schema.Service{},
		Apps:     map[string]schema.App{},
	}, nil
}
