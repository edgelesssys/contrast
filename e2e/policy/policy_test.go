package policy

import (
	"testing"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
)

var imageReplacementsFile string

func TestPolicy(t *testing.T) {
	ct := contrasttest.New(t, imageReplacementsFile)
}
