package lazy_calcuate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfiniteList(t *testing.T) {
	genNum := Generate(1, func(i int) int { return i + 1 })

	genOne := GenHead(genNum)
	assert.Equal(t, genOne, 1, "take one step we should got number 1")

	genFour := GenHead(GenTail(GenTail(GenTail(genNum))))
	assert.Equal(t, genFour, 4, "take four step we should got number 4")
}
