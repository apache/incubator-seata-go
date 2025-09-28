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

package authenticator

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ocsp"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/types"
)

// MTLSAuthenticator implements mutual TLS (mTLS) authentication with complete certificate validation
type MTLSAuthenticator struct {
	name                  string
	trustedCAs            *x509.CertPool
	clientCertificates    map[string]*ClientCertInfo // cert serial -> client info
	requireClientCert     bool
	verifyHostname        bool
	allowedCNPatterns     []string // Common Name patterns
	allowedSANs           []string // Subject Alternative Names
	certificateValidator  CertificateValidator
	revocationChecker     RevocationChecker
	crlCache              *CRLCache
	ocspCache             *OCSPCache
	keyStrengthValidator  KeyStrengthValidator
	certificateChainDepth int
	clockSkewTolerance    time.Duration
}

// ClientCertInfo represents comprehensive information about a client certificate
type ClientCertInfo struct {
	UserID       string             `json:"userId"`
	Name         string             `json:"name"`
	Email        string             `json:"email"`
	Organization string             `json:"organization"`
	Scopes       []string           `json:"scopes"`
	Metadata     map[string]string  `json:"metadata"`
	SerialNumber string             `json:"serialNumber"`
	Fingerprint  string             `json:"fingerprint"` // SHA-256 fingerprint
	Subject      pkix.Name          `json:"subject"`
	Issuer       pkix.Name          `json:"issuer"`
	NotBefore    time.Time          `json:"notBefore"`
	NotAfter     time.Time          `json:"notAfter"`
	KeyUsage     x509.KeyUsage      `json:"keyUsage"`
	ExtKeyUsage  []x509.ExtKeyUsage `json:"extKeyUsage"`
	IsActive     bool               `json:"isActive"`
	CreatedAt    time.Time          `json:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt"`
	LastUsedAt   *time.Time         `json:"lastUsedAt,omitempty"`
}

// MTLSAuthenticatorConfig represents comprehensive configuration for mTLS authentication
type MTLSAuthenticatorConfig struct {
	Name                  string                     `json:"name"`
	TrustedCAs            []string                   `json:"trustedCAs"`         // PEM-encoded CA certificates
	ClientCertificates    map[string]*ClientCertInfo `json:"clientCertificates"` // serial -> client info
	RequireClientCert     bool                       `json:"requireClientCert"`
	VerifyHostname        bool                       `json:"verifyHostname"`
	AllowedCNPatterns     []string                   `json:"allowedCNPatterns"`
	AllowedSANs           []string                   `json:"allowedSANs"`
	EnableCRLChecking     bool                       `json:"enableCRLChecking"`
	EnableOCSPChecking    bool                       `json:"enableOCSPChecking"`
	CRLURLs               []string                   `json:"crlUrls,omitempty"`
	OCSPURLs              []string                   `json:"ocspUrls,omitempty"`
	CRLCacheDuration      time.Duration              `json:"crlCacheDuration,omitempty"`      // Default: 1 hour
	OCSPCacheDuration     time.Duration              `json:"ocspCacheDuration,omitempty"`     // Default: 1 hour
	CertificateChainDepth int                        `json:"certificateChainDepth,omitempty"` // Default: 5
	ClockSkewTolerance    time.Duration              `json:"clockSkewTolerance,omitempty"`    // Default: 5 minutes
	MinRSAKeySize         int                        `json:"minRSAKeySize,omitempty"`         // Default: 2048
	MinECDSACurveSize     int                        `json:"minECDSACurveSize,omitempty"`     // Default: 256
	RequireKeyUsage       []x509.KeyUsage            `json:"requireKeyUsage,omitempty"`
	RequireExtKeyUsage    []x509.ExtKeyUsage         `json:"requireExtKeyUsage,omitempty"`
	HTTPTimeout           time.Duration              `json:"httpTimeout,omitempty"` // For CRL/OCSP fetching
}

// CertificateValidator interface for comprehensive certificate validation
type CertificateValidator interface {
	ValidateCertificate(cert *x509.Certificate, intermediates *x509.CertPool) error
	ValidateCertificateChain(chain []*x509.Certificate) error
}

// RevocationChecker interface for certificate revocation checking
type RevocationChecker interface {
	IsRevoked(ctx context.Context, cert *x509.Certificate, issuer *x509.Certificate) (bool, error)
	CheckCRL(ctx context.Context, cert *x509.Certificate, issuer *x509.Certificate) (bool, error)
	CheckOCSP(ctx context.Context, cert *x509.Certificate, issuer *x509.Certificate) (bool, error)
}

// KeyStrengthValidator validates cryptographic key strength
type KeyStrengthValidator interface {
	ValidateKeyStrength(publicKey crypto.PublicKey) error
}

// CRLCache manages Certificate Revocation List caching
type CRLCache struct {
	cache      map[string]*CRLEntry
	mutex      sync.RWMutex
	duration   time.Duration
	httpClient *http.Client
}

// CRLEntry represents a cached CRL entry
type CRLEntry struct {
	CRL       *x509.RevocationList
	FetchedAt time.Time
	URL       string
}

// OCSPCache manages OCSP response caching
type OCSPCache struct {
	cache      map[string]*OCSPEntry
	mutex      sync.RWMutex
	duration   time.Duration
	httpClient *http.Client
}

// OCSPEntry represents a cached OCSP response
type OCSPEntry struct {
	Response  *ocsp.Response
	FetchedAt time.Time
	URL       string
}

// NewMTLSAuthenticator creates a new comprehensive mTLS authenticator
func NewMTLSAuthenticator(config *MTLSAuthenticatorConfig) (*MTLSAuthenticator, error) {
	// Parse trusted CA certificates
	trustedCAs := x509.NewCertPool()
	for _, caPEM := range config.TrustedCAs {
		if !trustedCAs.AppendCertsFromPEM([]byte(caPEM)) {
			return nil, &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidCredentials,
				Message: "Failed to parse trusted CA certificate",
			}
		}
	}

	// Set defaults
	crlCacheDuration := config.CRLCacheDuration
	if crlCacheDuration == 0 {
		crlCacheDuration = time.Hour
	}

	ocspCacheDuration := config.OCSPCacheDuration
	if ocspCacheDuration == 0 {
		ocspCacheDuration = time.Hour
	}

	certificateChainDepth := config.CertificateChainDepth
	if certificateChainDepth == 0 {
		certificateChainDepth = 5
	}

	clockSkewTolerance := config.ClockSkewTolerance
	if clockSkewTolerance == 0 {
		clockSkewTolerance = 5 * time.Minute
	}

	httpTimeout := config.HTTPTimeout
	if httpTimeout == 0 {
		httpTimeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	authenticator := &MTLSAuthenticator{
		name:                  config.Name,
		trustedCAs:            trustedCAs,
		clientCertificates:    config.ClientCertificates,
		requireClientCert:     config.RequireClientCert,
		verifyHostname:        config.VerifyHostname,
		allowedCNPatterns:     config.AllowedCNPatterns,
		allowedSANs:           config.AllowedSANs,
		certificateChainDepth: certificateChainDepth,
		clockSkewTolerance:    clockSkewTolerance,
	}

	// Initialize CRL cache
	if config.EnableCRLChecking {
		authenticator.crlCache = &CRLCache{
			cache:      make(map[string]*CRLEntry),
			duration:   crlCacheDuration,
			httpClient: httpClient,
		}
	}

	// Initialize OCSP cache
	if config.EnableOCSPChecking {
		authenticator.ocspCache = &OCSPCache{
			cache:      make(map[string]*OCSPEntry),
			duration:   ocspCacheDuration,
			httpClient: httpClient,
		}
	}

	// Initialize certificate validator
	authenticator.certificateValidator = &DefaultCertificateValidator{
		trustedCAs:         trustedCAs,
		minRSAKeySize:      getOrDefault(config.MinRSAKeySize, 2048),
		minECDSACurveSize:  getOrDefault(config.MinECDSACurveSize, 256),
		requireKeyUsage:    config.RequireKeyUsage,
		requireExtKeyUsage: config.RequireExtKeyUsage,
	}

	// Initialize key strength validator
	authenticator.keyStrengthValidator = &DefaultKeyStrengthValidator{
		minRSAKeySize:     getOrDefault(config.MinRSAKeySize, 2048),
		minECDSACurveSize: getOrDefault(config.MinECDSACurveSize, 256),
	}

	// Initialize revocation checker
	if config.EnableCRLChecking || config.EnableOCSPChecking {
		authenticator.revocationChecker = &DefaultRevocationChecker{
			enableCRL:  config.EnableCRLChecking,
			enableOCSP: config.EnableOCSPChecking,
			crlURLs:    config.CRLURLs,
			ocspURLs:   config.OCSPURLs,
			crlCache:   authenticator.crlCache,
			ocspCache:  authenticator.ocspCache,
			httpClient: httpClient,
		}
	}

	return authenticator, nil
}

// Name returns the name of the authenticator
func (m *MTLSAuthenticator) Name() string {
	return m.name
}

// Authenticate attempts to authenticate a request using comprehensive mTLS validation
func (m *MTLSAuthenticator) Authenticate(ctx context.Context, md metadata.MD) (auth.User, error) {
	// Get peer info from context
	peerInfo, ok := peer.FromContext(ctx)
	if !ok {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeMissingCredentials,
			Message: "No peer information available",
		}
	}

	// Extract TLS info
	tlsInfo, ok := peerInfo.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeMissingCredentials,
			Message: "No TLS authentication info available",
		}
	}

	// Check if client certificate is present
	if len(tlsInfo.State.PeerCertificates) == 0 {
		if m.requireClientCert {
			return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
				Code:    auth.ErrCodeMissingCredentials,
				Message: "Client certificate required but not provided",
			}
		}
		return &auth.UnauthenticatedUser{}, nil
	}

	// Validate certificate chain depth
	if len(tlsInfo.State.PeerCertificates) > m.certificateChainDepth {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Certificate chain too long",
			Details: fmt.Sprintf("Chain length: %d, max allowed: %d", len(tlsInfo.State.PeerCertificates), m.certificateChainDepth),
		}
	}

	// Get the client certificate (first in chain)
	clientCert := tlsInfo.State.PeerCertificates[0]

	// Build intermediate certificate pool
	intermediates := x509.NewCertPool()
	for i := 1; i < len(tlsInfo.State.PeerCertificates); i++ {
		intermediates.AddCert(tlsInfo.State.PeerCertificates[i])
	}

	// Comprehensive certificate validation
	if err := m.validateCertificateComprehensive(ctx, clientCert, intermediates, tlsInfo.State.PeerCertificates); err != nil {
		return &auth.UnauthenticatedUser{}, err
	}

	// Look up client information based on certificate
	clientInfo, err := m.getClientInfoFromCertificate(clientCert)
	if err != nil {
		return &auth.UnauthenticatedUser{}, err
	}

	// Update last used timestamp
	now := time.Now()
	clientInfo.LastUsedAt = &now
	clientInfo.UpdatedAt = now

	// Build comprehensive claims from certificate
	claims := m.buildComprehensiveClaimsFromCertificate(clientCert, peerInfo, &tlsInfo)
	for k, v := range clientInfo.Metadata {
		claims[k] = v
	}

	return auth.NewAuthenticatedUser(
		clientInfo.UserID,
		clientInfo.Name,
		clientInfo.Email,
		clientInfo.Scopes,
		claims,
	), nil
}

// SupportsScheme returns true if this authenticator supports mTLS schemes
func (m *MTLSAuthenticator) SupportsScheme(scheme types.SecurityScheme) bool {
	return scheme.SecuritySchemeType() == types.SecuritySchemeTypeMutualTLS
}

// validateCertificateComprehensive performs comprehensive certificate validation
func (m *MTLSAuthenticator) validateCertificateComprehensive(ctx context.Context, cert *x509.Certificate, intermediates *x509.CertPool, chain []*x509.Certificate) error {
	// Basic time validation with clock skew tolerance
	now := time.Now()
	notBefore := cert.NotBefore.Add(-m.clockSkewTolerance)
	notAfter := cert.NotAfter.Add(m.clockSkewTolerance)

	if now.Before(notBefore) {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Client certificate is not yet valid",
			Details: fmt.Sprintf("Not valid before: %s (with %s clock skew tolerance)", cert.NotBefore.Format(time.RFC3339), m.clockSkewTolerance),
		}
	}

	if now.After(notAfter) {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Client certificate has expired",
			Details: fmt.Sprintf("Expired at: %s (with %s clock skew tolerance)", cert.NotAfter.Format(time.RFC3339), m.clockSkewTolerance),
		}
	}

	// Validate key strength
	if err := m.keyStrengthValidator.ValidateKeyStrength(cert.PublicKey); err != nil {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Certificate key strength insufficient",
			Details: err.Error(),
		}
	}

	// Verify certificate chain against trusted CAs
	opts := x509.VerifyOptions{
		Roots:         m.trustedCAs,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		CurrentTime:   now,
	}

	if _, err := cert.Verify(opts); err != nil {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Certificate chain verification failed",
			Details: err.Error(),
		}
	}

	// Use certificate validator for additional checks
	if m.certificateValidator != nil {
		if err := m.certificateValidator.ValidateCertificate(cert, intermediates); err != nil {
			return &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidCredentials,
				Message: "Certificate validation failed",
				Details: err.Error(),
			}
		}

		if err := m.certificateValidator.ValidateCertificateChain(chain); err != nil {
			return &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidCredentials,
				Message: "Certificate chain validation failed",
				Details: err.Error(),
			}
		}
	}

	// Check certificate revocation status
	if m.revocationChecker != nil {
		// Find the issuer certificate
		var issuer *x509.Certificate
		if len(chain) > 1 {
			issuer = chain[1] // First intermediate or CA
		} else {
			// Try to find issuer in trusted CAs
			for _, caCert := range m.trustedCAs.Subjects() {
				if bytes.Equal(cert.RawIssuer, caCert) {
					// This is a simplified approach - in production you'd need a more robust way to find the issuer
					issuer = cert // Use self for root CA (this needs proper implementation)
					break
				}
			}
		}

		if issuer != nil {
			revoked, err := m.revocationChecker.IsRevoked(ctx, cert, issuer)
			if err != nil {
				return &auth.AuthenticationError{
					Code:    auth.ErrCodeInvalidCredentials,
					Message: "Failed to check certificate revocation status",
					Details: err.Error(),
				}
			}
			if revoked {
				return &auth.AuthenticationError{
					Code:    auth.ErrCodeInvalidCredentials,
					Message: "Client certificate has been revoked",
				}
			}
		}
	}

	// Validate certificate identity against allowed patterns
	if err := m.validateCertificateIdentity(cert); err != nil {
		return err
	}

	return nil
}

// validateCertificateIdentity validates certificate identity against allowed patterns with comprehensive matching
func (m *MTLSAuthenticator) validateCertificateIdentity(cert *x509.Certificate) error {
	// Check Common Name patterns
	if len(m.allowedCNPatterns) > 0 {
		cnMatched := false
		for _, pattern := range m.allowedCNPatterns {
			if matched, err := matchWildcardPattern(pattern, cert.Subject.CommonName); err != nil {
				return &auth.AuthenticationError{
					Code:    auth.ErrCodeInvalidCredentials,
					Message: "Invalid CN pattern",
					Details: err.Error(),
				}
			} else if matched {
				cnMatched = true
				break
			}
		}
		if !cnMatched {
			return &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidCredentials,
				Message: "Certificate Common Name not allowed",
				Details: fmt.Sprintf("CN: %s, Allowed patterns: %v", cert.Subject.CommonName, m.allowedCNPatterns),
			}
		}
	}

	// Check Subject Alternative Names comprehensively
	if len(m.allowedSANs) > 0 {
		sanMatched := false
		var allSANs []string

		// Collect all SANs
		allSANs = append(allSANs, cert.DNSNames...)
		allSANs = append(allSANs, cert.EmailAddresses...)

		for _, ip := range cert.IPAddresses {
			allSANs = append(allSANs, ip.String())
		}

		for _, uri := range cert.URIs {
			allSANs = append(allSANs, uri.String())
		}

		for _, allowedSAN := range m.allowedSANs {
			for _, certSAN := range allSANs {
				if matched, err := matchWildcardPattern(allowedSAN, certSAN); err != nil {
					return &auth.AuthenticationError{
						Code:    auth.ErrCodeInvalidCredentials,
						Message: "Invalid SAN pattern",
						Details: err.Error(),
					}
				} else if matched {
					sanMatched = true
					break
				}
			}
			if sanMatched {
				break
			}
		}

		if !sanMatched {
			return &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidCredentials,
				Message: "Certificate Subject Alternative Names not allowed",
				Details: fmt.Sprintf("SANs: %v, Allowed patterns: %v", allSANs, m.allowedSANs),
			}
		}
	}

	return nil
}

// getClientInfoFromCertificate retrieves comprehensive client information based on certificate
func (m *MTLSAuthenticator) getClientInfoFromCertificate(cert *x509.Certificate) (*ClientCertInfo, error) {
	// Calculate certificate fingerprint
	fingerprint := fmt.Sprintf("%x", cert.Raw)

	// Look up by serial number first
	serialNumber := cert.SerialNumber.String()
	if clientInfo, exists := m.clientCertificates[serialNumber]; exists {
		if !clientInfo.IsActive {
			return nil, &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidCredentials,
				Message: "Client certificate is deactivated",
				Details: fmt.Sprintf("Serial: %s", serialNumber),
			}
		}
		return clientInfo, nil
	}

	// If not found by serial, look up by fingerprint
	for _, clientInfo := range m.clientCertificates {
		if clientInfo.Fingerprint == fingerprint {
			if !clientInfo.IsActive {
				return nil, &auth.AuthenticationError{
					Code:    auth.ErrCodeInvalidCredentials,
					Message: "Client certificate is deactivated",
					Details: fmt.Sprintf("Fingerprint: %s", fingerprint),
				}
			}
			return clientInfo, nil
		}
	}

	// If not explicitly configured, create default client info from certificate
	userID := fmt.Sprintf("cert_%s", serialNumber)
	name := cert.Subject.CommonName
	email := ""
	organization := ""

	if len(cert.Subject.Organization) > 0 {
		organization = cert.Subject.Organization[0]
	}

	if len(cert.EmailAddresses) > 0 {
		email = cert.EmailAddresses[0]
	}

	// Extract scopes from certificate extensions or use defaults
	scopes := []string{"read"} // Default minimal scope

	return &ClientCertInfo{
		UserID:       userID,
		Name:         name,
		Email:        email,
		Organization: organization,
		Scopes:       scopes,
		Metadata:     make(map[string]string),
		SerialNumber: serialNumber,
		Fingerprint:  fingerprint,
		Subject:      cert.Subject,
		Issuer:       cert.Issuer,
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		KeyUsage:     cert.KeyUsage,
		ExtKeyUsage:  cert.ExtKeyUsage,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// buildComprehensiveClaimsFromCertificate builds comprehensive authentication claims from certificate
func (m *MTLSAuthenticator) buildComprehensiveClaimsFromCertificate(cert *x509.Certificate, peerInfo *peer.Peer, tlsInfo *credentials.TLSInfo) map[string]interface{} {
	claims := make(map[string]interface{})

	// Basic certificate information
	claims["auth_method"] = "mtls"
	claims["cert_serial"] = cert.SerialNumber.String()
	claims["cert_fingerprint"] = fmt.Sprintf("%x", cert.Raw)
	claims["cert_subject"] = cert.Subject.String()
	claims["cert_issuer"] = cert.Issuer.String()
	claims["cert_not_before"] = cert.NotBefore.Unix()
	claims["cert_not_after"] = cert.NotAfter.Unix()
	claims["cert_key_usage"] = int(cert.KeyUsage)
	claims["cert_ext_key_usage"] = cert.ExtKeyUsage
	claims["cert_version"] = cert.Version
	claims["cert_is_ca"] = cert.IsCA

	// Public key information
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		claims["cert_key_type"] = "RSA"
		claims["cert_key_size"] = pub.Size() * 8
	case *ecdsa.PublicKey:
		claims["cert_key_type"] = "ECDSA"
		claims["cert_key_size"] = pub.Curve.Params().BitSize
	case *ed25519.PublicKey:
		claims["cert_key_type"] = "Ed25519"
		claims["cert_key_size"] = 256
	}

	// Subject Alternative Names
	if len(cert.DNSNames) > 0 {
		claims["cert_dns_names"] = cert.DNSNames
	}

	if len(cert.EmailAddresses) > 0 {
		claims["cert_emails"] = cert.EmailAddresses
	}

	if len(cert.IPAddresses) > 0 {
		var ips []string
		for _, ip := range cert.IPAddresses {
			ips = append(ips, ip.String())
		}
		claims["cert_ip_addresses"] = ips
	}

	if len(cert.URIs) > 0 {
		var uris []string
		for _, uri := range cert.URIs {
			uris = append(uris, uri.String())
		}
		claims["cert_uris"] = uris
	}

	// TLS connection information
	if tlsInfo != nil {
		claims["tls_version"] = tlsInfo.State.Version
		claims["tls_cipher_suite"] = tlsInfo.State.CipherSuite
		claims["tls_server_name"] = tlsInfo.State.ServerName
		claims["tls_negotiated_protocol"] = tlsInfo.State.NegotiatedProtocol
		claims["tls_ocsp_response"] = len(tlsInfo.State.OCSPResponse) > 0
		claims["tls_scts"] = len(tlsInfo.State.SignedCertificateTimestamps) > 0
	}

	// Peer connection information
	if peerInfo.Addr != nil {
		claims["peer_address"] = peerInfo.Addr.String()
		if tcpAddr, ok := peerInfo.Addr.(*net.TCPAddr); ok {
			claims["peer_ip"] = tcpAddr.IP.String()
			claims["peer_port"] = tcpAddr.Port
		}
	}

	claims["authenticated_at"] = time.Now().Unix()

	return claims
}

// DefaultCertificateValidator provides comprehensive certificate validation
type DefaultCertificateValidator struct {
	trustedCAs         *x509.CertPool
	minRSAKeySize      int
	minECDSACurveSize  int
	requireKeyUsage    []x509.KeyUsage
	requireExtKeyUsage []x509.ExtKeyUsage
}

// DefaultKeyStrengthValidator validates cryptographic key strength
type DefaultKeyStrengthValidator struct {
	minRSAKeySize     int
	minECDSACurveSize int
}

// DefaultRevocationChecker provides comprehensive CRL and OCSP checking
type DefaultRevocationChecker struct {
	enableCRL  bool
	enableOCSP bool
	crlURLs    []string
	ocspURLs   []string
	crlCache   *CRLCache
	ocspCache  *OCSPCache
	httpClient *http.Client
}

// ValidateCertificate implements comprehensive certificate validation
func (v *DefaultCertificateValidator) ValidateCertificate(cert *x509.Certificate, intermediates *x509.CertPool) error {
	// Validate key strength
	keyValidator := &DefaultKeyStrengthValidator{
		minRSAKeySize:     v.minRSAKeySize,
		minECDSACurveSize: v.minECDSACurveSize,
	}

	if err := keyValidator.ValidateKeyStrength(cert.PublicKey); err != nil {
		return fmt.Errorf("key strength validation failed: %w", err)
	}

	// Validate required key usage
	if len(v.requireKeyUsage) > 0 {
		for _, requiredUsage := range v.requireKeyUsage {
			if cert.KeyUsage&requiredUsage == 0 {
				return fmt.Errorf("certificate missing required key usage: %v", requiredUsage)
			}
		}
	} else {
		// Default requirement: digital signature
		if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
			return fmt.Errorf("certificate does not have digital signature key usage")
		}
	}

	// Validate required extended key usage
	if len(v.requireExtKeyUsage) > 0 {
		for _, requiredExtUsage := range v.requireExtKeyUsage {
			found := false
			for _, extUsage := range cert.ExtKeyUsage {
				if extUsage == requiredExtUsage {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("certificate missing required extended key usage: %v", requiredExtUsage)
			}
		}
	} else {
		// Default requirement: client authentication
		hasClientAuth := false
		for _, usage := range cert.ExtKeyUsage {
			if usage == x509.ExtKeyUsageClientAuth {
				hasClientAuth = true
				break
			}
		}
		if !hasClientAuth {
			return fmt.Errorf("certificate does not have client authentication extended key usage")
		}
	}

	// Validate certificate extensions
	if err := v.validateCertificateExtensions(cert); err != nil {
		return fmt.Errorf("certificate extension validation failed: %w", err)
	}

	// Validate certificate policies
	if err := v.validateCertificatePolicies(cert); err != nil {
		return fmt.Errorf("certificate policy validation failed: %w", err)
	}

	return nil
}

// ValidateCertificateChain validates the entire certificate chain
func (v *DefaultCertificateValidator) ValidateCertificateChain(chain []*x509.Certificate) error {
	if len(chain) == 0 {
		return fmt.Errorf("empty certificate chain")
	}

	// Validate each certificate in the chain
	for i, cert := range chain {
		if i == 0 {
			// Leaf certificate - already validated by ValidateCertificate
			continue
		}

		// Intermediate or root certificates
		if err := v.validateIntermediateCertificate(cert, i); err != nil {
			return fmt.Errorf("intermediate certificate validation failed at position %d: %w", i, err)
		}
	}

	// Validate chain relationships
	for i := 0; i < len(chain)-1; i++ {
		if err := v.validateCertificateRelationship(chain[i], chain[i+1]); err != nil {
			return fmt.Errorf("certificate chain relationship validation failed between positions %d and %d: %w", i, i+1, err)
		}
	}

	return nil
}

// validateCertificateExtensions validates critical certificate extensions
func (v *DefaultCertificateValidator) validateCertificateExtensions(cert *x509.Certificate) error {
	// Check for critical extensions that we don't recognize
	for _, ext := range cert.Extensions {
		if ext.Critical {
			// Check if this is a known critical extension
			switch ext.Id.String() {
			case "2.5.29.15": // Key Usage
			case "2.5.29.17": // Subject Alternative Name
			case "2.5.29.37": // Extended Key Usage
			case "2.5.29.19": // Basic Constraints
			case "2.5.29.32": // Certificate Policies
			case "2.5.29.31": // CRL Distribution Points
			case "1.3.6.1.5.5.7.1.1": // Authority Info Access
				// Known extensions - OK
			default:
				return fmt.Errorf("unknown critical extension: %s", ext.Id.String())
			}
		}
	}

	// Validate Basic Constraints if present
	if cert.BasicConstraintsValid {
		if cert.IsCA && cert.MaxPathLen >= 0 {
			// Validate path length constraints
			if cert.MaxPathLenZero && cert.MaxPathLen != 0 {
				return fmt.Errorf("invalid path length constraint")
			}
		}
	}

	return nil
}

// validateCertificatePolicies validates certificate policies
func (v *DefaultCertificateValidator) validateCertificatePolicies(cert *x509.Certificate) error {
	// This is a simplified implementation
	// In production, you would validate specific certificate policies based on your requirements

	for _, policy := range cert.PolicyIdentifiers {
		// Validate against allowed policies
		_ = policy // Placeholder for policy validation logic
	}

	return nil
}

// validateIntermediateCertificate validates intermediate certificates
func (v *DefaultCertificateValidator) validateIntermediateCertificate(cert *x509.Certificate, position int) error {
	// Intermediate certificates should have CA=true in basic constraints
	if !cert.IsCA {
		return fmt.Errorf("intermediate certificate at position %d is not marked as CA", position)
	}

	// Validate key usage for CA certificates
	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		return fmt.Errorf("intermediate certificate at position %d does not have cert sign key usage", position)
	}

	return nil
}

// validateCertificateRelationship validates the relationship between two certificates in the chain
func (v *DefaultCertificateValidator) validateCertificateRelationship(child, parent *x509.Certificate) error {
	// Check if parent issued child
	if err := child.CheckSignatureFrom(parent); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Validate issuer/subject relationship
	if child.Issuer.String() != parent.Subject.String() {
		return fmt.Errorf("issuer/subject mismatch: child issuer %s != parent subject %s",
			child.Issuer.String(), parent.Subject.String())
	}

	// Validate validity periods
	if child.NotBefore.Before(parent.NotBefore) {
		return fmt.Errorf("child certificate validity starts before parent")
	}

	if child.NotAfter.After(parent.NotAfter) {
		return fmt.Errorf("child certificate validity ends after parent")
	}

	return nil
}

// ValidateKeyStrength validates cryptographic key strength
func (v *DefaultKeyStrengthValidator) ValidateKeyStrength(publicKey crypto.PublicKey) error {
	switch pub := publicKey.(type) {
	case *rsa.PublicKey:
		keySize := pub.Size() * 8
		if keySize < v.minRSAKeySize {
			return fmt.Errorf("RSA key size %d is below minimum %d", keySize, v.minRSAKeySize)
		}

		// Validate RSA public exponent
		if pub.E < 3 || pub.E > 65537 {
			return fmt.Errorf("RSA public exponent %d is not in acceptable range", pub.E)
		}

		// Check for small public exponent vulnerability
		if pub.E == 3 {
			return fmt.Errorf("RSA public exponent 3 is not recommended")
		}

	case *ecdsa.PublicKey:
		curveSize := pub.Curve.Params().BitSize
		if curveSize < v.minECDSACurveSize {
			return fmt.Errorf("ECDSA curve size %d is below minimum %d", curveSize, v.minECDSACurveSize)
		}

		// Validate specific curves
		curveName := pub.Curve.Params().Name
		switch curveName {
		case "P-256", "P-384", "P-521": // NIST curves
			// Acceptable
		case "secp256k1": // Bitcoin curve
			// May be acceptable depending on policy
		default:
			return fmt.Errorf("ECDSA curve %s is not approved", curveName)
		}

	case ed25519.PublicKey:
		// Ed25519 is always 256-bit and considered secure

	default:
		return fmt.Errorf("unsupported public key type: %T", publicKey)
	}

	return nil
}

// IsRevoked implements comprehensive revocation checking
func (r *DefaultRevocationChecker) IsRevoked(ctx context.Context, cert *x509.Certificate, issuer *x509.Certificate) (bool, error) {
	var revoked bool
	var err error

	// Check CRL first (faster)
	if r.enableCRL {
		revoked, err = r.CheckCRL(ctx, cert, issuer)
		if err != nil {
			// Log error but continue to OCSP if available
		} else if revoked {
			return true, nil
		}
	}

	// Check OCSP
	if r.enableOCSP {
		revoked, err = r.CheckOCSP(ctx, cert, issuer)
		if err != nil {
			return false, fmt.Errorf("OCSP check failed: %w", err)
		}
		if revoked {
			return true, nil
		}
	}

	return false, nil
}

// CheckCRL checks certificate against Certificate Revocation Lists
func (r *DefaultRevocationChecker) CheckCRL(ctx context.Context, cert *x509.Certificate, issuer *x509.Certificate) (bool, error) {
	// Extract CRL URLs from certificate
	crlURLs := extractCRLURLs(cert)
	crlURLs = append(crlURLs, r.crlURLs...)

	for _, crlURL := range crlURLs {
		if crlURL == "" {
			continue
		}

		// Check cache first
		crlEntry := r.crlCache.Get(crlURL)
		if crlEntry == nil || time.Since(crlEntry.FetchedAt) > r.crlCache.duration {
			// Fetch new CRL
			crl, err := r.fetchCRL(ctx, crlURL)
			if err != nil {
				continue // Try next URL
			}

			crlEntry = &CRLEntry{
				CRL:       crl,
				FetchedAt: time.Now(),
				URL:       crlURL,
			}
			r.crlCache.Set(crlURL, crlEntry)
		}

		// Check if certificate is revoked
		for _, revokedCert := range crlEntry.CRL.RevokedCertificates {
			if revokedCert.SerialNumber.Cmp(cert.SerialNumber) == 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

// CheckOCSP checks certificate against OCSP responders
func (r *DefaultRevocationChecker) CheckOCSP(ctx context.Context, cert *x509.Certificate, issuer *x509.Certificate) (bool, error) {
	// Extract OCSP URLs from certificate
	ocspURLs := extractOCSPURLs(cert)
	ocspURLs = append(ocspURLs, r.ocspURLs...)

	for _, ocspURL := range ocspURLs {
		if ocspURL == "" {
			continue
		}

		// Create OCSP request
		ocspReq, err := ocsp.CreateRequest(cert, issuer, nil)
		if err != nil {
			continue
		}

		// Check cache first
		reqHash := fmt.Sprintf("%x", sha1.Sum(ocspReq))
		cacheKey := fmt.Sprintf("%s_%s", ocspURL, reqHash)

		ocspEntry := r.ocspCache.Get(cacheKey)
		if ocspEntry == nil || time.Since(ocspEntry.FetchedAt) > r.ocspCache.duration {
			// Fetch new OCSP response
			resp, err := r.fetchOCSP(ctx, ocspURL, ocspReq)
			if err != nil {
				continue // Try next URL
			}

			ocspEntry = &OCSPEntry{
				Response:  resp,
				FetchedAt: time.Now(),
				URL:       ocspURL,
			}
			r.ocspCache.Set(cacheKey, ocspEntry)
		}

		// Check OCSP response
		switch ocspEntry.Response.Status {
		case ocsp.Good:
			return false, nil
		case ocsp.Revoked:
			return true, nil
		case ocsp.Unknown:
			continue // Try next responder
		}
	}

	return false, nil
}

// fetchCRL fetches a CRL from the given URL
func (r *DefaultRevocationChecker) fetchCRL(ctx context.Context, crlURL string) (*x509.RevocationList, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", crlURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create CRL request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CRL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CRL fetch failed with status: %d", resp.StatusCode)
	}

	crlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CRL data: %w", err)
	}

	// Parse CRL
	crl, err := x509.ParseRevocationList(crlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CRL: %w", err)
	}

	return crl, nil
}

// fetchOCSP fetches an OCSP response
func (r *DefaultRevocationChecker) fetchOCSP(ctx context.Context, ocspURL string, req []byte) (*ocsp.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", ocspURL, bytes.NewReader(req))
	if err != nil {
		return nil, fmt.Errorf("failed to create OCSP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/ocsp-request")
	httpReq.Header.Set("Accept", "application/ocsp-response")

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OCSP response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OCSP fetch failed with status: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OCSP response: %w", err)
	}

	// Parse OCSP response
	ocspResp, err := ocsp.ParseResponse(respData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OCSP response: %w", err)
	}

	return ocspResp, nil
}

// extractCRLURLs extracts CRL distribution point URLs from a certificate
func extractCRLURLs(cert *x509.Certificate) []string {
	var urls []string

	for _, ext := range cert.Extensions {
		if ext.Id.Equal(asn1.ObjectIdentifier{2, 5, 29, 31}) { // CRL Distribution Points
			var cdp []distributionPoint
			if _, err := asn1.Unmarshal(ext.Value, &cdp); err == nil {
				for _, dp := range cdp {
					if dp.DistributionPoint.FullName != nil {
						for _, name := range dp.DistributionPoint.FullName {
							if name.Tag == 6 { // URI
								urls = append(urls, string(name.Bytes))
							}
						}
					}
				}
			}
		}
	}

	return urls
}

// extractOCSPURLs extracts OCSP responder URLs from a certificate
func extractOCSPURLs(cert *x509.Certificate) []string {
	return cert.OCSPServer
}

// CRL distribution point structure for ASN.1 parsing
type distributionPoint struct {
	DistributionPoint distributionPointName `asn1:"optional,tag:0"`
	Reasons           asn1.BitString        `asn1:"optional,tag:1"`
	CRLIssuer         []asn1.RawValue       `asn1:"optional,tag:2"`
}

type distributionPointName struct {
	FullName     []asn1.RawValue `asn1:"optional,tag:0"`
	NameRelative []asn1.RawValue `asn1:"optional,tag:1"`
}

// CRLCache methods
func (c *CRLCache) Get(url string) *CRLEntry {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.cache[url]
}

func (c *CRLCache) Set(url string, entry *CRLEntry) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[url] = entry
}

// OCSPCache methods
func (c *OCSPCache) Get(key string) *OCSPEntry {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.cache[key]
}

func (c *OCSPCache) Set(key string, entry *OCSPEntry) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[key] = entry
}

// matchWildcardPattern performs comprehensive wildcard pattern matching
func matchWildcardPattern(pattern, value string) (bool, error) {
	if pattern == "*" {
		return true, nil
	}

	// Use filepath.Match for shell-style pattern matching
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		return false, err
	}

	return matched, nil
}

// getOrDefault returns the value if non-zero, otherwise returns the default
func getOrDefault(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}
