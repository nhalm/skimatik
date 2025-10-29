package generator

import (
	"testing"
)

func TestSQLParser_Parse_Simple(t *testing.T) {
	parser := NewSQLParser()

	testCases := []struct {
		name     string
		sql      string
		wantType QueryType
	}{
		{"SELECT", "SELECT id FROM users", QueryTypeMany},
		{"INSERT", "INSERT INTO users (name) VALUES ($1)", QueryTypeExec},
		{"UPDATE", "UPDATE users SET name = $1 WHERE id = $2", QueryTypeExec},
		{"DELETE", "DELETE FROM users WHERE id = $1", QueryTypeExec},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.Parse(tc.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if info.Type != tc.wantType {
				t.Errorf("Type = %v, want %v", info.Type, tc.wantType)
			}
		})
	}
}

func TestSQLParser_Cache(t *testing.T) {
	parser := NewSQLParser()
	sql := "SELECT id FROM users"

	info1, err1 := parser.Parse(sql)
	info2, err2 := parser.Parse(sql)

	if err1 != nil || err2 != nil {
		t.Fatalf("Parse failed: %v, %v", err1, err2)
	}

	if info1 == nil || info2 == nil {
		t.Error("Expected valid query info from cache")
	}
}

func TestSQLParser_ExtractTables(t *testing.T) {
	parser := NewSQLParser()

	testCases := []struct {
		name       string
		sql        string
		wantTables int
		checkTable func(*testing.T, TableRef)
	}{
		{
			name:       "Simple FROM",
			sql:        "SELECT * FROM users",
			wantTables: 1,
			checkTable: func(t *testing.T, table TableRef) {
				if table.Name != "users" {
					t.Errorf("Name = %s, want 'users'", table.Name)
				}
			},
		},
		{
			name:       "Table with alias",
			sql:        "SELECT * FROM users u WHERE u.id = $1",
			wantTables: 1,
			checkTable: func(t *testing.T, table TableRef) {
				if table.Name != "users" {
					t.Errorf("Name = %s, want 'users'", table.Name)
				}
				if table.Alias != "u" {
					t.Errorf("Alias = %s, want 'u'", table.Alias)
				}
			},
		},
		{
			name:       "JOIN",
			sql:        "SELECT * FROM users u INNER JOIN posts p ON p.user_id = u.id",
			wantTables: 2,
		},
		{
			name:       "UPDATE",
			sql:        "UPDATE users SET name = $1",
			wantTables: 1,
		},
		{
			name:       "DELETE",
			sql:        "DELETE FROM users WHERE id = $1",
			wantTables: 1,
		},
		{
			name:       "INSERT",
			sql:        "INSERT INTO users (name, email) VALUES ($1, $2)",
			wantTables: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.Parse(tc.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if len(info.Tables) != tc.wantTables {
				t.Errorf("Got %d tables, want %d", len(info.Tables), tc.wantTables)
			}
			if tc.checkTable != nil && len(info.Tables) > 0 {
				tc.checkTable(t, info.Tables[0])
			}
		})
	}
}

func TestSQLParser_ExtractTables_Complex(t *testing.T) {
	parser := NewSQLParser()

	testCases := []struct {
		name       string
		sql        string
		wantTables []TableRef
	}{
		{
			name: "Multiple JOINs",
			sql:  "SELECT * FROM users u INNER JOIN posts p ON p.user_id = u.id LEFT JOIN comments c ON c.post_id = p.id",
			wantTables: []TableRef{
				{Name: "users", Alias: "u"},
				{Name: "posts", Alias: "p"},
				{Name: "comments", Alias: "c"},
			},
		},
		{
			name: "Table with schema",
			sql:  "SELECT * FROM public.users",
			wantTables: []TableRef{
				{Name: "users", Schema: "public"},
			},
		},
		{
			name: "Schema with alias",
			sql:  "SELECT * FROM public.users u WHERE u.id = $1",
			wantTables: []TableRef{
				{Name: "users", Schema: "public", Alias: "u"},
			},
		},
		{
			name: "Cross JOIN",
			sql:  "SELECT * FROM users u CROSS JOIN roles r",
			wantTables: []TableRef{
				{Name: "users", Alias: "u"},
				{Name: "roles", Alias: "r"},
			},
		},
		{
			name: "Self JOIN",
			sql:  "SELECT * FROM users u1 INNER JOIN users u2 ON u2.manager_id = u1.id",
			wantTables: []TableRef{
				{Name: "users", Alias: "u1"},
				{Name: "users", Alias: "u2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.Parse(tc.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(info.Tables) != len(tc.wantTables) {
				t.Fatalf("Got %d tables, want %d", len(info.Tables), len(tc.wantTables))
			}

			for i, want := range tc.wantTables {
				got := info.Tables[i]
				if got.Name != want.Name {
					t.Errorf("Table[%d].Name = %s, want %s", i, got.Name, want.Name)
				}
				if got.Alias != want.Alias {
					t.Errorf("Table[%d].Alias = %s, want %s", i, got.Alias, want.Alias)
				}
				if got.Schema != want.Schema {
					t.Errorf("Table[%d].Schema = %s, want %s", i, got.Schema, want.Schema)
				}
			}
		})
	}
}

func TestSQLParser_ExtractTables_DML(t *testing.T) {
	parser := NewSQLParser()

	testCases := []struct {
		name      string
		sql       string
		wantTable TableRef
	}{
		{
			name: "UPDATE basic",
			sql:  "UPDATE users SET name = $1 WHERE id = $2",
			wantTable: TableRef{
				Name: "users",
			},
		},
		{
			name: "UPDATE with schema",
			sql:  "UPDATE public.users SET name = $1",
			wantTable: TableRef{
				Name:   "users",
				Schema: "public",
			},
		},
		{
			name: "DELETE basic",
			sql:  "DELETE FROM users WHERE id = $1",
			wantTable: TableRef{
				Name: "users",
			},
		},
		{
			name: "DELETE with schema",
			sql:  "DELETE FROM auth.users WHERE id = $1",
			wantTable: TableRef{
				Name:   "users",
				Schema: "auth",
			},
		},
		{
			name: "INSERT basic",
			sql:  "INSERT INTO users (name, email) VALUES ($1, $2)",
			wantTable: TableRef{
				Name: "users",
			},
		},
		{
			name: "INSERT with schema",
			sql:  "INSERT INTO public.users (name) VALUES ($1)",
			wantTable: TableRef{
				Name:   "users",
				Schema: "public",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.Parse(tc.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(info.Tables) != 1 {
				t.Fatalf("Got %d tables, want 1", len(info.Tables))
			}

			got := info.Tables[0]
			if got.Name != tc.wantTable.Name {
				t.Errorf("Name = %s, want %s", got.Name, tc.wantTable.Name)
			}
			if got.Schema != tc.wantTable.Schema {
				t.Errorf("Schema = %s, want %s", got.Schema, tc.wantTable.Schema)
			}
		})
	}
}

func TestSQLParser_ExtractParameters(t *testing.T) {
	parser := NewSQLParser()

	testCases := []struct {
		name       string
		sql        string
		wantParams int
		checkParam func(*testing.T, []ParameterInfo)
	}{
		{
			name:       "Simple WHERE with equality",
			sql:        "SELECT * FROM users WHERE email = $1",
			wantParams: 1,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if params[0].ColumnName != "email" {
					t.Errorf("ColumnName = %s, want 'email'", params[0].ColumnName)
				}
				if !params[0].IsInWhere {
					t.Error("Expected IsInWhere=true")
				}
				if params[0].Operator != "=" {
					t.Errorf("Operator = %s, want '='", params[0].Operator)
				}
			},
		},
		{
			name:       "LIMIT parameter",
			sql:        "SELECT * FROM users LIMIT $1",
			wantParams: 1,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if !params[0].IsInLimit {
					t.Error("Expected IsInLimit=true")
				}
			},
		},
		{
			name:       "OFFSET parameter",
			sql:        "SELECT * FROM users OFFSET $1",
			wantParams: 1,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if !params[0].IsInOffset {
					t.Error("Expected IsInOffset=true")
				}
			},
		},
		{
			name:       "UPDATE SET clause",
			sql:        "UPDATE users SET name = $1 WHERE id = $2",
			wantParams: 2,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if params[0].ColumnName != "name" {
					t.Errorf("First param ColumnName = %s, want 'name'", params[0].ColumnName)
				}
				if !params[0].IsInSet {
					t.Error("Expected first param IsInSet=true")
				}
				if params[0].TableName != "users" {
					t.Errorf("First param TableName = %s, want 'users'", params[0].TableName)
				}

				if params[1].ColumnName != "id" {
					t.Errorf("Second param ColumnName = %s, want 'id'", params[1].ColumnName)
				}
				if !params[1].IsInWhere {
					t.Error("Expected second param IsInWhere=true")
				}
			},
		},
		{
			name:       "Multiple WHERE conditions with AND",
			sql:        "SELECT * FROM users WHERE email = $1 AND status = $2",
			wantParams: 2,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if params[0].ColumnName != "email" {
					t.Errorf("First param ColumnName = %s, want 'email'", params[0].ColumnName)
				}
				if params[1].ColumnName != "status" {
					t.Errorf("Second param ColumnName = %s, want 'status'", params[1].ColumnName)
				}
			},
		},
		{
			name:       "Qualified column name (table.column)",
			sql:        "SELECT * FROM users WHERE users.email = $1",
			wantParams: 1,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if params[0].ColumnName != "email" {
					t.Errorf("ColumnName = %s, want 'email'", params[0].ColumnName)
				}
				if params[0].TableName != "users" {
					t.Errorf("TableName = %s, want 'users'", params[0].TableName)
				}
			},
		},
		{
			name:       "Comparison operators (>)",
			sql:        "SELECT * FROM products WHERE price > $1",
			wantParams: 1,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if params[0].ColumnName != "price" {
					t.Errorf("ColumnName = %s, want 'price'", params[0].ColumnName)
				}
				if params[0].Operator != ">" {
					t.Errorf("Operator = %s, want '>'", params[0].Operator)
				}
			},
		},
		{
			name:       "INSERT statement",
			sql:        "INSERT INTO users (email, name, status) VALUES ($1, $2, $3)",
			wantParams: 3,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if params[0].ColumnName != "email" {
					t.Errorf("First param ColumnName = %s, want 'email'", params[0].ColumnName)
				}
				if params[0].TableName != "users" {
					t.Errorf("First param TableName = %s, want 'users'", params[0].TableName)
				}
				if params[1].ColumnName != "name" {
					t.Errorf("Second param ColumnName = %s, want 'name'", params[1].ColumnName)
				}
				if params[2].ColumnName != "status" {
					t.Errorf("Third param ColumnName = %s, want 'status'", params[2].ColumnName)
				}
			},
		},
		{
			name:       "DELETE statement",
			sql:        "DELETE FROM users WHERE id = $1",
			wantParams: 1,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if params[0].ColumnName != "id" {
					t.Errorf("ColumnName = %s, want 'id'", params[0].ColumnName)
				}
				if !params[0].IsInWhere {
					t.Error("Expected IsInWhere=true")
				}
			},
		},
		{
			name:       "Complex UPDATE with multiple SET and WHERE",
			sql:        "UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3 AND merchant_id = $4",
			wantParams: 4,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if params[0].ColumnName != "status" || !params[0].IsInSet {
					t.Error("First param should be 'status' in SET")
				}
				if params[1].ColumnName != "updated_at" || !params[1].IsInSet {
					t.Error("Second param should be 'updated_at' in SET")
				}
				if params[2].ColumnName != "id" || !params[2].IsInWhere {
					t.Error("Third param should be 'id' in WHERE")
				}
				if params[3].ColumnName != "merchant_id" || !params[3].IsInWhere {
					t.Error("Fourth param should be 'merchant_id' in WHERE")
				}
			},
		},
		{
			name:       "SELECT with LIMIT and OFFSET",
			sql:        "SELECT * FROM users WHERE status = $1 LIMIT $2 OFFSET $3",
			wantParams: 3,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if !params[0].IsInWhere {
					t.Error("First param should be in WHERE")
				}
				if !params[1].IsInLimit {
					t.Error("Second param should be in LIMIT")
				}
				if !params[2].IsInOffset {
					t.Error("Third param should be in OFFSET")
				}
			},
		},
		{
			name:       "No parameters",
			sql:        "SELECT * FROM users",
			wantParams: 0,
		},
		{
			name:       "Parameter in nested expression",
			sql:        "SELECT * FROM users WHERE (email = $1 OR username = $2) AND status = $3",
			wantParams: 3,
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if params[0].ColumnName != "email" {
					t.Errorf("First param ColumnName = %s, want 'email'", params[0].ColumnName)
				}
				if params[1].ColumnName != "username" {
					t.Errorf("Second param ColumnName = %s, want 'username'", params[1].ColumnName)
				}
				if params[2].ColumnName != "status" {
					t.Errorf("Third param ColumnName = %s, want 'status'", params[2].ColumnName)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.Parse(tc.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if len(info.Parameters) != tc.wantParams {
				t.Errorf("Got %d params, want %d", len(info.Parameters), tc.wantParams)
			}
			if tc.checkParam != nil && len(info.Parameters) > 0 {
				tc.checkParam(t, info.Parameters)
			}
		})
	}
}

func TestSQLParser_ParameterPositions(t *testing.T) {
	parser := NewSQLParser()

	testCases := []struct {
		name          string
		sql           string
		wantPositions []int
	}{
		{
			name:          "Sequential positions",
			sql:           "SELECT * FROM users WHERE email = $1 AND name = $2 AND status = $3",
			wantPositions: []int{1, 2, 3},
		},
		{
			name:          "Single position",
			sql:           "SELECT * FROM users WHERE id = $1",
			wantPositions: []int{1},
		},
		{
			name:          "Gaps in positions should fill",
			sql:           "SELECT * FROM users WHERE id = $1 AND status = $3",
			wantPositions: []int{1, 2, 3},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.Parse(tc.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(info.Parameters) != len(tc.wantPositions) {
				t.Fatalf("Got %d params, want %d", len(info.Parameters), len(tc.wantPositions))
			}

			for i, wantPos := range tc.wantPositions {
				if info.Parameters[i].Position != wantPos {
					t.Errorf("Parameter %d: Position = %d, want %d", i, info.Parameters[i].Position, wantPos)
				}
			}
		})
	}
}

func TestSQLParser_ComplexExpressions(t *testing.T) {
	parser := NewSQLParser()

	testCases := []struct {
		name       string
		sql        string
		checkParam func(*testing.T, []ParameterInfo)
	}{
		{
			name: "Function call with parameter",
			sql:  "SELECT * FROM users WHERE LOWER(email) = LOWER($1)",
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if len(params) != 1 {
					t.Fatalf("Expected 1 param, got %d", len(params))
				}
				if !params[0].IsInWhere {
					t.Error("Expected parameter in WHERE clause")
				}
			},
		},
		{
			name: "CASE expression with parameters",
			sql:  "SELECT CASE WHEN status = $1 THEN $2 ELSE $3 END FROM users",
			checkParam: func(t *testing.T, params []ParameterInfo) {
				if len(params) != 3 {
					t.Fatalf("Expected 3 params, got %d", len(params))
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.Parse(tc.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			tc.checkParam(t, info.Parameters)
		})
	}
}

func TestSQLParser_InvalidSQL(t *testing.T) {
	parser := NewSQLParser()

	testCases := []struct {
		name string
		sql  string
	}{
		{
			name: "Syntax error",
			sql:  "SELECT * FROM WHERE",
		},
		{
			name: "Incomplete statement",
			sql:  "SELECT * FROM",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parser.Parse(tc.sql)
			if err == nil {
				t.Error("Expected error for invalid SQL, got none")
			}
		})
	}
}

func TestSQLParser_ExtractSelectTargets(t *testing.T) {
	parser := NewSQLParser()

	testCases := []struct {
		name        string
		sql         string
		wantTargets int
		checkTarget func(*testing.T, SelectTarget)
	}{
		{
			name:        "Simple columns",
			sql:         "SELECT id, name FROM users",
			wantTargets: 2,
		},
		{
			name:        "COUNT aggregate",
			sql:         "SELECT COUNT(*) as total FROM users",
			wantTargets: 1,
			checkTarget: func(t *testing.T, target SelectTarget) {
				if !target.IsCount {
					t.Error("Expected IsCount=true")
				}
				if target.Alias != "total" {
					t.Errorf("Alias = %s, want 'total'", target.Alias)
				}
			},
		},
		{
			name:        "Multiple aggregates",
			sql:         "SELECT COUNT(*) as cnt, SUM(amount) as total, AVG(amount) as avg FROM payments",
			wantTargets: 3,
		},
		{
			name:        "Mixed columns and aggregates",
			sql:         "SELECT id, name, COUNT(*) as post_count FROM users",
			wantTargets: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.Parse(tc.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if len(info.SelectTargets) != tc.wantTargets {
				t.Errorf("Got %d targets, want %d", len(info.SelectTargets), tc.wantTargets)
			}
			if tc.checkTarget != nil && len(info.SelectTargets) > 0 {
				tc.checkTarget(t, info.SelectTargets[0])
			}
		})
	}
}
