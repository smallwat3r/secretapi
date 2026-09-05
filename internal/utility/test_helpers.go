package utility

import "testing"

// LowerCryptoParamsForTest lowers the argon2 memory cost to speed up tests
// and restores it on cleanup. Tests in this module do not run in parallel.
func LowerCryptoParamsForTest(t *testing.T) {
	t.Helper()
	original := argonMemory
	argonMemory = 1024 // 1 MB
	t.Cleanup(func() { argonMemory = original })
}
