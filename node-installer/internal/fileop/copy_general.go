//go:build !linux

package fileop

import (
	"os"
)

// copyAt copies the contents of src to dst.
func (o *OS) copyAt(src, dst *os.File) error {
	return o.copyTraditional(src, dst)
}
