package utility

import "testing"

// LowerCryptoParamsForTest lowers the argon2 params to speed up tests.
// It should only be called from tests. It uses t.Cleanup to restore the
// original values.
func LowerCryptoParamsForTest(t *testing.T) {
	t.Helper()
	originalConfig := cryptoConfig
	cryptoConfig = TestCryptoConfig()
	t.Cleanup(func() {
		cryptoConfig = originalConfig
	})
}
