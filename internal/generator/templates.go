package generator

import "embed"

// Embed all template files at build time
//
//go:embed templates/crud/* templates/pagination/* templates/repository/* templates/queries/* templates/shared/* templates/tests/*
var templateFS embed.FS

// Template file paths (constants for type safety)
const (
	// CRUD templates
	TemplateGetByID = "templates/crud/get_by_id.tmpl"
	TemplateCreate  = "templates/crud/create.tmpl"
	TemplateUpdate  = "templates/crud/update.tmpl"
	TemplateDelete  = "templates/crud/delete.tmpl"
	TemplateList    = "templates/crud/list.tmpl"

	// Pagination templates
	TemplatePaginationShared              = "templates/pagination/shared_types.tmpl"
	TemplatePaginationInline              = "templates/pagination/inline_paginated.tmpl"
	TemplatePaginationUtils               = "templates/pagination/pagination_utils.tmpl"
	TemplatePaginationSharedTypes         = "templates/pagination/shared_pagination_types.tmpl"
	TemplatePaginationSharedListPaginated = "templates/pagination/shared_list_paginated.tmpl"

	// Query templates
	TemplateQueryResultStruct = "templates/queries/result_struct.tmpl"
	TemplateQueryRepository   = "templates/queries/repository.tmpl"
	TemplateQueryOne          = "templates/queries/one_query.tmpl"
	TemplateQueryMany         = "templates/queries/many_query.tmpl"
	TemplateQueryExec         = "templates/queries/exec_query.tmpl"
	TemplateQueryPaginated    = "templates/queries/paginated_query.tmpl"

	// Repository templates
	TemplateRepositoryStruct = "templates/repository/repository_struct.tmpl"
	TemplateRepositoryRetry  = "templates/repository/retry_methods.tmpl"
	TemplateRepositoryHealth = "templates/repository/health_methods.tmpl"

	// Shared templates
	TemplateStruct             = "templates/shared/struct.tmpl"
	TemplateHeader             = "templates/shared/header.tmpl"
	TemplateErrorHandling      = "templates/shared/error_handling.tmpl"
	TemplateSharedErrors       = "templates/shared/errors.tmpl"
	TemplateDatabaseOperations = "templates/shared/database_operations.tmpl"

	// Test templates
	TemplateRepositoryTest = "templates/tests/repository_test.tmpl"
)
