package repository

import (
	"github.com/nhalm/skimatik/example-app/service"
)

// Compile-time interface compliance tests
// These variables will cause compilation to fail if the types don't implement the interfaces

var (
	_ service.PostRepository = (*PostRepositoryStub)(nil)
	_ service.UserRepository = (*UserRepositoryStub)(nil)
)
