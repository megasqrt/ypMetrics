package middlewares

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
)

type cryptoMiddleware struct {
	privateKey *rsa.PrivateKey
}

func NewCryptoMiddleware(privateKeyPath string) (*cryptoMiddleware, error) {
	if privateKeyPath == "" {
		return &cryptoMiddleware{privateKey: nil}, nil
	}

	keyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &cryptoMiddleware{privateKey: privateKey}, nil
}

func (cm *cryptoMiddleware) CryptoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cm.privateKey == nil {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read request body")
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		decryptedBody, err := rsa.DecryptPKCS1v15(rand.Reader, cm.privateKey, body)
		if err != nil {
			log.Error().Err(err).Msg("Failed to decrypt request body")
			http.Error(w, "Failed to decrypt request body", http.StatusBadRequest)
			return
		}

		r.Body = io.NopCloser(bytes.NewBuffer(decryptedBody))
		next.ServeHTTP(w, r)
	})
}
