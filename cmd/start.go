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

package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	// Version information set by build flags
	Version     string
	Branch      string
	Revision    string
	showVersion bool
	showHelp    bool
)

func init() {
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&showVersion, "v", false, "show version information")
	flag.BoolVar(&showHelp, "help", false, "show help information")
	flag.BoolVar(&showHelp, "h", false, "show help information")
}

func main() {

	hasV := false
	hasVersion := false

	for _, arg := range os.Args[1:] {
		if arg == "-v" {
			hasV = true
		}
		if arg == "-version" || arg == "--version" {
			hasVersion = true
		}
	}

	if hasV && hasVersion {
		fmt.Fprintf(os.Stderr, "Error: cannot use both -v and --version at the same time\n")
		os.Exit(1)
	}

	flag.Parse()
	if showHelp {
		flag.Usage()
		return
	}

	if showVersion {
		versionPrint := Version
		if versionPrint == "" {
			versionPrint = "unknown"
		}
		fmt.Printf("Seata-go version: %s\n", versionPrint)
		if Branch != "" {
			fmt.Printf("Branch: %s\n", Branch)
		}
		if Revision != "" {
			fmt.Printf("Revision: %s\n", Revision)
		}
		return
	}

	// start the server
}
