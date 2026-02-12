package ksef

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
)

// SessionService handles KSeF session lifecycle.
type SessionService struct {
	client *Client
}

// AuthorisationChallenge requests an authorisation challenge for the given NIP.
// This is the first step in the token-based authentication flow.
func (s *SessionService) AuthorisationChallenge(ctx context.Context, nip string) (*AuthorisationChallengeResponse, error) {
	req := AuthorisationChallengeRequest{
		ContextIdentifier: ContextIdentifier{
			Type:       "onip",
			Identifier: nip,
		},
	}

	var resp AuthorisationChallengeResponse
	if err := s.client.doJSON(ctx, http.MethodPost, "/online/Session/AuthorisationChallenge", req, &resp, nil); err != nil {
		return nil, fmt.Errorf("authorisation challenge: %w", err)
	}

	return &resp, nil
}

// InitToken initializes a session using a token.
// The token is encrypted with the KSeF public key before being sent.
// For the test environment the Ministry of Finance provides test tokens.
func (s *SessionService) InitToken(ctx context.Context, nip, token, challenge string) (*InitSessionResponse, error) {
	// Build the init session XML (KSeF expects octet-stream for this endpoint)
	xmlBody := buildInitTokenXML(nip, token, challenge)

	var resp InitSessionResponse
	raw, err := s.client.doRawBody(
		ctx,
		http.MethodPost,
		"/online/Session/InitToken",
		xmlBody,
		"application/octet-stream",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("init token session: %w", err)
	}

	if err := parseJSON(raw, &resp); err != nil {
		return nil, fmt.Errorf("init token session: decode response: %w", err)
	}

	return &resp, nil
}

// Status checks the current session status.
func (s *SessionService) Status(ctx context.Context, sessionToken string, pageSize, pageOffset int) (*SessionStatusResponse, error) {
	path := fmt.Sprintf("/online/Session/Status?PageSize=%d&PageOffset=%d&IncludeDetails=true", pageSize, pageOffset)

	headers := map[string]string{
		"SessionToken": sessionToken,
	}

	var resp SessionStatusResponse
	if err := s.client.doJSON(ctx, http.MethodGet, path, nil, &resp, headers); err != nil {
		return nil, fmt.Errorf("session status: %w", err)
	}

	return &resp, nil
}

// Terminate closes an active session.
func (s *SessionService) Terminate(ctx context.Context, sessionToken string) (*TerminateSessionResponse, error) {
	headers := map[string]string{
		"SessionToken": sessionToken,
	}

	var resp TerminateSessionResponse
	if err := s.client.doJSON(ctx, http.MethodGet, "/online/Session/Terminate", nil, &resp, headers); err != nil {
		return nil, fmt.Errorf("terminate session: %w", err)
	}

	return &resp, nil
}

// EncryptToken encrypts a KSeF authorization token with an RSA public key.
// The encrypted result is base64-encoded, ready for the InitToken request.
func EncryptToken(token string, publicKeyPEM []byte) (string, error) {
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return "", fmt.Errorf("ksef: failed to parse PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		cert, certErr := x509.ParseCertificate(block.Bytes)
		if certErr != nil {
			return "", fmt.Errorf("ksef: failed to parse public key: %w", err)
		}
		pub = cert.PublicKey
	}

	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("ksef: public key is not RSA")
	}

	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaKey, []byte(token), nil)
	if err != nil {
		return "", fmt.Errorf("ksef: encryption failed: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// EncryptTokenWithAES encrypts a token using AES-256-CBC as used by the KSeF API.
// Returns the base64-encoded encrypted token and the AES key/IV for the session.
func EncryptTokenWithAES(token string) (encryptedToken string, aesKey []byte, iv []byte, err error) {
	aesKey = make([]byte, 32) // AES-256
	if _, err = io.ReadFull(rand.Reader, aesKey); err != nil {
		return "", nil, nil, fmt.Errorf("ksef: generate AES key: %w", err)
	}

	iv = make([]byte, aes.BlockSize)
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", nil, nil, fmt.Errorf("ksef: generate IV: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", nil, nil, fmt.Errorf("ksef: create AES cipher: %w", err)
	}

	// PKCS7 padding
	padLen := aes.BlockSize - len(token)%aes.BlockSize
	padded := make([]byte, len(token)+padLen)
	copy(padded, token)
	for i := len(token); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	encrypted := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, padded)

	encryptedToken = base64.StdEncoding.EncodeToString(encrypted)
	return encryptedToken, aesKey, iv, nil
}
