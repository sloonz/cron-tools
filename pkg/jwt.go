package crontools

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"
	"os"
	"reflect"

	"github.com/gbrlsnchs/jwt/v3"
	"golang.org/x/crypto/ssh"
)

func NewJWTAlgorithm(privKey any) (jwt.Algorithm, error) {
	switch k := privKey.(type) {
	case *ed25519.PrivateKey:
		return jwt.NewEd25519(jwt.Ed25519PrivateKey(*k)), nil
	case *rsa.PrivateKey:
		return jwt.NewRS256(jwt.RSAPrivateKey(k)), nil
	case *ecdsa.PrivateKey:
		return jwt.NewES256(jwt.ECDSAPrivateKey(k)), nil
	default:
		return nil, fmt.Errorf("unknown private key type: %s", reflect.TypeOf(privKey).String())
	}
}

func LoadJWTAlgorithmFromPrivateKeyFile(path string) (jwt.Algorithm, error) {
	privKeyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	privKey, err := ssh.ParseRawPrivateKey(privKeyData)
	if err != nil {
		return nil, err
	}

	return NewJWTAlgorithm(privKey)
}

func SignJobToken(job Job, alg jwt.Algorithm) ([]byte, error) {
	return jwt.Sign(job, alg)
}
