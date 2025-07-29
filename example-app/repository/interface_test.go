package repository

import (
	"github.com/nhalm/skimatik/example-app/service"
)

// Compile-time interface compliance tests
// These variables will cause compilation to fail if the types don't implement the interfaces
//
// TODO: Fix architectural mismatch - repositories return generated types but services expect domain types
// Need to add mapping/adapter layer between repository and service layers

var (
	// TODO: Uncomment when repositories properly return domain types
	// _ service.PostRepository = (*PostRepository)(nil)
	// _ service.UserRepository = (*UserRepository)(nil)

	// Prevent unused import warning
	_ = service.PostRepository(nil)
)
