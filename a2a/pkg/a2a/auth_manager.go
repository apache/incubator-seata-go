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

package a2a

import (
	"context"
	"fmt"
	"sync"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/types"

	"google.golang.org/grpc/metadata"
)

// AuthManager manages multiple authenticators and provides unified authentication
type AuthManager struct {
	authenticators []auth.Authenticator
	mu             sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	return &AuthManager{
		authenticators: make([]auth.Authenticator, 0),
	}
}

// AddAuthenticator adds an authenticator to the manager
func (am *AuthManager) AddAuthenticator(authenticator auth.Authenticator) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.authenticators = append(am.authenticators, authenticator)
}

// Authenticate attempts to authenticate using all configured authenticators
func (am *AuthManager) Authenticate(ctx context.Context, md metadata.MD) (auth.User, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if len(am.authenticators) == 0 {
		// No authenticators configured, allow anonymous access
		return &auth.UnauthenticatedUser{}, nil
	}

	var lastError error

	// Try each authenticator
	for _, authenticator := range am.authenticators {
		user, err := authenticator.Authenticate(ctx, md)
		if err == nil && user.IsAuthenticated() {
			return user, nil
		}

		if err != nil {
			lastError = err
		}
	}

	// If no authenticator succeeded, return the last error
	if lastError != nil {
		return &auth.UnauthenticatedUser{}, lastError
	}

	return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
		Code:    auth.ErrCodeInvalidCredentials,
		Message: "Authentication failed with all configured authenticators",
	}
}

// GetAuthenticatorForScheme returns the first authenticator that supports the given scheme
func (am *AuthManager) GetAuthenticatorForScheme(scheme types.SecurityScheme) auth.Authenticator {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, authenticator := range am.authenticators {
		if authenticator.SupportsScheme(scheme) {
			return authenticator
		}
	}

	return nil
}

// ListAuthenticators returns the names of all configured authenticators
func (am *AuthManager) ListAuthenticators() []string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	names := make([]string, len(am.authenticators))
	for i, authenticator := range am.authenticators {
		names[i] = authenticator.Name()
	}

	return names
}

// ValidateAgentCardSecurity validates that the agent card's security requirements can be satisfied
func (am *AuthManager) ValidateAgentCardSecurity(agentCard *types.AgentCard) error {
	if agentCard.Security == nil || len(agentCard.Security) == 0 {
		// No security requirements
		return nil
	}

	am.mu.RLock()
	defer am.mu.RUnlock()

	// Check each security requirement
	for _, security := range agentCard.Security {
		satisfied := false

		for schemeName := range security.Schemes {
			// Look up the scheme in the agent card's security schemes
			if scheme, exists := agentCard.SecuritySchemes[schemeName]; exists {
				// Check if we have an authenticator that supports this scheme
				if am.GetAuthenticatorForScheme(scheme) != nil {
					satisfied = true
					break
				}
			}
		}

		if !satisfied {
			return fmt.Errorf("no authenticator available for security requirement with schemes: %v",
				getSchemeNames(security.Schemes))
		}
	}

	return nil
}

// getSchemeNames extracts scheme names from a security schemes map
func getSchemeNames(schemes map[string][]string) []string {
	names := make([]string, 0, len(schemes))
	for name := range schemes {
		names = append(names, name)
	}
	return names
}
