/*
This class it to circumvent alg:ed25519 expectation of PrivX
*/
package privx

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	jwt "github.com/golang-jwt/jwt/v5"
)

var (
	ErrEd25519KeyType = errors.New("Ed25519 key type mismatch")
)

type signingMethodEd25519 struct{}

func (m signingMethodEd25519) Alg() string { return "Ed25519" }

// Sign signs the JWT signing string using Ed25519.
// Note: Comments are in English by request.
func (m signingMethodEd25519) Sign(signingString string, key any) ([]byte, error) {
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: got %T", ErrEd25519KeyType, key)
	}
	sig := ed25519.Sign(priv, []byte(signingString))
	return sig, nil
}

func (m signingMethodEd25519) Verify(signingString string, signature []byte, key any) error {
	pub, ok := key.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("%w: got %T", ErrEd25519KeyType, key)
	}
	if !ed25519.Verify(pub, []byte(signingString), signature) {
		return jwt.ErrSignatureInvalid
	}
	return nil
}

func init() {
	// Register custom method so parser can resolve alg="Ed25519".
	jwt.RegisterSigningMethod("Ed25519", func() jwt.SigningMethod {
		return signingMethodEd25519{}
	})
}

// Optional: helper if you want a handle without relying on init order.
func SigningMethodEd25519() jwt.SigningMethod {
	return signingMethodEd25519{}
}
