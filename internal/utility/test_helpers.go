package utility

import "testing"

// LowerCryptoParamsForTest lowers the argon2 params to speed up tests.
// It should only be called from tests. It uses t.Cleanup to restore the
// original values.
func LowerCryptoParamsForTest(t *testing.T) {
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
