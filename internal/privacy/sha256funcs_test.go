package privacy

import (
	"bytes"
	"crypto/md5"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptB2B(t *testing.T) {

	testingString := "\t во поле берёзка стояла, в натуре\n"

	passwordKey := MakeHash(nil, []byte(testingString), []byte(testingString)) // let it be

	keyB16 := md5.Sum(passwordKey)
	require.Equal(t, len(keyB16), 16)

	keyB := keyB16[:]
	coded, err := EncryptB2B([]byte(testingString), keyB)
	require.NoError(t, err)

	telo, err := DecryptB2B(coded, keyB)
	require.NoError(t, err)

	require.Equal(t, testingString, string(telo))

}
func TestAssym(t *testing.T) {
	// any file for testing bytes
	acc, err := os.ReadFile("../minios3/testfile.binary")
	require.NoError(t, err)

	// Public Key File
	pkb, err := os.ReadFile("../../cmd/tls/cert.pem")
	require.NoError(t, err)

	// Private key file
	priv, err := os.ReadFile("../../cmd/tls/key.pem")
	require.NoError(t, err)

	cipherByte, err := Encrypt(acc, pkb)
	require.NoError(t, err)

	txt, err := Decrypt(cipherByte, priv)
	require.NoError(t, err)

	require.True(t, bytes.Equal(acc, txt))
}
