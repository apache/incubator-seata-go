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

package types

import (
	"fmt"

	pb "seata-go-ai-a2a/pkg/proto/v1"
)

// Security represents security requirements for contacting the agent
type Security struct {
	Schemes map[string][]string `json:"schemes"`
}

// SecurityScheme represents different types of security schemes
type SecurityScheme interface {
	SecuritySchemeType() SecuritySchemeTypeEnum
}

// SecuritySchemeTypeEnum identifies the type of security scheme
type SecuritySchemeTypeEnum int

const (
	SecuritySchemeTypeAPIKey SecuritySchemeTypeEnum = iota
	SecuritySchemeTypeHTTPAuth
	SecuritySchemeTypeOAuth2
	SecuritySchemeTypeOpenIDConnect
	SecuritySchemeTypeMutualTLS
)

// APIKeySecurityScheme represents API key based authentication
type APIKeySecurityScheme struct {
	Description string `json:"description,omitempty"`
	Location    string `json:"location"` // "query", "header", or "cookie"
	Name        string `json:"name"`
}

func (a *APIKeySecurityScheme) SecuritySchemeType() SecuritySchemeTypeEnum {
	return SecuritySchemeTypeAPIKey
}

// HTTPAuthSecurityScheme represents HTTP authentication
type HTTPAuthSecurityScheme struct {
	Description  string `json:"description,omitempty"`
	Scheme       string `json:"scheme"`                  // HTTP Authentication scheme
	BearerFormat string `json:"bearer_format,omitempty"` // Bearer token format hint
}

func (h *HTTPAuthSecurityScheme) SecuritySchemeType() SecuritySchemeTypeEnum {
	return SecuritySchemeTypeHTTPAuth
}

// OAuth2SecurityScheme represents OAuth 2.0 authentication
type OAuth2SecurityScheme struct {
	Description       string     `json:"description,omitempty"`
	Flows             OAuthFlows `json:"flows,omitempty"`
	OAuth2MetadataURL string     `json:"oauth2_metadata_url,omitempty"`
}

func (o *OAuth2SecurityScheme) SecuritySchemeType() SecuritySchemeTypeEnum {
	return SecuritySchemeTypeOAuth2
}

// OpenIDConnectSecurityScheme represents OpenID Connect authentication
type OpenIDConnectSecurityScheme struct {
	Description      string `json:"description,omitempty"`
	OpenIDConnectURL string `json:"open_id_connect_url"`
}

func (o *OpenIDConnectSecurityScheme) SecuritySchemeType() SecuritySchemeTypeEnum {
	return SecuritySchemeTypeOpenIDConnect
}

// MutualTLSSecurityScheme represents mutual TLS authentication
type MutualTLSSecurityScheme struct {
	Description string `json:"description,omitempty"`
}

func (m *MutualTLSSecurityScheme) SecuritySchemeType() SecuritySchemeTypeEnum {
	return SecuritySchemeTypeMutualTLS
}

// OAuthFlows represents OAuth flow configurations
type OAuthFlows interface {
	OAuthFlowType() OAuthFlowTypeEnum
}

// OAuthFlowTypeEnum identifies the type of OAuth flow
type OAuthFlowTypeEnum int

const (
	OAuthFlowTypeAuthorizationCode OAuthFlowTypeEnum = iota
	OAuthFlowTypeClientCredentials
	OAuthFlowTypeImplicit
	OAuthFlowTypePassword
)

// AuthorizationCodeOAuthFlow represents authorization code OAuth flow
type AuthorizationCodeOAuthFlow struct {
	AuthorizationURL string            `json:"authorization_url"`
	TokenURL         string            `json:"token_url"`
	RefreshURL       string            `json:"refresh_url,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

func (a *AuthorizationCodeOAuthFlow) OAuthFlowType() OAuthFlowTypeEnum {
	return OAuthFlowTypeAuthorizationCode
}

// ClientCredentialsOAuthFlow represents client credentials OAuth flow
type ClientCredentialsOAuthFlow struct {
	TokenURL   string            `json:"token_url"`
	RefreshURL string            `json:"refresh_url,omitempty"`
	Scopes     map[string]string `json:"scopes,omitempty"`
}

func (c *ClientCredentialsOAuthFlow) OAuthFlowType() OAuthFlowTypeEnum {
	return OAuthFlowTypeClientCredentials
}

// ImplicitOAuthFlow represents implicit OAuth flow
type ImplicitOAuthFlow struct {
	AuthorizationURL string            `json:"authorization_url"`
	RefreshURL       string            `json:"refresh_url,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

func (i *ImplicitOAuthFlow) OAuthFlowType() OAuthFlowTypeEnum {
	return OAuthFlowTypeImplicit
}

// PasswordOAuthFlow represents password OAuth flow
type PasswordOAuthFlow struct {
	TokenURL   string            `json:"token_url"`
	RefreshURL string            `json:"refresh_url,omitempty"`
	Scopes     map[string]string `json:"scopes,omitempty"`
}

func (p *PasswordOAuthFlow) OAuthFlowType() OAuthFlowTypeEnum {
	return OAuthFlowTypePassword
}

// SecurityToProto converts Security to pb.Security
func SecurityToProto(security *Security) *pb.Security {
	if security == nil {
		return nil
	}

	schemes := make(map[string]*pb.StringList)
	for name, list := range security.Schemes {
		schemes[name] = &pb.StringList{List: list}
	}

	return &pb.Security{
		Schemes: schemes,
	}
}

// SecurityFromProto converts pb.Security to Security
func SecurityFromProto(security *pb.Security) *Security {
	if security == nil {
		return nil
	}

	schemes := make(map[string][]string)
	for name, stringList := range security.Schemes {
		schemes[name] = stringList.List
	}

	return &Security{
		Schemes: schemes,
	}
}

// SecuritySchemeToProto converts SecurityScheme to pb.SecurityScheme
func SecuritySchemeToProto(scheme SecurityScheme) (*pb.SecurityScheme, error) {
	if scheme == nil {
		return nil, nil
	}

	pbScheme := &pb.SecurityScheme{}

	switch s := scheme.(type) {
	case *APIKeySecurityScheme:
		pbScheme.Scheme = &pb.SecurityScheme_ApiKeySecurityScheme{
			ApiKeySecurityScheme: &pb.APIKeySecurityScheme{
				Description: s.Description,
				Location:    s.Location,
				Name:        s.Name,
			},
		}
	case *HTTPAuthSecurityScheme:
		pbScheme.Scheme = &pb.SecurityScheme_HttpAuthSecurityScheme{
			HttpAuthSecurityScheme: &pb.HTTPAuthSecurityScheme{
				Description:  s.Description,
				Scheme:       s.Scheme,
				BearerFormat: s.BearerFormat,
			},
		}
	case *OAuth2SecurityScheme:
		var flows *pb.OAuthFlows
		var err error
		if s.Flows != nil {
			flows, err = OAuthFlowsToProto(s.Flows)
			if err != nil {
				return nil, fmt.Errorf("converting OAuth flows: %w", err)
			}
		}
		pbScheme.Scheme = &pb.SecurityScheme_Oauth2SecurityScheme{
			Oauth2SecurityScheme: &pb.OAuth2SecurityScheme{
				Description:       s.Description,
				Flows:             flows,
				Oauth2MetadataUrl: s.OAuth2MetadataURL,
			},
		}
	case *OpenIDConnectSecurityScheme:
		pbScheme.Scheme = &pb.SecurityScheme_OpenIdConnectSecurityScheme{
			OpenIdConnectSecurityScheme: &pb.OpenIdConnectSecurityScheme{
				Description:      s.Description,
				OpenIdConnectUrl: s.OpenIDConnectURL,
			},
		}
	case *MutualTLSSecurityScheme:
		pbScheme.Scheme = &pb.SecurityScheme_MtlsSecurityScheme{
			MtlsSecurityScheme: &pb.MutualTlsSecurityScheme{
				Description: s.Description,
			},
		}
	default:
		return nil, fmt.Errorf("unknown security scheme type: %T", scheme)
	}

	return pbScheme, nil
}

// SecuritySchemeFromProto converts pb.SecurityScheme to SecurityScheme
func SecuritySchemeFromProto(scheme *pb.SecurityScheme) (SecurityScheme, error) {
	if scheme == nil {
		return nil, nil
	}

	switch s := scheme.Scheme.(type) {
	case *pb.SecurityScheme_ApiKeySecurityScheme:
		return &APIKeySecurityScheme{
			Description: s.ApiKeySecurityScheme.Description,
			Location:    s.ApiKeySecurityScheme.Location,
			Name:        s.ApiKeySecurityScheme.Name,
		}, nil
	case *pb.SecurityScheme_HttpAuthSecurityScheme:
		return &HTTPAuthSecurityScheme{
			Description:  s.HttpAuthSecurityScheme.Description,
			Scheme:       s.HttpAuthSecurityScheme.Scheme,
			BearerFormat: s.HttpAuthSecurityScheme.BearerFormat,
		}, nil
	case *pb.SecurityScheme_Oauth2SecurityScheme:
		var flows OAuthFlows
		var err error
		if s.Oauth2SecurityScheme.Flows != nil {
			flows, err = OAuthFlowsFromProto(s.Oauth2SecurityScheme.Flows)
			if err != nil {
				return nil, fmt.Errorf("converting OAuth flows: %w", err)
			}
		}
		return &OAuth2SecurityScheme{
			Description:       s.Oauth2SecurityScheme.Description,
			Flows:             flows,
			OAuth2MetadataURL: s.Oauth2SecurityScheme.Oauth2MetadataUrl,
		}, nil
	case *pb.SecurityScheme_OpenIdConnectSecurityScheme:
		return &OpenIDConnectSecurityScheme{
			Description:      s.OpenIdConnectSecurityScheme.Description,
			OpenIDConnectURL: s.OpenIdConnectSecurityScheme.OpenIdConnectUrl,
		}, nil
	case *pb.SecurityScheme_MtlsSecurityScheme:
		return &MutualTLSSecurityScheme{
			Description: s.MtlsSecurityScheme.Description,
		}, nil
	default:
		return nil, fmt.Errorf("unknown security scheme type: %T", s)
	}
}

// OAuthFlowsToProto converts OAuthFlows to pb.OAuthFlows
func OAuthFlowsToProto(flows OAuthFlows) (*pb.OAuthFlows, error) {
	if flows == nil {
		return nil, nil
	}

	pbFlows := &pb.OAuthFlows{}

	switch f := flows.(type) {
	case *AuthorizationCodeOAuthFlow:
		pbFlows.Flow = &pb.OAuthFlows_AuthorizationCode{
			AuthorizationCode: &pb.AuthorizationCodeOAuthFlow{
				AuthorizationUrl: f.AuthorizationURL,
				TokenUrl:         f.TokenURL,
				RefreshUrl:       f.RefreshURL,
				Scopes:           f.Scopes,
			},
		}
	case *ClientCredentialsOAuthFlow:
		pbFlows.Flow = &pb.OAuthFlows_ClientCredentials{
			ClientCredentials: &pb.ClientCredentialsOAuthFlow{
				TokenUrl:   f.TokenURL,
				RefreshUrl: f.RefreshURL,
				Scopes:     f.Scopes,
			},
		}
	case *ImplicitOAuthFlow:
		pbFlows.Flow = &pb.OAuthFlows_Implicit{
			Implicit: &pb.ImplicitOAuthFlow{
				AuthorizationUrl: f.AuthorizationURL,
				RefreshUrl:       f.RefreshURL,
				Scopes:           f.Scopes,
			},
		}
	case *PasswordOAuthFlow:
		pbFlows.Flow = &pb.OAuthFlows_Password{
			Password: &pb.PasswordOAuthFlow{
				TokenUrl:   f.TokenURL,
				RefreshUrl: f.RefreshURL,
				Scopes:     f.Scopes,
			},
		}
	default:
		return nil, fmt.Errorf("unknown OAuth flow type: %T", flows)
	}

	return pbFlows, nil
}

// OAuthFlowsFromProto converts pb.OAuthFlows to OAuthFlows
func OAuthFlowsFromProto(flows *pb.OAuthFlows) (OAuthFlows, error) {
	if flows == nil {
		return nil, nil
	}

	switch f := flows.Flow.(type) {
	case *pb.OAuthFlows_AuthorizationCode:
		return &AuthorizationCodeOAuthFlow{
			AuthorizationURL: f.AuthorizationCode.AuthorizationUrl,
			TokenURL:         f.AuthorizationCode.TokenUrl,
			RefreshURL:       f.AuthorizationCode.RefreshUrl,
			Scopes:           f.AuthorizationCode.Scopes,
		}, nil
	case *pb.OAuthFlows_ClientCredentials:
		return &ClientCredentialsOAuthFlow{
			TokenURL:   f.ClientCredentials.TokenUrl,
			RefreshURL: f.ClientCredentials.RefreshUrl,
			Scopes:     f.ClientCredentials.Scopes,
		}, nil
	case *pb.OAuthFlows_Implicit:
		return &ImplicitOAuthFlow{
			AuthorizationURL: f.Implicit.AuthorizationUrl,
			RefreshURL:       f.Implicit.RefreshUrl,
			Scopes:           f.Implicit.Scopes,
		}, nil
	case *pb.OAuthFlows_Password:
		return &PasswordOAuthFlow{
			TokenURL:   f.Password.TokenUrl,
			RefreshURL: f.Password.RefreshUrl,
			Scopes:     f.Password.Scopes,
		}, nil
	default:
		return nil, fmt.Errorf("unknown OAuth flow type: %T", f)
	}
}
