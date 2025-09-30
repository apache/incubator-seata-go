/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jws

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/types"
)

// Signer implements JWS signing for Agent Cards
type Signer struct{}

// NewSigner creates a new JWS signer
func NewSigner() *Signer {
	return &Signer{}
}

// SignAgentCard signs an Agent Card and returns the signature
func (s *Signer) SignAgentCard(ctx context.Context, agentCard *types.AgentCard, privateKey crypto.PrivateKey, keyID, jwksURL string) (*types.AgentCardSignature, error) {
	// Remove existing signatures from the agent card for signing
	agentCardCopy := *agentCard
	agentCardCopy.Signatures = nil

	// Serialize the agent card (canonical JSON)
	payload, err := json.Marshal(agentCardCopy)
	if err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to serialize agent card",
			Details: err.Error(),
		}
	}

	// Create JWK from private key
	jwkKey, err := jwk.FromRaw(privateKey)
	if err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to create JWK from private key",
			Details: err.Error(),
		}
	}

	// Set key metadata
	if err := jwkKey.Set(jwk.KeyIDKey, keyID); err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to set key ID",
			Details: err.Error(),
		}
	}

	// Determine the algorithm based on the actual key type (not interface)
	algorithm, err := s.determineAlgorithm(privateKey)
	if err != nil {
		return nil, err
	}

	if err := jwkKey.Set(jwk.AlgorithmKey, algorithm); err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to set algorithm",
			Details: err.Error(),
		}
	}

	// Create JWS headers
	headers := jws.NewHeaders()
	if err := headers.Set(jws.AlgorithmKey, algorithm); err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to set algorithm header",
			Details: err.Error(),
		}
	}
	if err := headers.Set(jws.TypeKey, "JOSE"); err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to set type header",
			Details: err.Error(),
		}
	}
	if err := headers.Set(jws.KeyIDKey, keyID); err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to set key ID header",
			Details: err.Error(),
		}
	}
	if err := headers.Set("jku", jwksURL); err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to set JWKS URL header",
			Details: err.Error(),
		}
	}

	// Sign the payload
	signed, err := jws.Sign(payload, jws.WithKey(algorithm, jwkKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to sign agent card",
			Details: err.Error(),
		}
	}

	// Parse the JWS to extract components
	parsedJWS, err := jws.Parse(signed)
	if err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to parse signed JWS",
			Details: err.Error(),
		}
	}

	// Extract the signature components
	if len(parsedJWS.Signatures()) == 0 {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "No signatures found in JWS",
		}
	}

	signature := parsedJWS.Signatures()[0]

	// Marshal protected headers to get bytes
	protectedBytes, err := json.Marshal(signature.ProtectedHeaders())
	if err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to marshal protected headers",
			Details: err.Error(),
		}
	}

	// Convert public headers (unprotected) to map[string]any if they exist
	headerMap := make(map[string]any)
	if signature.PublicHeaders() != nil {
		if publicHeaderMap, err := signature.PublicHeaders().AsMap(context.Background()); err == nil {
			headerMap = publicHeaderMap
		}
	}

	return &types.AgentCardSignature{
		Protected: base64.RawURLEncoding.EncodeToString(protectedBytes),
		Signature: base64.RawURLEncoding.EncodeToString(signature.Signature()),
		Header:    headerMap,
	}, nil
}

// determineAlgorithm determines the appropriate JWS algorithm for the given private key
func (s *Signer) determineAlgorithm(privateKey crypto.PrivateKey) (jwa.SignatureAlgorithm, error) {
	switch key := privateKey.(type) {
	case *ecdsa.PrivateKey:
		// ECDSA keys
		switch key.Curve.Params().BitSize {
		case 256:
			return jwa.ES256, nil
		case 384:
			return jwa.ES384, nil
		case 521:
			return jwa.ES512, nil
		default:
			return "", &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidSignature,
				Message: "Unsupported ECDSA curve",
				Details: fmt.Sprintf("Curve bit size: %d", key.Curve.Params().BitSize),
			}
		}
	case *rsa.PrivateKey:
		// RSA keys - use RS256 for most cases
		return jwa.RS256, nil
	case ed25519.PrivateKey:
		// EdDSA keys
		return jwa.EdDSA, nil
	default:
		return "", &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Unsupported private key type",
			Details: fmt.Sprintf("Key type: %T", privateKey),
		}
	}
}
