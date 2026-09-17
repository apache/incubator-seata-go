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

package parser

import (
	"reflect"
	"sort"

	aparser "github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	// Import test_driver to register the parser driver required by TiDB parser to instantiate ParamMarkerExpr.
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func DoParser(query string) (*types.ParseContext, error) {
	p := aparser.New()
	stmtNodes, _, err := p.Parse(query, "", "")
	if err != nil {
		return nil, err
	}

	assignParamMarkerOrders(stmtNodes)

	if len(stmtNodes) == 1 {
		return parseParseContext(stmtNodes[0]), err
	}

	parserCtx := types.ParseContext{
		SQLType:      types.SQLTypeMulti,
		ExecutorType: types.MultiExecutor,
		MultiStmt:    make([]*types.ParseContext, 0, len(stmtNodes)),
	}

	for _, node := range stmtNodes {
		parserCtx.MultiStmt = append(parserCtx.MultiStmt, parseParseContext(node))
	}

	return &parserCtx, nil
}

type markerOffset struct {
	marker ast.ParamMarkerExpr
	offset int
}

type paramMarkerOrderVisitor struct {
	markers []markerOffset
}

func (v *paramMarkerOrderVisitor) Enter(node ast.Node) (ast.Node, bool) {
	if marker, ok := node.(ast.ParamMarkerExpr); ok {
		offset := getParamMarkerOffset(node)
		v.markers = append(v.markers, markerOffset{marker: marker, offset: offset})
	}
	return node, false
}

func getParamMarkerOffset(node ast.Node) int {
	if getter, ok := node.(interface{ GetOffset() int }); ok {
		return getter.GetOffset()
	}
	val := reflect.ValueOf(node)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	if val.IsValid() && val.Kind() == reflect.Struct {
		if field := val.FieldByName("Offset"); field.IsValid() && field.CanInt() {
			return int(field.Int())
		}
	}
	return node.OriginTextPosition()
}

func (v *paramMarkerOrderVisitor) Leave(node ast.Node) (ast.Node, bool) {
	return node, true
}

func assignParamMarkerOrders(stmtNodes []ast.StmtNode) {
	visitor := &paramMarkerOrderVisitor{
		markers: make([]markerOffset, 0, len(stmtNodes)*4),
	}
	for _, node := range stmtNodes {
		node.Accept(visitor)
	}

	sort.SliceStable(visitor.markers, func(i, j int) bool {
		return visitor.markers[i].offset < visitor.markers[j].offset
	})
	for i, item := range visitor.markers {
		item.marker.SetOrder(i)
	}
}

func GetParamMarkerOrder(node ast.Node) (int, bool) {
	if marker, ok := node.(ast.ParamMarkerExpr); ok {
		if getter, ok := marker.(interface{ GetOrder() int }); ok {
			return getter.GetOrder(), true
		}
		val := reflect.ValueOf(marker)
		if val.Kind() == reflect.Pointer {
			val = val.Elem()
		}
		if val.IsValid() && val.Kind() == reflect.Struct {
			if field := val.FieldByName("Order"); field.IsValid() && field.CanInt() {
				return int(field.Int()), true
			}
		}
	}
	return 0, false
}

func parseParseContext(stmtNode ast.StmtNode) *types.ParseContext {
	parserCtx := new(types.ParseContext)

	switch stmt := stmtNode.(type) {
	case *ast.InsertStmt:
		parserCtx.SQLType = types.SQLTypeInsert
		parserCtx.InsertStmt = stmt
		parserCtx.ExecutorType = types.InsertExecutor

		if stmt.IsReplace {
			parserCtx.ExecutorType = types.ReplaceIntoExecutor
		}
		if len(stmt.OnDuplicate) != 0 {
			parserCtx.SQLType = types.SQLTypeInsertOnDuplicateUpdate
			parserCtx.ExecutorType = types.InsertOnDuplicateExecutor
		}
	case *ast.UpdateStmt:
		parserCtx.SQLType = types.SQLTypeUpdate
		parserCtx.UpdateStmt = stmt
		parserCtx.ExecutorType = types.UpdateExecutor
	case *ast.SelectStmt:
		if stmt.LockInfo != nil && stmt.LockInfo.LockType == ast.SelectLockForUpdate {
			parserCtx.SQLType = types.SQLTypeSelectForUpdate
			parserCtx.SelectStmt = stmt
			parserCtx.ExecutorType = types.SelectForUpdateExecutor
		} else {
			parserCtx.SQLType = types.SQLTypeSelect
			parserCtx.SelectStmt = stmt
			parserCtx.ExecutorType = types.SelectExecutor
		}
	case *ast.DeleteStmt:
		parserCtx.SQLType = types.SQLTypeDelete
		parserCtx.DeleteStmt = stmt
		parserCtx.ExecutorType = types.DeleteExecutor
	}
	return parserCtx
}
