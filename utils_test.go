package vaar

import (
	"testing"

	"gotest.tools/v3/assert"
)

func Test_validateRelPath(t *testing.T) {
	assert.ErrorContains(t, validateRelPath(""), "empty")
	assert.ErrorContains(t, validateRelPath("/etc"), "forbidden")
	assert.ErrorContains(t, validateRelPath("./../etc"), "forbidden")
	assert.ErrorContains(t, validateRelPath("../"), "forbidden")
	assert.ErrorContains(t, validateRelPath("./.."), "forbidden")
	assert.NilError(t, validateRelPath("test"))
	assert.NilError(t, validateRelPath("test/aaa"))
	assert.NilError(t, validateRelPath("./test/././aaa"))
}
