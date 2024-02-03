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
	"bytes"
	"compress/flate"
	"io"

	"seata.apache.org/seata-go/pkg/util/log"
)

type DeflateCompress struct{}

func (*DeflateCompress) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	fw, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer fw.Close()
	fw.Write(data)
	fw.Flush()
	return buf.Bytes(), nil
}

func (*DeflateCompress) Decompress(data []byte) ([]byte, error) {
	fr := flate.NewReader(bytes.NewBuffer(data))
	defer fr.Close()
	return io.ReadAll(fr)
}

func (*DeflateCompress) GetCompressorType() CompressorType {
	return CompressorDeflate
}
