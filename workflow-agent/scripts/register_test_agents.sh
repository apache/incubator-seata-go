#!/bin/bash
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Script to register test agents to AgentHub for workflow orchestration testing

HUB_URL="${HUB_URL:-http://localhost:8080}"

echo "=== Registering Test Agents to AgentHub ==="
echo "Hub URL: $HUB_URL"
echo ""

# Function to register an agent
register_agent() {
    local name=$1
    local description=$2
    local url=$3
    local skills=$4

    echo "Registering agent: $name"

    response=$(curl -s -X POST "$HUB_URL/api/v1/agents/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"agent_card\": {
                \"name\": \"$name\",
                \"description\": \"$description\",
                \"url\": \"$url\",
                \"version\": \"1.0.0\",
                \"capabilities\": {},
                \"defaultInputModes\": [\"text\"],
                \"defaultOutputModes\": [\"text\"],
                \"skills\": $skills
            },
            \"host\": \"localhost\",
            \"port\": 9001
        }")

    if echo "$response" | grep -q '"success":true'; then
        echo "✓ Successfully registered: $name"
    else
        echo "✗ Failed to register: $name"
        echo "  Response: $response"
    fi
    echo ""
}

# Register Inventory Service Agent
register_agent \
    "inventory-service" \
    "Manages product inventory, reduces stock and handles compensation" \
    "http://localhost:9001" \
    '[
        {
            "id": "inventory-reduce",
            "name": "Reduce Inventory",
            "description": "Reduce inventory quantity for a product",
            "tags": ["inventory", "reduce", "stock", "deduct"],
            "examples": ["reduce inventory", "deduct stock", "decrease quantity"]
        },
        {
            "id": "inventory-compensate",
            "name": "Compensate Inventory Reduction",
            "description": "Restore inventory quantity after failed transaction",
            "tags": ["inventory", "compensate", "restore", "rollback"],
            "examples": ["restore inventory", "rollback stock", "compensate reduction"]
        }
    ]'

# Register Balance Service Agent
register_agent \
    "balance-service" \
    "Manages customer balance, handles deductions and refunds" \
    "http://localhost:9002" \
    '[
        {
            "id": "balance-deduct",
            "name": "Deduct Balance",
            "description": "Deduct amount from customer balance",
            "tags": ["balance", "deduct", "payment", "charge"],
            "examples": ["deduct balance", "charge customer", "payment deduction"]
        },
        {
            "id": "balance-refund",
            "name": "Refund Balance",
            "description": "Refund amount to customer balance after failed transaction",
            "tags": ["balance", "refund", "restore", "rollback"],
            "examples": ["refund balance", "restore amount", "compensate deduction"]
        }
    ]'

# Register Order Service Agent
register_agent \
    "order-service" \
    "Creates and manages orders, handles order cancellation" \
    "http://localhost:9003" \
    '[
        {
            "id": "order-create",
            "name": "Create Order",
            "description": "Create a new order record in the system",
            "tags": ["order", "create", "record", "save"],
            "examples": ["create order", "save order", "record order"]
        },
        {
            "id": "order-cancel",
            "name": "Cancel Order",
            "description": "Cancel an order and mark as failed",
            "tags": ["order", "cancel", "rollback", "fail"],
            "examples": ["cancel order", "rollback order", "mark failed"]
        }
    ]'

echo "=== Agent Registration Complete ==="
echo ""
echo "To verify registered agents, run:"
echo "  curl $HUB_URL/api/v1/agents"
echo ""
echo "To discover agents by skill, run:"
echo "  curl -X POST $HUB_URL/api/v1/agents/discover -H 'Content-Type: application/json' -d '{\"query\": \"inventory\"}'"
