package goflow_test

import (
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestEmpty(t *testing.T) {
	s := stream.Empty[int]()
	assert.Equal(t, 0, s.Count())
}

func TestEmptySource(t *testing.T) {
	s := stream.Empty[string]()
	assert.Empty(t, s.Collect())
}

func ExampleEmpty() {
	fmt.Println(stream.Empty[int]().Count())
	// Output: 0
}
