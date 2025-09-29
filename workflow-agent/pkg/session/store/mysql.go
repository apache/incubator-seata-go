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

package store

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"seata-go-ai-workflow-agent/pkg/session/config"
	"seata-go-ai-workflow-agent/pkg/session/types"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

// MySQLStore implements SessionStore using MySQL as backend
type MySQLStore struct {
	db        *sql.DB
	config    config.MySQLConfig
	tableName string
	startTime time.Time
}

// NewMySQLStore creates a new MySQL-based session store
func NewMySQLStore(cfg config.MySQLConfig) (*MySQLStore, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = buildDSN(cfg)
	}

	// Configure TLS if enabled
	if cfg.TLS.Enabled {
		if err := configureTLS(cfg.TLS); err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	tableName := cfg.TableName
	if tableName == "" {
		tableName = "sessions"
	}

	store := &MySQLStore{
		db:        db,
		config:    cfg,
		tableName: tableName,
		startTime: time.Now(),
	}

	// Initialize schema
	if err := store.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// Get retrieves a session by ID
func (m *MySQLStore) Get(ctx context.Context, sessionID string) (*types.SessionData, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, data, created_at, updated_at, expires_at, is_active
		FROM %s 
		WHERE id = ? AND (expires_at IS NULL OR expires_at > NOW())
	`, m.tableName)

	row := m.db.QueryRowContext(ctx, query, sessionID)

	var session types.SessionData
	var dataJSON string
	var expiresAt sql.NullTime

	err := row.Scan(
		&session.ID,
		&session.UserID,
		&dataJSON,
		&session.CreatedAt,
		&session.UpdatedAt,
		&expiresAt,
		&session.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, types.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	if expiresAt.Valid {
		session.ExpiresAt = &expiresAt.Time
	}

	// Unmarshal session data
	var sessionDataContent struct {
		Messages []types.Message        `json:"messages"`
		Context  map[string]interface{} `json:"context"`
		Metadata map[string]string      `json:"metadata"`
	}

	if err := json.Unmarshal([]byte(dataJSON), &sessionDataContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	session.Messages = sessionDataContent.Messages
	session.Context = sessionDataContent.Context
	session.Metadata = sessionDataContent.Metadata
	session.MessageCount = len(session.Messages)

	if session.IsExpired() {
		// Clean up expired session
		m.Delete(ctx, sessionID)
		return nil, types.ErrSessionExpired
	}

	return &session, nil
}

// Set stores or updates a session
func (m *MySQLStore) Set(ctx context.Context, session *types.SessionData) error {
	if session == nil {
		return types.ErrInvalidSession
	}

	session.UpdatedAt = time.Now()

	// Marshal session data
	sessionDataContent := struct {
		Messages []types.Message        `json:"messages"`
		Context  map[string]interface{} `json:"context"`
		Metadata map[string]string      `json:"metadata"`
	}{
		Messages: session.Messages,
		Context:  session.Context,
		Metadata: session.Metadata,
	}

	dataJSON, err := json.Marshal(sessionDataContent)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, user_id, data, created_at, updated_at, expires_at, is_active, message_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			data = VALUES(data),
			updated_at = VALUES(updated_at),
			expires_at = VALUES(expires_at),
			is_active = VALUES(is_active),
			message_count = VALUES(message_count)
	`, m.tableName)

	var expiresAt interface{}
	if session.ExpiresAt != nil {
		expiresAt = *session.ExpiresAt
	}

	_, err = m.db.ExecContext(ctx, query,
		session.ID,
		session.UserID,
		string(dataJSON),
		session.CreatedAt,
		session.UpdatedAt,
		expiresAt,
		session.IsActive,
		session.MessageCount,
	)

	if err != nil {
		return fmt.Errorf("failed to set session: %w", err)
	}

	return nil
}

// Delete removes a session
func (m *MySQLStore) Delete(ctx context.Context, sessionID string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", m.tableName)

	result, err := m.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return types.ErrSessionNotFound
	}

	return nil
}

// Exists checks if a session exists
func (m *MySQLStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT 1 FROM %s 
		WHERE id = ? AND (expires_at IS NULL OR expires_at > NOW())
		LIMIT 1
	`, m.tableName)

	row := m.db.QueryRowContext(ctx, query, sessionID)

	var exists int
	err := row.Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return true, nil
}

// List returns session IDs for a user with pagination
func (m *MySQLStore) List(ctx context.Context, userID string, offset, limit int) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT id FROM %s 
		WHERE user_id = ? AND (expires_at IS NULL OR expires_at > NOW()) AND is_active = 1
		ORDER BY updated_at DESC
		LIMIT ? OFFSET ?
	`, m.tableName)

	rows, err := m.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessionIDs []string
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return nil, fmt.Errorf("failed to scan session ID: %w", err)
		}
		sessionIDs = append(sessionIDs, sessionID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return sessionIDs, nil
}

// Count returns the total number of sessions for a user
func (m *MySQLStore) Count(ctx context.Context, userID string) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s 
		WHERE user_id = ? AND (expires_at IS NULL OR expires_at > NOW()) AND is_active = 1
	`, m.tableName)

	row := m.db.QueryRowContext(ctx, query, userID)

	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	return count, nil
}

// Cleanup removes expired sessions
func (m *MySQLStore) Cleanup(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE expires_at IS NOT NULL AND expires_at <= NOW()
	`, m.tableName)

	result, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// Close closes the store and releases resources
func (m *MySQLStore) Close() error {
	return m.db.Close()
}

// Health checks store connectivity
func (m *MySQLStore) Health(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

// GetStats returns store statistics
func (m *MySQLStore) GetStats(ctx context.Context) (types.StoreStats, error) {
	var stats types.StoreStats

	// Get total sessions count
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", m.tableName)
	row := m.db.QueryRowContext(ctx, query)
	if err := row.Scan(&stats.TotalSessions); err != nil {
		return stats, fmt.Errorf("failed to get total sessions: %w", err)
	}

	// Get active sessions count
	query = fmt.Sprintf(`
		SELECT COUNT(*) FROM %s 
		WHERE (expires_at IS NULL OR expires_at > NOW()) AND is_active = 1
	`, m.tableName)
	row = m.db.QueryRowContext(ctx, query)
	if err := row.Scan(&stats.ActiveSessions); err != nil {
		return stats, fmt.Errorf("failed to get active sessions: %w", err)
	}

	stats.ExpiredSessions = stats.TotalSessions - stats.ActiveSessions

	// Get average messages
	query = fmt.Sprintf(`
		SELECT AVG(message_count) FROM %s 
		WHERE (expires_at IS NULL OR expires_at > NOW()) AND is_active = 1
	`, m.tableName)
	row = m.db.QueryRowContext(ctx, query)
	var avgMessages sql.NullFloat64
	if err := row.Scan(&avgMessages); err != nil {
		return stats, fmt.Errorf("failed to get average messages: %w", err)
	}
	if avgMessages.Valid {
		stats.AverageMessages = avgMessages.Float64
	}

	// Get oldest and newest sessions
	query = fmt.Sprintf(`
		SELECT MIN(created_at), MAX(created_at) FROM %s 
		WHERE (expires_at IS NULL OR expires_at > NOW()) AND is_active = 1
	`, m.tableName)
	row = m.db.QueryRowContext(ctx, query)
	var oldest, newest sql.NullTime
	if err := row.Scan(&oldest, &newest); err != nil {
		return stats, fmt.Errorf("failed to get session timestamps: %w", err)
	}
	if oldest.Valid {
		stats.OldestSession = &oldest.Time
	}
	if newest.Valid {
		stats.NewestSession = &newest.Time
	}

	return stats, nil
}

// initSchema creates the sessions table if it doesn't exist
func (m *MySQLStore) initSchema(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			data JSON NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NULL,
			is_active BOOLEAN NOT NULL DEFAULT 1,
			message_count INT NOT NULL DEFAULT 0,
			INDEX idx_user_id (user_id),
			INDEX idx_user_updated (user_id, updated_at),
			INDEX idx_expires_at (expires_at),
			INDEX idx_active (is_active)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, m.tableName)

	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	return nil
}

// buildDSN constructs the MySQL DSN from config
func buildDSN(cfg config.MySQLConfig) string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	params := []string{
		"parseTime=true",
		"charset=utf8mb4",
		"collation=utf8mb4_unicode_ci",
	}

	if cfg.TLS.Enabled {
		params = append(params, "tls=custom")
	}

	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}

	return dsn
}

// configureTLS configures TLS for MySQL connection
func configureTLS(cfg config.TLSConfig) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.CAFile != "" {
		caCert, err := ioutil.ReadFile(cfg.CAFile)
		if err != nil {
			return fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return mysql.RegisterTLSConfig("custom", tlsConfig)
}
