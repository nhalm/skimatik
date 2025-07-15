package generator

import (
	"strings"
	"testing"
)

// TestTable_GetColumn - simplified to test core functionality only
func TestTable_GetColumn(t *testing.T) {
	table := Table{
		Name:   "users",
		Schema: "public",
		Columns: []Column{
			{Name: "id", Type: "uuid", GoType: "uuid.UUID"},
			{Name: "name", Type: "text", GoType: "string"},
		},
	}

	// Test existing column
	col := table.GetColumn("name")
	if col == nil || col.Name != "name" {
		t.Errorf("GetColumn() failed for existing column")
	}

	// Test non-existing column
	if table.GetColumn("nonexistent") != nil {
		t.Errorf("GetColumn() should return nil for non-existing column")
	}
}

// TestTable_GetPrimaryKeyColumn - simplified to test core functionality
func TestTable_GetPrimaryKeyColumn(t *testing.T) {
	// Test single primary key
	table := Table{
		Name:       "users",
		PrimaryKey: []string{"id"},
		Columns: []Column{
			{Name: "id", Type: "uuid", GoType: "uuid.UUID"},
		},
	}

	col := table.GetPrimaryKeyColumn()
	if col == nil || col.Name != "id" {
		t.Errorf("GetPrimaryKeyColumn() failed for single primary key")
	}

	// Test composite primary key (should return nil)
	table.PrimaryKey = []string{"user_id", "role_id"}
	if table.GetPrimaryKeyColumn() != nil {
		t.Errorf("GetPrimaryKeyColumn() should return nil for composite primary key")
	}

	// Test no primary key
	table.PrimaryKey = []string{}
	if table.GetPrimaryKeyColumn() != nil {
		t.Errorf("GetPrimaryKeyColumn() should return nil for no primary key")
	}
}

// TestTable_GoStructName - keep essential naming tests
func TestTable_GoStructName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"users", "Users"},
		{"user_profiles", "UserProfiles"},
		{"", ""},
	}

	for _, tt := range tests {
		table := Table{Name: tt.name}
		if got := table.GoStructName(); got != tt.want {
			t.Errorf("GoStructName() = %v, want %v", got, tt.want)
		}
	}
}

// TestTable_GoStructName_SpecialCharacters - test edge cases with special characters
func TestTable_GoStructName_SpecialCharacters(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "underscore_case",
			input:    "user_profile",
			expected: "UserProfile",
		},
		{
			name:     "multiple_underscores",
			input:    "user_profile_data",
			expected: "UserProfileData",
		},
		{
			name:     "leading_underscore",
			input:    "_private_field",
			expected: "PrivateField",
		},
		{
			name:     "trailing_underscore",
			input:    "field_name_",
			expected: "FieldName",
		},
		{
			name:     "multiple_consecutive_underscores",
			input:    "user__profile",
			expected: "UserProfile",
		},
		{
			name:     "empty_string",
			input:    "",
			expected: "",
		},
		{
			name:     "single_character",
			input:    "a",
			expected: "A",
		},
		{
			name:     "already_pascal_case",
			input:    "UserProfile",
			expected: "UserProfile",
		},
		{
			name:     "mixed_case",
			input:    "userId",
			expected: "UserId",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			table := Table{Name: tc.input}
			result := table.GoStructName()
			if result != tc.expected {
				t.Errorf("GoStructName(%s) = %s, want %s", tc.input, result, tc.expected)
			}
		})
	}
}

// TestTable_GoFileName - keep essential filename tests
func TestTable_GoFileName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"users", "users_generated.go"},
		{"user_profiles", "user_profiles_generated.go"},
		{"UserProfiles", "user_profiles_generated.go"},
	}

	for _, tt := range tests {
		table := Table{Name: tt.name}
		if got := table.GoFileName(); got != tt.want {
			t.Errorf("GoFileName() = %v, want %v", got, tt.want)
		}
	}
}

// TestTable_LongNames - test handling of very long table names
func TestTable_LongNames(t *testing.T) {
	// Test handling of very long table and column names
	longName := strings.Repeat("very_long_name_", 10) + "end"

	table := Table{
		Name:   longName,
		Schema: "public",
	}

	// Test that long names are handled properly
	structName := table.GoStructName()
	if structName == "" {
		t.Error("Long table name should produce non-empty struct name")
	}

	// Test that the result is valid Go identifier (starts with uppercase)
	if len(structName) > 0 && (structName[0] < 'A' || structName[0] > 'Z') {
		t.Errorf("Struct name should start with uppercase letter, got %s", structName)
	}

	// Test filename generation
	fileName := table.GoFileName()
	if fileName == "" {
		t.Error("Long table name should produce non-empty filename")
	}

	if !strings.HasSuffix(fileName, "_generated.go") {
		t.Errorf("Filename should end with _generated.go, got %s", fileName)
	}
}

// TestColumn_IsUUID - keep type validation tests
func TestColumn_IsUUID(t *testing.T) {
	tests := []struct {
		columnType string
		want       bool
	}{
		{"uuid", true},
		{"UUID", true},
		{"text", false},
		{"", false},
	}

	for _, tt := range tests {
		col := Column{Type: tt.columnType}
		if got := col.IsUUID(); got != tt.want {
			t.Errorf("IsUUID() for %s = %v, want %v", tt.columnType, got, tt.want)
		}
	}
}

// TestColumn_IsString - keep type validation tests
func TestColumn_IsString(t *testing.T) {
	tests := []struct {
		columnType string
		want       bool
	}{
		{"text", true},
		{"varchar", true},
		{"character varying", true},
		{"TEXT", true},
		{"integer", false},
		{"", false},
	}

	for _, tt := range tests {
		col := Column{Type: tt.columnType}
		if got := col.IsString(); got != tt.want {
			t.Errorf("IsString() for %s = %v, want %v", tt.columnType, got, tt.want)
		}
	}
}

// TestColumn_IsInteger - keep type validation tests
func TestColumn_IsInteger(t *testing.T) {
	tests := []struct {
		columnType string
		want       bool
	}{
		{"integer", true},
		{"int", true},
		{"bigint", true},
		{"smallint", true},
		{"INTEGER", true},
		{"text", false},
		{"", false},
	}

	for _, tt := range tests {
		col := Column{Type: tt.columnType}
		if got := col.IsInteger(); got != tt.want {
			t.Errorf("IsInteger() for %s = %v, want %v", tt.columnType, got, tt.want)
		}
	}
}

// TestColumn_IsBoolean - keep type validation tests
func TestColumn_IsBoolean(t *testing.T) {
	tests := []struct {
		columnType string
		want       bool
	}{
		{"boolean", true},
		{"bool", true},
		{"BOOLEAN", true},
		{"text", false},
		{"", false},
	}

	for _, tt := range tests {
		col := Column{Type: tt.columnType}
		if got := col.IsBoolean(); got != tt.want {
			t.Errorf("IsBoolean() for %s = %v, want %v", tt.columnType, got, tt.want)
		}
	}
}

// TestColumn_IsTimestamp - keep type validation tests
func TestColumn_IsTimestamp(t *testing.T) {
	tests := []struct {
		columnType string
		want       bool
	}{
		{"timestamp", true},
		{"timestamptz", true},
		{"timestamp with time zone", true},
		{"TIMESTAMP", true},
		{"date", true},
		{"time", true},
		{"text", false},
		{"", false},
	}

	for _, tt := range tests {
		col := Column{Type: tt.columnType}
		if got := col.IsTimestamp(); got != tt.want {
			t.Errorf("IsTimestamp() for %s = %v, want %v", tt.columnType, got, tt.want)
		}
	}
}

// TestColumn_GoFieldName - keep essential naming tests
func TestColumn_GoFieldName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"id", "Id"},
		{"user_name", "UserName"},
		{"created_at", "CreatedAt"},
		{"", ""},
	}

	for _, tt := range tests {
		col := Column{Name: tt.name}
		if got := col.GoFieldName(); got != tt.want {
			t.Errorf("GoFieldName() = %v, want %v", got, tt.want)
		}
	}
}

// TestColumn_GoStructTag - keep essential struct tag tests
func TestColumn_GoStructTag(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"id", `json:"id" db:"id"`},
		{"user_name", `json:"user_name" db:"user_name"`},
		{"created_at", `json:"created_at" db:"created_at"`},
	}

	for _, tt := range tests {
		col := Column{Name: tt.name}
		if got := col.GoStructTag(); got != tt.want {
			t.Errorf("GoStructTag() = %v, want %v", got, tt.want)
		}
	}
}

// TestToPascalCase - keep essential string conversion tests
func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user", "User"},
		{"user_profile", "UserProfile"},
		{"user_profile_settings", "UserProfileSettings"},
		{"", ""},
		{"UserProfile", "UserProfile"},
	}

	for _, tt := range tests {
		if got := toPascalCase(tt.input); got != tt.want {
			t.Errorf("toPascalCase(%s) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// TestToSnakeCase - keep essential string conversion tests
func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"User", "user"},
		{"UserProfile", "user_profile"},
		{"UserProfileSettings", "user_profile_settings"},
		{"", ""},
		{"user_profile", "user_profile"},
	}

	for _, tt := range tests {
		if got := toSnakeCase(tt.input); got != tt.want {
			t.Errorf("toSnakeCase(%s) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// TestQueryType_Constants - keep essential constant tests
func TestQueryType_Constants(t *testing.T) {
	tests := []struct {
		name  string
		value QueryType
		want  string
	}{
		{"QueryTypeOne", QueryTypeOne, "one"},
		{"QueryTypeMany", QueryTypeMany, "many"},
		{"QueryTypeExec", QueryTypeExec, "exec"},
		{"QueryTypePaginated", QueryTypePaginated, "paginated"},
	}

	for _, tt := range tests {
		if got := string(tt.value); got != tt.want {
			t.Errorf("QueryType constant %s = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestTable_NilHandling - test handling of nil inputs for robustness
func TestTable_NilHandling(t *testing.T) {
	// Test GetColumn with nil columns slice
	table := Table{
		Name:    "test_table",
		Schema:  "public",
		Columns: nil,
	}

	col := table.GetColumn("nonexistent")
	if col != nil {
		t.Errorf("GetColumn on table with nil columns should return nil, got %v", col)
	}

	// Test GetPrimaryKeyColumn with nil primary key
	pkCol := table.GetPrimaryKeyColumn()
	if pkCol != nil {
		t.Errorf("GetPrimaryKeyColumn on table with nil primary key should return nil, got %v", pkCol)
	}

	// Test empty primary key slice
	table.PrimaryKey = []string{}
	pkCol = table.GetPrimaryKeyColumn()
	if pkCol != nil {
		t.Errorf("GetPrimaryKeyColumn on table with empty primary key should return nil, got %v", pkCol)
	}
}
