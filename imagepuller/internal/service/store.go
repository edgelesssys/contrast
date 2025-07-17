package service

import (
	"io"

	"github.com/containers/storage"
)

// Store is a subset of containers' storage.Store interface.
type Store interface {
	// AddNames adds the list of names for a layer, image, or container.
	// Duplicate names are removed from the list automatically.
	AddNames(id string, names []string) error

	// CreateContainer creates a new container, optionally with the
	// specified ID (one will be assigned if none is specified), with
	// optional names, using the specified image's top layer as the basis
	// for the container's layer, and assigning the specified ID to that
	// layer (one will be created if none is specified).  A container is a
	// layer which is associated with additional bookkeeping information
	// which the library stores for the convenience of its caller.
	CreateContainer(id string, names []string, image, layer, metadata string, options *storage.ContainerOptions) (*storage.Container, error)

	// CreateImage creates a new image, optionally with the specified ID
	// (one will be assigned if none is specified), with optional names,
	// referring to a specified image, and with optional metadata.  An
	// image is a record which associates the ID of a layer with a
	// additional bookkeeping information which the library stores for the
	// convenience of its caller.
	CreateImage(id string, names []string, layer, metadata string, options *storage.ImageOptions) (*storage.Image, error)

	// Lookup returns the ID of a layer, image, or container with the specified
	// name or ID.
	Lookup(name string) (string, error)

	// Mount attempts to mount a layer, image, or container for access, and
	// returns the pathname if it succeeds.
	// Note if the mountLabel == "", the default label for the container
	// will be used.
	//
	// Note that we do some of this work in a child process.  The calling
	// process's main() function needs to import our pkg/reexec package and
	// should begin with something like this in order to allow us to
	// properly start that child process:
	//   if reexec.Init() {
	//       return
	//   }
	Mount(id, mountLabel string) (string, error)

	// PutLayer combines the functions of CreateLayer and ApplyDiff,
	// marking the layer for automatic removal if applying the diff fails
	// for any reason.
	//
	// Note that we do some of this work in a child process.  The calling
	// process's main() function needs to import our pkg/reexec package and
	// should begin with something like this in order to allow us to
	// properly start that child process:
	//   if reexec.Init() {
	//       return
	//   }
	PutLayer(id, parent string, names []string, mountLabel string, writeable bool, options *storage.LayerOptions, diff io.Reader) (*storage.Layer, int64, error)

	// RemoveNames removes the list of names for a layer, image, or container.
	// Duplicate names are removed from the list automatically.
	RemoveNames(id string, names []string) error
}
