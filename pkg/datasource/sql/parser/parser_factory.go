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
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	aparser "github.com/arana-db/parser"
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/mysql"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func DoParser(query string) (*types.ParseContext, error) {
	return DoParserForDB(query, types.DBTypeMySQL)
}

// DoParserForDB parses a statement and retains quoted table identifiers for
// dialect-specific metadata resolution.
func DoParserForDB(query string, dbType types.DBType) (*types.ParseContext, error) {
	ctx, err := ParseSQLForDB(query, dbType)
	if err != nil {
		return nil, err
	}
	if err := BindTableRefs(ctx); err != nil {
		return nil, err
	}
	return ctx, nil
}

// ParseSQLForDB parses and classifies SQL without binding table identifiers.
// Metadata consumers must call BindTableRefs before resolving table identities.
func ParseSQLForDB(query string, dbType types.DBType) (*types.ParseContext, error) {
	p := aparser.New()
	if dbType == types.DBTypePostgreSQL {
		p.SetSQLMode(mysql.ModeANSIQuotes)
	}
	stmtNodes, _, err := p.Parse(query, "", "")
	if err != nil {
		return nil, err
	}

	if len(stmtNodes) == 1 {
		return parseParseContext(stmtNodes[0], dbType), nil
	}

	parserCtx := types.ParseContext{
		DBType:       dbType,
		SQLType:      types.SQLTypeMulti,
		ExecutorType: types.MultiExecutor,
		MultiStmt:    make([]*types.ParseContext, 0, len(stmtNodes)),
	}

	for _, node := range stmtNodes {
		parserCtx.MultiStmt = append(parserCtx.MultiStmt, parseParseContext(node, dbType))
	}

	return &parserCtx, nil
}

// BindTableRefs retains quoted identifiers from each statement's original SQL.
// Complete existing bindings are validated and preserved after CopyTableRefs.
func BindTableRefs(ctx *types.ParseContext) error {
	if ctx == nil {
		return fmt.Errorf("cannot bind table references: nil parse context")
	}
	for i, child := range ctx.MultiStmt {
		if err := BindTableRefs(child); err != nil {
			return fmt.Errorf("cannot bind statement %d: %w", i+1, err)
		}
	}

	tables := tableNames(ctx)
	if ctx.TableRefs != nil {
		if len(ctx.TableRefs) != len(tables) {
			return fmt.Errorf("cannot bind table references: found %d bound references for %d AST tables", len(ctx.TableRefs), len(tables))
		}
		for i, table := range tables {
			ref, ok := ctx.TableRefs[table]
			if !ok || ref.Qualifier != table.Schema.O || ref.TableName != table.Name.O {
				return fmt.Errorf("cannot bind table reference %d: existing binding does not match AST table %q.%q", i+1, table.Schema.O, table.Name.O)
			}
		}
		return nil
	}
	if len(tables) == 0 {
		return nil
	}

	var stmt ast.StmtNode
	switch {
	case ctx.InsertStmt != nil:
		stmt = ctx.InsertStmt
	case ctx.UpdateStmt != nil:
		stmt = ctx.UpdateStmt
	case ctx.DeleteStmt != nil:
		stmt = ctx.DeleteStmt
	case ctx.SelectStmt != nil:
		stmt = ctx.SelectStmt
	}
	return captureTableRefs(ctx, stmt.Text())
}

type identifierTokenKind uint8

const (
	identifierWord identifierTokenKind = iota
	identifierString
	identifierSymbol
)

type identifierToken struct {
	value  string
	quoted bool
	kind   identifierTokenKind
	start  int
	end    int
}

func (t identifierToken) keyword(word string) bool {
	return t.kind == identifierWord && !t.quoted && strings.EqualFold(t.value, word)
}

func (t identifierToken) symbol(value string) bool {
	return t.kind == identifierSymbol && t.value == value
}

// scanIdentifiers keeps the spelling and quoting that the AST's CIStr drops.
func scanIdentifiers(sql string, dbType types.DBType) []identifierToken {
	var tokens []identifierToken
	mysqlComments := dbType == 0 || dbType == types.DBTypeUnknown || dbType == types.DBTypeMySQL
	inExecutableComment := false
	for i := 0; i < len(sql); {
		if inExecutableComment && strings.HasPrefix(sql[i:], "*/") {
			i += 2
			inExecutableComment = false
			continue
		}
		if strings.HasPrefix(sql[i:], "/*") {
			if mysqlComments && strings.HasPrefix(sql[i:], "/*!") {
				i += 3
				// arana recognizes these comments regardless of version and
				// consumes a version prefix only when all five digits are present.
				versionEnd := i
				for versionEnd < len(sql) && versionEnd < i+5 && sql[versionEnd] >= '0' && sql[versionEnd] <= '9' {
					versionEnd++
				}
				if versionEnd == i+5 {
					i = versionEnd
				}
				inExecutableComment = true
				continue
			}
			if end := strings.Index(sql[i+2:], "*/"); end >= 0 {
				i += end + 4
			} else {
				break
			}
			continue
		}
		if sql[i] == '#' || (strings.HasPrefix(sql[i:], "--") && i+2 < len(sql) && sql[i+2] <= ' ') {
			if end := strings.IndexByte(sql[i:], '\n'); end >= 0 {
				i += end + 1
			} else {
				break
			}
			continue
		}
		if sql[i] == '\'' || sql[i] == '`' || sql[i] == '"' {
			start := i
			quote := sql[i]
			i++
			var value strings.Builder
			for i < len(sql) {
				if sql[i] == quote {
					if i+1 < len(sql) && sql[i+1] == quote {
						value.WriteByte(quote)
						i += 2
						continue
					}
					i++
					break
				}
				if sql[i] == '\\' && i+1 < len(sql) {
					i++
					value.WriteByte(sql[i])
					i++
					continue
				}
				value.WriteByte(sql[i])
				i++
			}
			if quote == '\'' {
				tokens = append(tokens, identifierToken{value: value.String(), kind: identifierString, start: start, end: i})
			} else {
				tokens = append(tokens, identifierToken{value: value.String(), quoted: true, kind: identifierWord, start: start, end: i})
			}
			continue
		}
		r, size := utf8.DecodeRuneInString(sql[i:])
		if r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			start := i
			i += size
			for i < len(sql) {
				r, size = utf8.DecodeRuneInString(sql[i:])
				if r != '_' && r != '$' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					break
				}
				i += size
			}
			tokens = append(tokens, identifierToken{value: sql[start:i], kind: identifierWord, start: start, end: i})
			continue
		}
		if !unicode.IsSpace(r) {
			tokens = append(tokens, identifierToken{value: sql[i : i+size], kind: identifierSymbol, start: i, end: i + size})
		}
		i += size
	}
	return tokens
}

func captureTableRefs(ctx *types.ParseContext, sql string) error {
	tables := tableNames(ctx)
	if len(tables) == 0 {
		return nil
	}
	tokens := scanIdentifiers(sql, ctx.DBType)
	region, err := tableReferenceRegion(ctx, tokens)
	if err != nil {
		return err
	}
	refs, err := collectTableReferences(region)
	if err != nil {
		return err
	}
	if len(refs) != len(tables) {
		return fmt.Errorf("cannot bind table references: found %d SQL references for %d AST tables", len(refs), len(tables))
	}
	bound := make(map[*ast.TableName]types.TableRef, len(tables))
	for i, table := range tables {
		ref := refs[i]
		if ref.Qualifier != table.Schema.O || ref.TableName != table.Name.O {
			return fmt.Errorf("cannot bind table reference %d: SQL table %q.%q does not match AST table %q.%q", i+1, ref.Qualifier, ref.TableName, table.Schema.O, table.Name.O)
		}
		bound[table] = ref
	}
	ctx.TableRefs = bound
	return nil
}

// topLevelKeyword excludes nested queries, function arguments and quoted names.
func topLevelKeyword(tokens []identifierToken, words ...string) int {
	depth := 0
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		switch {
		case token.symbol("(") || token.symbol("{"):
			depth++
		case token.symbol(")") || token.symbol("}"):
			depth--
		case depth == 0:
			if token.symbol(";") {
				return -1
			}
			if i > 0 && tokens[i-1].symbol(".") {
				continue
			}
			if end := indexHintEnd(tokens, i); end >= 0 {
				i = end
				continue
			}
			for _, word := range words {
				if token.keyword(word) {
					return i
				}
			}
		}
	}
	return -1
}

func tableReferenceRegion(ctx *types.ParseContext, tokens []identifierToken) ([]identifierToken, error) {
	for len(tokens) > 0 && tokens[len(tokens)-1].symbol(";") {
		tokens = tokens[:len(tokens)-1]
	}
	for len(tokens) > 1 && tokens[0].symbol("(") && closingDelimiter(tokens, 0) == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	start := -1
	endWords := []string{"WHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "FOR", "LOCK", "WINDOW", "UNION", "EXCEPT", "INTERSECT"}
	switch {
	case ctx.InsertStmt != nil:
		start = topLevelKeyword(tokens, "INSERT", "REPLACE")
	case ctx.UpdateStmt != nil:
		start = topLevelKeyword(tokens, "UPDATE")
		endWords = []string{"SET"}
	case ctx.DeleteStmt != nil:
		keyword := "FROM"
		if ctx.DeleteStmt.IsMultiTable && !ctx.DeleteStmt.BeforeFrom {
			keyword = "USING"
		}
		start = topLevelKeyword(tokens, keyword)
	case ctx.SelectStmt != nil:
		keyword := "FROM"
		if ctx.SelectStmt.Kind == ast.SelectStmtKindTable {
			keyword = "TABLE"
		}
		start = topLevelKeyword(tokens, keyword)
	}
	if start < 0 {
		return nil, fmt.Errorf("cannot locate table reference region")
	}
	tokens = tokens[start+1:]
	if ctx.InsertStmt != nil || ctx.UpdateStmt != nil {
		for len(tokens) > 0 && (tokens[0].keyword("LOW_PRIORITY") || tokens[0].keyword("HIGH_PRIORITY") || tokens[0].keyword("DELAYED") || tokens[0].keyword("IGNORE") || tokens[0].keyword("INTO")) {
			tokens = tokens[1:]
		}
	}
	if ctx.InsertStmt != nil {
		_, count, err := readTableReference(tokens)
		return tokens[:count], err
	}
	if end := topLevelKeyword(tokens, endWords...); end >= 0 {
		tokens = tokens[:end]
	}
	return tokens, nil
}

func closingDelimiter(tokens []identifierToken, start int) int {
	opening, closing := "(", ")"
	if tokens[start].symbol("{") {
		opening, closing = "{", "}"
	}
	depth := 0
	for i := start; i < len(tokens); i++ {
		if tokens[i].symbol(opening) {
			depth++
		} else if tokens[i].symbol(closing) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// Index hints contain JOIN / ORDER / GROUP keywords without starting a new clause.
func indexHintEnd(tokens []identifierToken, start int) int {
	if start+2 >= len(tokens) || !(tokens[start].keyword("USE") || tokens[start].keyword("FORCE") || tokens[start].keyword("IGNORE")) || !(tokens[start+1].keyword("INDEX") || tokens[start+1].keyword("KEY")) {
		return -1
	}
	i := start + 2
	if tokens[i].keyword("FOR") {
		i++
		if i < len(tokens) && tokens[i].keyword("JOIN") {
			i++
		} else if i+1 < len(tokens) && (tokens[i].keyword("ORDER") || tokens[i].keyword("GROUP")) && tokens[i+1].keyword("BY") {
			i += 2
		} else {
			return -1
		}
	}
	if i < len(tokens) && tokens[i].symbol("(") {
		return closingDelimiter(tokens, i)
	}
	return -1
}

func readTableReference(tokens []identifierToken) (types.TableRef, int, error) {
	if len(tokens) == 0 || tokens[0].kind != identifierWord {
		return types.TableRef{}, 0, fmt.Errorf("cannot bind table reference: expected an identifier")
	}
	ref := types.TableRef{TableName: tokens[0].value, TableNameQuoted: tokens[0].quoted}
	if len(tokens) > 1 && tokens[1].symbol(".") {
		if len(tokens) < 3 || tokens[2].kind != identifierWord {
			return types.TableRef{}, 0, fmt.Errorf("cannot bind qualified table reference at byte %d", tokens[0].start)
		}
		ref.Qualifier, ref.QualifierQuoted = ref.TableName, ref.TableNameQuoted
		ref.TableName, ref.TableNameQuoted = tokens[2].value, tokens[2].quoted
		return ref, 3, nil
	}
	return ref, 1, nil
}

// collectTableReferences only interprets commas within a table-reference region.
// Parenthesized and ODBC table groups share that region; derived queries and expressions do not.
func collectTableReferences(tokens []identifierToken) ([]types.TableRef, error) {
	var refs []types.TableRef
	expectTable := true
	for i := 0; i < len(tokens); {
		token := tokens[i]
		if token.symbol(";") {
			break
		}
		if token.symbol(")") || token.symbol("}") {
			return nil, fmt.Errorf("cannot bind table reference: unexpected %s at byte %d", token.value, token.start)
		}
		if token.symbol("(") || token.symbol("{") {
			end := closingDelimiter(tokens, i)
			if end < 0 {
				return nil, fmt.Errorf("cannot bind table reference: unmatched %s at byte %d", token.value, token.start)
			}
			inner := tokens[i+1 : end]
			if expectTable && token.symbol("{") {
				if len(inner) < 2 || !inner[0].keyword("OJ") {
					return nil, fmt.Errorf("cannot bind table reference: expected ODBC OJ group at byte %d", token.start)
				}
				inner = inner[1:]
			}
			if expectTable && topLevelKeyword(inner, "SELECT", "WITH", "TABLE") < 0 {
				group, err := collectTableReferences(inner)
				if err != nil {
					return nil, err
				}
				refs = append(refs, group...)
			}
			expectTable = false
			i = end + 1
			continue
		}
		if expectTable {
			ref, count, err := readTableReference(tokens[i:])
			if err != nil {
				return nil, err
			}
			refs = append(refs, ref)
			i += count
			expectTable = false
			continue
		}
		if end := indexHintEnd(tokens, i); end >= 0 {
			i = end + 1
			continue
		}
		if token.symbol(",") || ((token.keyword("JOIN") || token.keyword("STRAIGHT_JOIN")) && (i == 0 || !tokens[i-1].symbol("."))) {
			expectTable = true
		}
		i++
	}
	return refs, nil
}

// CopyTableRefs keeps the original identifier quoting when a statement is
// restored and reparsed to rebase its parameter markers.
func CopyTableRefs(dst, src *types.ParseContext) {
	dstTables := tableNames(dst)
	srcTables := tableNames(src)
	if len(dstTables) != len(srcTables) {
		return
	}
	dst.TableRefs = make(map[*ast.TableName]types.TableRef, len(dstTables))
	for i, table := range dstTables {
		if ref, ok := src.TableRefs[srcTables[i]]; ok {
			dst.TableRefs[table] = ref
		}
	}
}

func tableNames(ctx *types.ParseContext) []*ast.TableName {
	var clause *ast.TableRefsClause
	switch {
	case ctx.InsertStmt != nil:
		clause = ctx.InsertStmt.Table
	case ctx.UpdateStmt != nil:
		clause = ctx.UpdateStmt.TableRefs
	case ctx.DeleteStmt != nil:
		clause = ctx.DeleteStmt.TableRefs
	case ctx.SelectStmt != nil:
		clause = ctx.SelectStmt.From
	}
	if clause == nil || clause.TableRefs == nil {
		return nil
	}
	var tables []*ast.TableName
	var visit func(ast.ResultSetNode)
	visit = func(source ast.ResultSetNode) {
		switch node := source.(type) {
		case *ast.Join:
			visit(node.Left)
			if node.Right != nil {
				visit(node.Right)
			}
		case *ast.TableSource:
			visit(node.Source)
		case *ast.TableName:
			tables = append(tables, node)
		}
	}
	visit(clause.TableRefs)
	return tables
}

func parseParseContext(stmtNode ast.StmtNode, dbType types.DBType) *types.ParseContext {
	parserCtx := &types.ParseContext{DBType: dbType}

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
