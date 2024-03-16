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

package flagext

import (
	"flag"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"seata.apache.org/seata-go/pkg/util/log"
)

// DeprecatedFlagsUsed is the metric that counts deprecated flags set.
var DeprecatedFlagsUsed = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "deprecated_flags_inuse_total",
		Help:      "The number of deprecated flags currently set.",
	})

type deprecatedFlag struct {
	name string
}

func (deprecatedFlag) String() string {
	return "deprecated"
}

func (d deprecatedFlag) Set(string) error {
	log.Infof("msg", "flag disabled", "flag", d.name)
	DeprecatedFlagsUsed.Inc()
	return nil
}

// DeprecatedFlag logs a warning when you try to use it.
func DeprecatedFlag(f *flag.FlagSet, name, message string) {
	f.Var(deprecatedFlag{name}, name, message)
}
