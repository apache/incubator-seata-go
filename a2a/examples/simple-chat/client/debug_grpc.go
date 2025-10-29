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
	"context"
	"fmt"
	"log"

	"seata-go-ai-a2a/pkg/transport/grpc"
	"seata-go-ai-a2a/pkg/types"
)

func main() {
	client, err := grpc.NewClient(&grpc.ClientConfig{Address: "localhost:9090"})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	req := &types.SendMessageRequest{
		Request: &types.Message{
			Role:  types.RoleUser,
			Parts: []types.Part{&types.TextPart{Text: "Hello, can you tell me the time?"}},
		},
		Configuration: &types.SendMessageConfiguration{Blocking: true},
	}

	task, err := client.SendMessage(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task ID: %s\n", task.ID)
	fmt.Printf("Task State: %v\n", task.Status.State)
	fmt.Printf("History entries: %d\n", len(task.History))

	for i, msg := range task.History {
		fmt.Printf("Message %d - Role: %v\n", i, msg.Role)
		for j, part := range msg.Parts {
			if textPart, ok := part.(*types.TextPart); ok {
				fmt.Printf("  Part %d (Text): %s\n", j, textPart.Text)
			}
		}
	}

	fmt.Printf("Artifacts: %d\n", len(task.Artifacts))
	for i, artifact := range task.Artifacts {
		fmt.Printf("Artifact %d - Name: %s\n", i, artifact.Name)
	}
}
