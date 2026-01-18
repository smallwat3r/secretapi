package utility

import "testing"

// LowerCryptoParamsForTest lowers the argon2 params to speed up tests.
// It should only be called from tests. It uses t.Cleanup to restore the
// original values. Thread-safe for concurrent test execution.
func LowerCryptoParamsForTest(t *testing.T) {
	t.Helper()
	originalConfig := getCryptoConfig()
	setCryptoConfig(TestCryptoConfig())
	t.Cleanup(func() {
		setCryptoConfig(originalConfig)
	})
}
