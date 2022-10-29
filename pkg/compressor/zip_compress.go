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
	"github.com/klauspost/compress/zlib"
	"io"
)

type Zip struct{}

////这里就不强行解释拉
////zip压缩
//func ZipBytes(data []byte) []byte {
//
//	var in bytes.Buffer
//	z:=zlib.NewWriter(&in)
//	z.Write(data)
//	z.Close()
//	return  in.Bytes()
//}
////zip解压
//func UZipBytes(data []byte) []byte  {
//	var out bytes.Buffer
//	var in bytes.Buffer
//	in.Write(data)
//	r,_:=zlib.NewReader(&in)
//	r.Close()
//	io.Copy(&out,r)
//	return out.Bytes()
//}

func (z Zip) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	var zp = zlib.NewWriter(&buf)
	if _, err := zp.Write(data); err != nil {
		return nil, err
	}
	if err := zp.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (z Zip) Decompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	//var in bytes.Buffer
	buf.Write(data)
	r, err := zlib.NewReader(&buf)
	if err != nil {
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (z Zip) GetCompressorType() CompressorType {
	return CompressorZstd
}
