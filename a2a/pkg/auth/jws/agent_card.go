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
	"encoding/json"
	"fmt"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/types"
)

// ValidateAgentCard validates all JWS signatures in an Agent Card
func ValidateAgentCard(ctx context.Context, agentCard *types.AgentCard, validator auth.JWSValidator) error {
	if len(agentCard.Signatures) == 0 {
		// No signatures to validate
		return nil
	}

	// Prepare the payload (agent card without signatures)
	agentCardCopy := *agentCard
	agentCardCopy.Signatures = nil

	payload, err := json.Marshal(agentCardCopy)
	if err != nil {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to serialize agent card for validation",
			Details: err.Error(),
		}
	}

	// Validate each signature
	for i, signature := range agentCard.Signatures {
		if err := validator.ValidateSignature(ctx, payload, signature); err != nil {
			return fmt.Errorf("signature validation failed at index %d: %w", i, err)
		}
	}

	return nil
}

// SignAgentCardInPlace signs an agent card and adds the signature to the card
func SignAgentCardInPlace(ctx context.Context, agentCard *types.AgentCard, signer auth.JWSSigner, privateKey crypto.PrivateKey, keyID, jwksURL string) error {
	signature, err := signer.SignAgentCard(ctx, agentCard, privateKey, keyID, jwksURL)
	if err != nil {
		return err
	}

	// Add signature to the agent card
	agentCard.Signatures = append(agentCard.Signatures, signature)

	return nil
}
