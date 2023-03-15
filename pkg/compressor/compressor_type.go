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

type CompressorType int8

const (
	CompressorNone CompressorType = iota
	CompressorGzip
	CompressorZip
	CompressorSevenz
	CompressorBzip2
	CompressorLz4
	CompressorDeflate
	CompressorZstd
)

func (c CompressorType) String() string {
	switch c {
	case CompressorNone:
		return "CompressorNone"
	case CompressorGzip:
		return "CompressorGzip"
	case CompressorZip:
		return "CompressorZip"
	case CompressorSevenz:
		return "CompressorSevenz"
	case CompressorBzip2:
		return "CompressorBzip2"
	case CompressorLz4:
		return "CompressorLz4"
	case CompressorZstd:
		return "CompressorZstd"
	case CompressorDeflate:
		return "CompressorDeflate"
	default:
		return ""
	}
}

var compressor map[string]CompressorType

func GetByName(name string) CompressorType {
	if compressor == nil {
		compressor = map[string]CompressorType{
			CompressorNone.String():    CompressorNone,
			CompressorGzip.String():    CompressorGzip,
			CompressorZip.String():     CompressorZip,
			CompressorSevenz.String():  CompressorSevenz,
			CompressorBzip2.String():   CompressorBzip2,
			CompressorLz4.String():     CompressorLz4,
			CompressorZstd.String():    CompressorZstd,
			CompressorDeflate.String(): CompressorDeflate,
		}
	}

	if v, ok := compressor[name]; ok {
		return v
	} else {
		return CompressorNone
	}
}
