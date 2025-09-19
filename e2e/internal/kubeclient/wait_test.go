// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kubeclient

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestStringRepresentation(t *testing.T) {
	sel := labels.SelectorFromSet(map[string]string{"foo": "bar"})
	for _, cond := range []PodCondition{
		&numReady{n: 5, ls: sel},
		&numSucceeded{n: 3, ls: sel},
		&singlePodReady{name: "my-pod"},
		&oneRunning{ls: sel},
	} {
		t.Run(fmt.Sprintf("%T", cond), func(t *testing.T) {
			assert := assert.New(t)
			assert.Implements((*fmt.Stringer)(nil), cond)
			repr := fmt.Sprint(cond)
			t.Log(repr)
			assert.False(strings.HasPrefix(repr, "&"), "String() output is not pretty")
		})
	}
}
