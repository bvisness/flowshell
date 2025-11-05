package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerializeNodes(t *testing.T) {
	t.Run("LoadFileAction", func(t *testing.T) {
		before := NewLoadFileNode("foo/bar")

		enc := NewEncoder(1)
		before.Serialize(enc, before)
		assert.Nil(t, enc.Err)

		buf := enc.Bytes()
		t.Log(buf)

		dec := NewDecoder(buf)
		var after Node
		after.Serialize(dec, &after)

		assert.Nil(t, dec.Err)
		assert.Equal(t, *before, after)
	})
}
