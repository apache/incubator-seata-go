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

package utils

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

// TokenCounter provides token counting functionality
type TokenCounter struct {
	encoder *tiktoken.Tiktoken
}

// NewTokenCounter creates a new token counter
func NewTokenCounter(encoding string) (*TokenCounter, error) {
	if encoding == "" {
		encoding = "cl100k_base" // Default encoding for GPT-4
	}

	encoder, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to get tiktoken encoding %s: %w", encoding, err)
	}

	return &TokenCounter{
		encoder: encoder,
	}, nil
}

// CountTokens counts the number of tokens in the given text
func (tc *TokenCounter) CountTokens(text string) int {
	if tc.encoder == nil {
		// Fallback: approximate token count as words/4 * 3
		words := len(text) / 4
		return words * 3
	}

	tokens := tc.encoder.Encode(text, nil, nil)
	return len(tokens)
}

// EstimateTokens provides a rough estimation of token count without tokenizer
func EstimateTokens(text string) int {
	// Very rough estimation: ~4 characters per token on average
	return len(text) / 4
}
