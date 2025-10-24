package utility

import (
	"testing"
)

func lowerCryptoParamsForTest(t *testing.T) {
	t.Helper()
	originalArgonTime := ArgonTime
	originalArgonMemory := ArgonMemory
	ArgonTime = 1
	ArgonMemory = 1024
	t.Cleanup(func() {
		ArgonTime = originalArgonTime
		ArgonMemory = originalArgonMemory
	})
}
