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

package auth

// AuthenticatedUser represents an authenticated user
type AuthenticatedUser struct {
	userID string
	name   string
	email  string
	scopes []string
	claims map[string]interface{}
}

// NewAuthenticatedUser creates a new authenticated user
func NewAuthenticatedUser(userID, name, email string, scopes []string, claims map[string]interface{}) *AuthenticatedUser {
	if claims == nil {
		claims = make(map[string]interface{})
	}

	return &AuthenticatedUser{
		userID: userID,
		name:   name,
		email:  email,
		scopes: scopes,
		claims: claims,
	}
}

func (u *AuthenticatedUser) GetUserID() string                 { return u.userID }
func (u *AuthenticatedUser) GetName() string                   { return u.name }
func (u *AuthenticatedUser) GetEmail() string                  { return u.email }
func (u *AuthenticatedUser) GetScopes() []string               { return u.scopes }
func (u *AuthenticatedUser) GetClaims() map[string]interface{} { return u.claims }
func (u *AuthenticatedUser) IsAuthenticated() bool             { return true }

// UnauthenticatedUser represents an unauthenticated user
type UnauthenticatedUser struct{}

func (u *UnauthenticatedUser) GetUserID() string                 { return "" }
func (u *UnauthenticatedUser) GetName() string                   { return "" }
func (u *UnauthenticatedUser) GetEmail() string                  { return "" }
func (u *UnauthenticatedUser) GetScopes() []string               { return nil }
func (u *UnauthenticatedUser) GetClaims() map[string]interface{} { return nil }
func (u *UnauthenticatedUser) IsAuthenticated() bool             { return false }
