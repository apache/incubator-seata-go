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

package compressor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZstdCompress(t *testing.T) {
	ts := []struct {
		text string
	}{
		{
			text: strings.Repeat("Don't communicate by sharing memory, share memory by communicating.", 1000),
		},
		{
			text: "88888msj0*&^^%$$#$@!~jjdjfjdlfjkhhdh//><|}{{|\"",
		},
	}

	dc := &Zstd{}
	assert.EqualValues(t, CompressorZstd, dc.GetCompressorType())

	for _, s := range ts {
		var data = []byte(s.text)
		dataCompressed, _ := dc.Compress(data)
		ret, _ := dc.Decompress(dataCompressed)
		assert.EqualValues(t, s.text, string(ret))
	}
}
