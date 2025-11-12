package generator

import (
	"fmt"
	"strings"

	pg_query_go "github.com/pganalyze/pg_query_go/v6"
	pg_query "github.com/wasilibs/go-pgquery"
)

// SQLParser provides SQL analysis using PostgreSQL's parser
type SQLParser struct{}

// NewSQLParser creates a new SQL parser
func NewSQLParser() *SQLParser {
	return &SQLParser{}
}

// QueryInfo contains extracted metadata from SQL query
type QueryInfo struct {
	Type          QueryType
	Parameters    []ParameterInfo
	SelectTargets []SelectTarget
	Tables        []TableRef
	CTEs          []CTEInfo
	Joins         []JoinInfo
}

// ParameterInfo contains parameter metadata from parse tree
type ParameterInfo struct {
	Position   int    // 1-based ($1, $2, etc.)
	ColumnName string // Column name if detected
	TableName  string // Table name if detected
	Operator   string // "=", ">", "IN", "LIKE", etc.
	IsInWhere  bool   // Found in WHERE clause
	IsInSet    bool   // Found in UPDATE SET clause
	IsInLimit  bool   // Found in LIMIT clause
	IsInOffset bool   // Found in OFFSET clause
}

// SelectTarget represents a column in SELECT clause
type SelectTarget struct {
	Alias             string // Column alias or name
	IsCount           bool   // COUNT aggregate
	IsSum             bool   // SUM aggregate
	IsAvg             bool   // AVG aggregate
	IsMax             bool   // MAX aggregate
	IsMin             bool   // MIN aggregate
	IsCoalesce        bool   // COALESCE function
	HasNonNullLiteral bool   // COALESCE/CASE has non-null literal (guarantees non-null)
	IsCaseWithElse    bool   // CASE expression with ELSE clause
	IsRowNumber       bool   // ROW_NUMBER() window function
	IsRank            bool   // RANK() window function
	IsDenseRank       bool   // DENSE_RANK() window function
	Expression        string // Full expression as string
}

// TableRef represents a table reference in query
type TableRef struct {
	Name         string     // Table name (empty for subqueries)
	Alias        string     // Table alias (if any)
	Schema       string     // Schema name (if specified)
	IsSubquery   bool       // True if this is a subquery in FROM
	SubqueryInfo *QueryInfo // Parsed subquery (if IsSubquery is true)
}

// CTEInfo represents a Common Table Expression
type CTEInfo struct {
	Name  string     // CTE name
	Query *QueryInfo // Recursively parsed CTE query
}

// JoinType represents the type of JOIN operation
type JoinType int

const (
	JoinTypeInner JoinType = iota
	JoinTypeLeft
	JoinTypeRight
	JoinTypeFull
)

// JoinInfo represents a JOIN operation in the query
type JoinInfo struct {
	Type       JoinType // Type of join (INNER, LEFT, RIGHT, FULL)
	LeftTable  string   // Left table name or alias
	RightTable string   // Right table name or alias
}

// Parse analyzes SQL and returns structured metadata
func (sp *SQLParser) Parse(sql string) (*QueryInfo, error) {
	result, err := pg_query.Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SQL: %w", err)
	}

	return sp.extractInfo(result)
}

// extractInfo extracts metadata from parse result
func (sp *SQLParser) extractInfo(result *pg_query_go.ParseResult) (*QueryInfo, error) {
	if len(result.Stmts) == 0 {
		return nil, fmt.Errorf("no statements found in SQL")
	}

	stmt := result.Stmts[0].Stmt
	if stmt == nil {
		return nil, fmt.Errorf("statement is nil")
	}

	info := &QueryInfo{}

	info.Type = sp.determineQueryType(stmt)
	info.Tables = sp.extractTables(stmt)
	info.Parameters = sp.extractParameters(stmt)

	if selectStmt := stmt.GetSelectStmt(); selectStmt != nil {
		info.SelectTargets = sp.extractSelectTargets(selectStmt)
		info.Joins = sp.extractJoins(selectStmt)
		info.CTEs = sp.extractCTEs(selectStmt)
	}

	return info, nil
}

// determineQueryType determines QueryType from parse tree node
func (sp *SQLParser) determineQueryType(stmt *pg_query_go.Node) QueryType {
	if stmt.GetSelectStmt() != nil {
		return QueryTypeMany
	}
	if stmt.GetInsertStmt() != nil {
		return QueryTypeExec
	}
	if stmt.GetUpdateStmt() != nil {
		return QueryTypeExec
	}
	if stmt.GetDeleteStmt() != nil {
		return QueryTypeExec
	}
	return QueryTypeMany
}

// extractJoins extracts JOIN information from SELECT statement
func (sp *SQLParser) extractJoins(selectStmt *pg_query_go.SelectStmt) []JoinInfo {
	var joins []JoinInfo

	if selectStmt.FromClause == nil {
		return joins
	}

	for _, fromNode := range selectStmt.FromClause {
		sp.extractJoinsFromNode(fromNode, &joins)
	}

	return joins
}

// extractJoinsFromNode recursively extracts JOIN info from a node
func (sp *SQLParser) extractJoinsFromNode(node *pg_query_go.Node, joins *[]JoinInfo) {
	joinExpr := node.GetJoinExpr()
	if joinExpr == nil {
		return
	}

	// Recursively process nested joins first (to maintain SQL order)
	sp.extractJoinsFromNode(joinExpr.Larg, joins)
	sp.extractJoinsFromNode(joinExpr.Rarg, joins)

	// Map PostgreSQL join type enum to our JoinType
	var joinType JoinType
	switch joinExpr.Jointype {
	case 1: // JOIN_INNER
		joinType = JoinTypeInner
	case 2: // JOIN_LEFT
		joinType = JoinTypeLeft
	case 3: // JOIN_FULL
		joinType = JoinTypeFull
	case 4: // JOIN_RIGHT
		joinType = JoinTypeRight
	default:
		joinType = JoinTypeInner
	}

	// Extract left and right table names/aliases
	leftTable := sp.extractTableIdentifier(joinExpr.Larg)
	rightTable := sp.extractTableIdentifier(joinExpr.Rarg)

	*joins = append(*joins, JoinInfo{
		Type:       joinType,
		LeftTable:  leftTable,
		RightTable: rightTable,
	})
}

// extractTableIdentifier extracts table name or alias from a range node
func (sp *SQLParser) extractTableIdentifier(node *pg_query_go.Node) string {
	if rangeVar := node.GetRangeVar(); rangeVar != nil {
		// Prefer alias if available, otherwise use table name
		if rangeVar.Alias != nil && rangeVar.Alias.Aliasname != "" {
			return rangeVar.Alias.Aliasname
		}
		return rangeVar.Relname
	}

	// Check for subquery
	if rangeSubselect := node.GetRangeSubselect(); rangeSubselect != nil {
		if rangeSubselect.Alias != nil {
			return rangeSubselect.Alias.Aliasname
		}
	}

	return ""
}

// extractSelectTargets extracts columns from SELECT clause
func (sp *SQLParser) extractSelectTargets(selectStmt *pg_query_go.SelectStmt) []SelectTarget {
	var targets []SelectTarget

	for _, node := range selectStmt.TargetList {
		target := sp.analyzeTargetNode(node)
		if target != nil {
			targets = append(targets, *target)
		}
	}

	return targets
}

// analyzeTargetNode analyzes a single target node
func (sp *SQLParser) analyzeTargetNode(node *pg_query_go.Node) *SelectTarget {
	resTarget := node.GetResTarget()
	if resTarget == nil {
		return nil
	}

	target := &SelectTarget{
		Alias: resTarget.Name,
	}

	// Check if it's a function call (aggregate or special function)
	if funcCall := resTarget.Val.GetFuncCall(); funcCall != nil {
		sp.analyzeFuncCall(funcCall, target)
	}

	// Check if it's a COALESCE expression
	if coalesceExpr := resTarget.Val.GetCoalesceExpr(); coalesceExpr != nil {
		sp.analyzeCoalesceExpr(coalesceExpr, target)
	}

	// Check if it's a CASE expression
	if caseExpr := resTarget.Val.GetCaseExpr(); caseExpr != nil {
		sp.analyzeCaseExpr(caseExpr, target)
	}

	// If no alias, try to extract column name
	if target.Alias == "" {
		target.Alias = sp.extractColumnNameFromNode(resTarget.Val)
	}

	return target
}

// analyzeFuncCall detects aggregate and special functions
func (sp *SQLParser) analyzeFuncCall(funcCall *pg_query_go.FuncCall, target *SelectTarget) {
	if len(funcCall.Funcname) == 0 {
		return
	}

	lastFunc := funcCall.Funcname[len(funcCall.Funcname)-1]
	if lastFunc == nil {
		return
	}

	strNode := lastFunc.GetString_()
	if strNode == nil {
		return
	}

	funcName := strNode.Sval

	switch strings.ToLower(funcName) {
	case "count":
		target.IsCount = true
	case "sum":
		target.IsSum = true
	case "avg":
		target.IsAvg = true
	case "max":
		target.IsMax = true
	case "min":
		target.IsMin = true
	case "coalesce":
		target.IsCoalesce = true
		target.HasNonNullLiteral = sp.hasNonNullLiteralArg(funcCall.Args)
	case "row_number":
		target.IsRowNumber = true
	case "rank":
		target.IsRank = true
	case "dense_rank":
		target.IsDenseRank = true
	}
}

// analyzeCoalesceExpr detects COALESCE expressions
func (sp *SQLParser) analyzeCoalesceExpr(coalesceExpr *pg_query_go.CoalesceExpr, target *SelectTarget) {
	target.IsCoalesce = true
	// Check if any argument is a non-null literal
	target.HasNonNullLiteral = sp.hasNonNullLiteralArg(coalesceExpr.Args)
}

// analyzeCaseExpr detects CASE expressions and checks for ELSE clause
func (sp *SQLParser) analyzeCaseExpr(caseExpr *pg_query_go.CaseExpr, target *SelectTarget) {
	// Check if there's an ELSE clause (defresult)
	if caseExpr.Defresult != nil {
		target.IsCaseWithElse = true
		// Check if the ELSE result is a non-null literal
		target.HasNonNullLiteral = sp.isNonNullLiteral(caseExpr.Defresult)
	}
}

// hasNonNullLiteralArg checks if function args contain a non-null literal
func (sp *SQLParser) hasNonNullLiteralArg(args []*pg_query_go.Node) bool {
	for _, arg := range args {
		if sp.isNonNullLiteral(arg) {
			return true
		}
	}
	return false
}

// isNonNullLiteral checks if a node is a non-null literal value
func (sp *SQLParser) isNonNullLiteral(node *pg_query_go.Node) bool {
	if node == nil {
		return false
	}

	// Check for A_Const (constant value)
	if aConst := node.GetAConst(); aConst != nil {
		// If it has any value (string, int, float, bool), it's non-null
		if aConst.GetIsnull() {
			return false
		}
		return aConst.GetSval() != nil || aConst.GetIval() != nil ||
			aConst.GetFval() != nil || aConst.GetBoolval() != nil
	}

	return false
}

// extractColumnNameFromNode attempts to extract column name from expression
func (sp *SQLParser) extractColumnNameFromNode(node *pg_query_go.Node) string {
	if colRef := node.GetColumnRef(); colRef != nil {
		if len(colRef.Fields) == 0 {
			return ""
		}
		lastField := colRef.Fields[len(colRef.Fields)-1]
		if lastField == nil {
			return ""
		}
		if str := lastField.GetString_(); str != nil {
			return str.Sval
		}
	}
	return ""
}

// extractParameters extracts all parameters with context
func (sp *SQLParser) extractParameters(stmt *pg_query_go.Node) []ParameterInfo {
	params := make(map[int]*ParameterInfo)

	if selectStmt := stmt.GetSelectStmt(); selectStmt != nil {
		// Walk CTEs first
		if selectStmt.WithClause != nil {
			for _, cteNode := range selectStmt.WithClause.Ctes {
				if commonTableExpr := cteNode.GetCommonTableExpr(); commonTableExpr != nil {
					if commonTableExpr.Ctequery != nil {
						if cteSelect := commonTableExpr.Ctequery.GetSelectStmt(); cteSelect != nil {
							sp.walkSelectForParams(cteSelect, params)
						}
					}
				}
			}
		}
		sp.walkSelectForParams(selectStmt, params)
	} else if updateStmt := stmt.GetUpdateStmt(); updateStmt != nil {
		sp.walkUpdateForParams(updateStmt, params)
	} else if insertStmt := stmt.GetInsertStmt(); insertStmt != nil {
		sp.walkInsertForParams(insertStmt, params)
	} else if deleteStmt := stmt.GetDeleteStmt(); deleteStmt != nil {
		sp.walkDeleteForParams(deleteStmt, params)
	}

	return sp.sortParameters(params)
}

// walkSelectForParams walks SELECT statement for parameters
func (sp *SQLParser) walkSelectForParams(selectStmt *pg_query_go.SelectStmt, params map[int]*ParameterInfo) {
	for _, target := range selectStmt.TargetList {
		if resTarget := target.GetResTarget(); resTarget != nil && resTarget.Val != nil {
			sp.walkExprForParams(resTarget.Val, params, false, false)
		}
	}

	if selectStmt.WhereClause != nil {
		sp.walkExprForParams(selectStmt.WhereClause, params, true, false)
	}

	if selectStmt.LimitCount != nil {
		if paramRef := selectStmt.LimitCount.GetParamRef(); paramRef != nil {
			pos := int(paramRef.Number)
			if params[pos] == nil {
				params[pos] = &ParameterInfo{Position: pos}
			}
			params[pos].IsInLimit = true
		}
	}

	if selectStmt.LimitOffset != nil {
		if paramRef := selectStmt.LimitOffset.GetParamRef(); paramRef != nil {
			pos := int(paramRef.Number)
			if params[pos] == nil {
				params[pos] = &ParameterInfo{Position: pos}
			}
			params[pos].IsInOffset = true
		}
	}

	// Walk HAVING clause for parameters
	if selectStmt.HavingClause != nil {
		sp.walkExprForParams(selectStmt.HavingClause, params, false, false)
	}
}

// walkUpdateForParams walks UPDATE statement for parameters
func (sp *SQLParser) walkUpdateForParams(updateStmt *pg_query_go.UpdateStmt, params map[int]*ParameterInfo) {
	tableName := ""
	if updateStmt.Relation != nil {
		tableName = updateStmt.Relation.Relname
	}

	for _, node := range updateStmt.TargetList {
		if resTarget := node.GetResTarget(); resTarget != nil {
			columnName := resTarget.Name
			targetTable := tableName
			if len(resTarget.Indirection) > 0 {
				lastIndirection := resTarget.Indirection[len(resTarget.Indirection)-1]
				if lastIndirection != nil {
					if str := lastIndirection.GetString_(); str != nil {
						columnName = str.Sval
					}
				}
				targetTable = resTarget.Name
			}
			sp.walkExprForParamsWithColumn(resTarget.Val, params, targetTable, columnName, false, true)
		}
	}

	if updateStmt.WhereClause != nil {
		sp.walkExprForParams(updateStmt.WhereClause, params, true, false)
	}
}

// walkInsertForParams walks INSERT values for parameters
func (sp *SQLParser) walkInsertForParams(insertStmt *pg_query_go.InsertStmt, params map[int]*ParameterInfo) {
	if insertStmt.SelectStmt == nil {
		return
	}

	selectStmt := insertStmt.SelectStmt.GetSelectStmt()
	if selectStmt == nil {
		return
	}

	tableName := ""
	if insertStmt.Relation != nil {
		tableName = insertStmt.Relation.Relname
	}

	columnNames := make([]string, 0)
	for _, col := range insertStmt.Cols {
		if resTarget := col.GetResTarget(); resTarget != nil {
			columnNames = append(columnNames, resTarget.Name)
		}
	}

	if len(selectStmt.ValuesLists) > 0 {
		valuesList := selectStmt.ValuesLists[0]
		if list := valuesList.GetList(); list != nil {
			for i, node := range list.Items {
				if paramRef := node.GetParamRef(); paramRef != nil {
					pos := int(paramRef.Number)
					if params[pos] == nil {
						params[pos] = &ParameterInfo{Position: pos}
					}
					if i < len(columnNames) {
						params[pos].ColumnName = columnNames[i]
						params[pos].TableName = tableName
					}
				}
			}
		}
	}
}

// walkDeleteForParams walks DELETE statement for parameters
func (sp *SQLParser) walkDeleteForParams(deleteStmt *pg_query_go.DeleteStmt, params map[int]*ParameterInfo) {
	if deleteStmt.WhereClause != nil {
		sp.walkExprForParams(deleteStmt.WhereClause, params, true, false)
	}
}

// walkExprForParams walks an expression tree for parameters
func (sp *SQLParser) walkExprForParams(node *pg_query_go.Node, params map[int]*ParameterInfo, isInWhere, isInSet bool) {
	if node == nil {
		return
	}

	if paramRef := node.GetParamRef(); paramRef != nil {
		pos := int(paramRef.Number)
		if params[pos] == nil {
			params[pos] = &ParameterInfo{Position: pos}
		}
		params[pos].IsInWhere = params[pos].IsInWhere || isInWhere
		params[pos].IsInSet = params[pos].IsInSet || isInSet
		return
	}

	if aExpr := node.GetAExpr(); aExpr != nil {
		if colRef := aExpr.Lexpr.GetColumnRef(); colRef != nil {
			columnName := sp.extractColumnNameFromRef(colRef)
			tableName := sp.extractTableNameFromRef(colRef)

			if paramRef := aExpr.Rexpr.GetParamRef(); paramRef != nil {
				pos := int(paramRef.Number)
				if params[pos] == nil {
					params[pos] = &ParameterInfo{Position: pos}
				}
				params[pos].ColumnName = columnName
				if tableName != "" {
					params[pos].TableName = tableName
				}
				params[pos].Operator = sp.getOperatorName(aExpr)
				params[pos].IsInWhere = isInWhere
			} else if list := aExpr.Rexpr.GetList(); list != nil {
				for _, item := range list.Items {
					if paramRef := item.GetParamRef(); paramRef != nil {
						pos := int(paramRef.Number)
						if params[pos] == nil {
							params[pos] = &ParameterInfo{Position: pos}
						}
						params[pos].ColumnName = columnName
						if tableName != "" {
							params[pos].TableName = tableName
						}
						params[pos].Operator = sp.getOperatorName(aExpr)
						params[pos].IsInWhere = isInWhere
					}
				}
			}
		}

		sp.walkExprForParams(aExpr.Lexpr, params, isInWhere, isInSet)
		sp.walkExprForParams(aExpr.Rexpr, params, isInWhere, isInSet)
	}

	if boolExpr := node.GetBoolExpr(); boolExpr != nil {
		for _, arg := range boolExpr.Args {
			sp.walkExprForParams(arg, params, isInWhere, isInSet)
		}
	}

	if subLink := node.GetSubLink(); subLink != nil {
		if subLink.Testexpr != nil {
			sp.walkExprForParams(subLink.Testexpr, params, isInWhere, isInSet)
		}
	}

	if funcCall := node.GetFuncCall(); funcCall != nil {
		for _, arg := range funcCall.Args {
			sp.walkExprForParams(arg, params, isInWhere, isInSet)
		}
	}

	if nullTest := node.GetNullTest(); nullTest != nil {
		sp.walkExprForParams(nullTest.Arg, params, isInWhere, isInSet)
	}

	if caseExpr := node.GetCaseExpr(); caseExpr != nil {
		if caseExpr.Arg != nil {
			sp.walkExprForParams(caseExpr.Arg, params, isInWhere, isInSet)
		}
		for _, whenClause := range caseExpr.Args {
			if when := whenClause.GetCaseWhen(); when != nil {
				sp.walkExprForParams(when.Expr, params, isInWhere, isInSet)
				sp.walkExprForParams(when.Result, params, isInWhere, isInSet)
			}
		}
		if caseExpr.Defresult != nil {
			sp.walkExprForParams(caseExpr.Defresult, params, isInWhere, isInSet)
		}
	}

	// Handle IN clauses: status IN ($1) becomes ScalarArrayOpExpr
	if scalarArrayOp := node.GetScalarArrayOpExpr(); scalarArrayOp != nil {
		// Args[0] is the column, Args[1] is the parameter/array
		if len(scalarArrayOp.Args) >= 2 {
			if colRef := scalarArrayOp.Args[0].GetColumnRef(); colRef != nil {
				columnName := sp.extractColumnNameFromRef(colRef)
				tableName := sp.extractTableNameFromRef(colRef)

				// Second argument could be a parameter
				if paramRef := scalarArrayOp.Args[1].GetParamRef(); paramRef != nil {
					pos := int(paramRef.Number)
					if params[pos] == nil {
						params[pos] = &ParameterInfo{Position: pos}
					}
					params[pos].ColumnName = columnName
					if tableName != "" {
						params[pos].TableName = tableName
					}
					params[pos].Operator = "IN"
					params[pos].IsInWhere = isInWhere
				}
			}
		}

		// Recursively walk all args
		for _, arg := range scalarArrayOp.Args {
			sp.walkExprForParams(arg, params, isInWhere, isInSet)
		}
	}

	if list := node.GetList(); list != nil {
		for _, item := range list.Items {
			sp.walkExprForParams(item, params, isInWhere, isInSet)
		}
	}
}

// walkExprForParamsWithColumn walks expression with known column context
func (sp *SQLParser) walkExprForParamsWithColumn(node *pg_query_go.Node, params map[int]*ParameterInfo, tableName, columnName string, isInWhere, isInSet bool) {
	if node == nil {
		return
	}

	if paramRef := node.GetParamRef(); paramRef != nil {
		pos := int(paramRef.Number)
		if params[pos] == nil {
			params[pos] = &ParameterInfo{Position: pos}
		}
		params[pos].ColumnName = columnName
		if tableName != "" {
			params[pos].TableName = tableName
		}
		params[pos].IsInSet = isInSet
		params[pos].IsInWhere = isInWhere
		return
	}

	sp.walkExprForParams(node, params, isInWhere, isInSet)
}

// extractColumnNameFromRef extracts column name from ColumnRef
func (sp *SQLParser) extractColumnNameFromRef(colRef *pg_query_go.ColumnRef) string {
	if len(colRef.Fields) == 0 {
		return ""
	}

	lastField := colRef.Fields[len(colRef.Fields)-1]
	if lastField == nil {
		return ""
	}

	if str := lastField.GetString_(); str != nil {
		return str.Sval
	}

	return ""
}

// extractTableNameFromRef extracts table name from qualified ColumnRef
func (sp *SQLParser) extractTableNameFromRef(colRef *pg_query_go.ColumnRef) string {
	if len(colRef.Fields) < 2 {
		return ""
	}

	firstField := colRef.Fields[0]
	if firstField == nil {
		return ""
	}

	if str := firstField.GetString_(); str != nil {
		return str.Sval
	}

	return ""
}

// getOperatorName extracts operator name from A_Expr
func (sp *SQLParser) getOperatorName(aExpr *pg_query_go.A_Expr) string {
	if len(aExpr.Name) == 0 {
		return ""
	}

	firstOp := aExpr.Name[0]
	if firstOp == nil {
		return ""
	}

	if str := firstOp.GetString_(); str != nil {
		return str.Sval
	}

	return ""
}

// sortParameters converts map to sorted slice
func (sp *SQLParser) sortParameters(params map[int]*ParameterInfo) []ParameterInfo {
	if len(params) == 0 {
		return []ParameterInfo{}
	}

	maxPos := 0
	for pos := range params {
		if pos > maxPos {
			maxPos = pos
		}
	}

	result := make([]ParameterInfo, 0, maxPos)
	for i := 1; i <= maxPos; i++ {
		if param, exists := params[i]; exists {
			result = append(result, *param)
		} else {
			result = append(result, ParameterInfo{Position: i})
		}
	}

	return result
}

// extractTables extracts all table references from query
func (sp *SQLParser) extractTables(stmt *pg_query_go.Node) []TableRef {
	var tables []TableRef

	if selectStmt := stmt.GetSelectStmt(); selectStmt != nil {
		tables = sp.extractTablesFromSelect(selectStmt)
	} else if updateStmt := stmt.GetUpdateStmt(); updateStmt != nil {
		if updateStmt.Relation != nil {
			tables = append(tables, sp.makeTableRef(updateStmt.Relation))
		}
	} else if deleteStmt := stmt.GetDeleteStmt(); deleteStmt != nil {
		if deleteStmt.Relation != nil {
			tables = append(tables, sp.makeTableRef(deleteStmt.Relation))
		}
	} else if insertStmt := stmt.GetInsertStmt(); insertStmt != nil {
		if insertStmt.Relation != nil {
			tables = append(tables, sp.makeTableRef(insertStmt.Relation))
		}
	}

	return tables
}

// extractTablesFromSelect extracts tables from SELECT statement
func (sp *SQLParser) extractTablesFromSelect(selectStmt *pg_query_go.SelectStmt) []TableRef {
	var tables []TableRef

	for _, fromNode := range selectStmt.FromClause {
		tables = append(tables, sp.walkFromClause(fromNode)...)
	}

	return tables
}

// extractCTEs extracts Common Table Expressions (WITH clause) from SELECT statement
func (sp *SQLParser) extractCTEs(selectStmt *pg_query_go.SelectStmt) []CTEInfo {
	var ctes []CTEInfo

	if selectStmt.WithClause == nil {
		return ctes
	}

	for _, cteNode := range selectStmt.WithClause.Ctes {
		if commonTableExpr := cteNode.GetCommonTableExpr(); commonTableExpr != nil {
			cte := CTEInfo{
				Name: commonTableExpr.Ctename,
			}

			if commonTableExpr.Ctequery != nil {
				if cteSelect := commonTableExpr.Ctequery.GetSelectStmt(); cteSelect != nil {
					cteQueryInfo := sp.extractInfoFromSelectStmt(cteSelect)
					cte.Query = cteQueryInfo
				}
			}

			ctes = append(ctes, cte)
		}
	}

	return ctes
}

// walkFromClause walks FROM clause nodes
func (sp *SQLParser) walkFromClause(node *pg_query_go.Node) []TableRef {
	var tables []TableRef

	if rangeVar := node.GetRangeVar(); rangeVar != nil {
		tables = append(tables, sp.makeTableRef(rangeVar))
		return tables
	}

	if rangeSubselect := node.GetRangeSubselect(); rangeSubselect != nil {
		tables = append(tables, sp.makeSubqueryTableRef(rangeSubselect))
		return tables
	}

	if joinExpr := node.GetJoinExpr(); joinExpr != nil {
		tables = append(tables, sp.walkFromClause(joinExpr.Larg)...)
		tables = append(tables, sp.walkFromClause(joinExpr.Rarg)...)
		return tables
	}

	return tables
}

// makeTableRef creates TableRef from RangeVar
func (sp *SQLParser) makeTableRef(rangeVar *pg_query_go.RangeVar) TableRef {
	ref := TableRef{
		Name: rangeVar.Relname,
	}

	if rangeVar.Alias != nil {
		ref.Alias = rangeVar.Alias.Aliasname
	}

	if rangeVar.Schemaname != "" {
		ref.Schema = rangeVar.Schemaname
	}

	return ref
}

// makeSubqueryTableRef creates TableRef from RangeSubselect
func (sp *SQLParser) makeSubqueryTableRef(rangeSubselect *pg_query_go.RangeSubselect) TableRef {
	ref := TableRef{
		IsSubquery: true,
	}

	if rangeSubselect.Alias != nil {
		ref.Alias = rangeSubselect.Alias.Aliasname
	}

	if rangeSubselect.Subquery != nil {
		if selectStmt := rangeSubselect.Subquery.GetSelectStmt(); selectStmt != nil {
			subqueryInfo := sp.extractInfoFromSelectStmt(selectStmt)
			ref.SubqueryInfo = subqueryInfo
		}
	}

	return ref
}

// extractInfoFromSelectStmt extracts QueryInfo from a SelectStmt node
func (sp *SQLParser) extractInfoFromSelectStmt(selectStmt *pg_query_go.SelectStmt) *QueryInfo {
	info := &QueryInfo{
		Type:          QueryTypeMany,
		SelectTargets: sp.extractSelectTargets(selectStmt),
		Joins:         sp.extractJoins(selectStmt),
		Tables:        sp.extractTablesFromSelect(selectStmt),
	}
	return info
}
