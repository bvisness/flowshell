package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerializeNodes(t *testing.T) {
	t.Run("LoadFileAction", func(t *testing.T) {
		before := NewLoadFileNode("foo/bar")

		enc := NewEncoder(1)
		assert.True(t, SThing(enc, before))
		assert.True(t, enc.Ok())

		buf := enc.Bytes()
		t.Log("encoded:", buf)

		dec := NewDecoder(buf)
		var after Node
		assert.True(t, SThing(dec, &after))

		assert.True(t, dec.Ok())
		assert.Equal(t, *before, after)
	})
}
