# Repository Embedding Patterns

## Overview

skimatik generates repositories that are designed for embedding and extension. This guide shows how teams can integrate generated repositories into their architecture while maintaining clean separation of concerns.

## Core Philosophy

- **Teams define interfaces** - create interfaces that match your domain needs
- **Embed generated repositories** - get all CRUD operations automatically  
- **Extend with business logic** - add custom methods using shared utilities
- **Compose at service layer** - combine multiple repositories for complex operations

## Pattern 1: Direct Embedding

### Simple Use Case
When you need basic CRUD operations with minimal custom logic:

```go
package services

import (
    "context"
    "your-project/repositories"
)

// Define interface based on your domain
type UserManager interface {
    CreateUser(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error)
    GetUser(ctx context.Context, id uuid.UUID) (*repositories.Users, error)
    UpdateUser(ctx context.Context, id uuid.UUID, params repositories.UpdateUsersParams) (*repositories.Users, error)
    DeleteUser(ctx context.Context, id uuid.UUID) error
    ListUsers(ctx context.Context) ([]repositories.Users, error)
}

// Implementation embeds generated repository
type UserService struct {
    *repositories.UsersRepository
}

func NewUserService(userRepo *repositories.UsersRepository) UserManager {
    return &UserService{
        UsersRepository: userRepo,
    }
}

// All interface methods automatically satisfied by embedded repository
// No additional code needed!
```

### Benefits
- ✅ **Zero boilerplate** - embedded methods satisfy interface
- ✅ **Type safety** - compile-time checking of interface compliance
- ✅ **Testability** - easy to mock interfaces for unit tests
- ✅ **Flexibility** - can swap implementations easily

## Pattern 2: Embedding with Extensions

### Adding Custom Business Logic

```go
package services

import (
    "context"
    "fmt"
    "your-project/repositories"
)

// Interface includes both standard and custom methods
type UserService interface {
    // Standard CRUD (satisfied by embedding)
    CreateUser(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error)
    GetUser(ctx context.Context, id uuid.UUID) (*repositories.Users, error)
    
    // Custom business logic
    CreateUserWithProfile(ctx context.Context, userData repositories.CreateUsersParams, bio string) (*repositories.Users, error)
    GetActiveUsers(ctx context.Context) ([]repositories.Users, error)
    ActivateUser(ctx context.Context, id uuid.UUID) error
}

type userService struct {
    *repositories.UsersRepository  // Embedded - provides standard CRUD
    profileRepo *repositories.ProfilesRepository
}

func NewUserService(userRepo *repositories.UsersRepository, profileRepo *repositories.ProfilesRepository) UserService {
    return &userService{
        UsersRepository: userRepo,
        profileRepo:     profileRepo,
    }
}

// Standard methods automatically available via embedding

// Custom business logic using shared utilities
func (s *userService) CreateUserWithProfile(ctx context.Context, userData repositories.CreateUsersParams, bio string) (*repositories.Users, error) {
    return repositories.RetryOperation(ctx, repositories.DefaultRetryConfig, "create_user_with_profile", func(ctx context.Context) (*repositories.Users, error) {
        // Create user first
        user, err := s.UsersRepository.Create(ctx, userData)
        if err != nil {
            return nil, err
        }
        
        // Create profile
        _, err = s.profileRepo.Create(ctx, repositories.CreateProfilesParams{
            UserID: user.Id,
            Bio:    bio,
        })
        if err != nil {
            // In production, you might want to rollback the user creation
            return nil, fmt.Errorf("failed to create profile for user %s: %w", user.Id, err)
        }
        
        return user, nil
    })
}

func (s *userService) GetActiveUsers(ctx context.Context) ([]repositories.Users, error) {
    // Custom query using shared database utilities
    query := `SELECT id, name, email, created_at FROM users WHERE is_active = true ORDER BY created_at DESC`
    
    rows, err := repositories.ExecuteQuery(ctx, s.db, "get_active_users", "Users", query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []repositories.Users
    for rows.Next() {
        var user repositories.Users
        err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
        if err != nil {
            return nil, repositories.HandleDatabaseError("scan", "Users", err)
        }
        results = append(results, user)
    }
    
    return results, repositories.HandleRowsResult("Users", rows)
}

func (s *userService) ActivateUser(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE users SET is_active = true, updated_at = NOW() WHERE id = $1`
    return repositories.ExecuteNonQuery(ctx, s.db, "activate_user", "Users", query, id)
}
```

## Pattern 3: Composition with Multiple Repositories

### Complex Business Operations

```go
package services

import (
    "context"
    "fmt"
    "your-project/repositories"
)

// Interface focusing on business operations
type BlogService interface {
    CreateBlogPost(ctx context.Context, authorID uuid.UUID, title, content string) (*BlogPost, error)
    GetUserBlogPosts(ctx context.Context, userID uuid.UUID, limit int) ([]BlogPost, error)
    PublishPost(ctx context.Context, postID uuid.UUID) error
    DeleteUserAccount(ctx context.Context, userID uuid.UUID) error
}

// Domain model (different from database models)
type BlogPost struct {
    ID       uuid.UUID `json:"id"`
    Title    string    `json:"title"`
    Content  string    `json:"content"`
    Author   string    `json:"author_name"`
    Created  time.Time `json:"created_at"`
}

type blogService struct {
    userRepo *repositories.UsersRepository
    postRepo *repositories.PostsRepository
}

func NewBlogService(userRepo *repositories.UsersRepository, postRepo *repositories.PostsRepository) BlogService {
    return &blogService{
        userRepo: userRepo,
        postRepo: postRepo,
    }
}

func (s *blogService) CreateBlogPost(ctx context.Context, authorID uuid.UUID, title, content string) (*BlogPost, error) {
    return repositories.RetryOperation(ctx, repositories.DefaultRetryConfig, "create_blog_post", func(ctx context.Context) (*BlogPost, error) {
        // Verify user exists
        user, err := s.userRepo.GetByID(ctx, authorID)
        if err != nil {
            return nil, fmt.Errorf("invalid author: %w", err)
        }
        
        // Create post
        post, err := s.postRepo.Create(ctx, repositories.CreatePostsParams{
            UserID:  authorID,
            Title:   title,
            Content: content,
        })
        if err != nil {
            return nil, err
        }
        
        // Return domain model
        return &BlogPost{
            ID:      post.Id,
            Title:   post.Title,
            Content: post.Content,
            Author:  user.Name,
            Created: post.CreatedAt.Time,
        }, nil
    })
}

func (s *blogService) GetUserBlogPosts(ctx context.Context, userID uuid.UUID, limit int) ([]BlogPost, error) {
    // Complex query across multiple tables using shared utilities
    query := `
        SELECT p.id, p.title, p.content, u.name, p.created_at 
        FROM posts p 
        JOIN users u ON p.user_id = u.id 
        WHERE p.user_id = $1 
        ORDER BY p.created_at DESC 
        LIMIT $2
    `
    
    rows, err := repositories.ExecuteQuery(ctx, s.userRepo.DB(), "get_user_blog_posts", "Posts", query, userID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []BlogPost
    for rows.Next() {
        var post BlogPost
        err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Author, &post.Created)
        if err != nil {
            return nil, repositories.HandleDatabaseError("scan", "Posts", err)
        }
        results = append(results, post)
    }
    
    return results, repositories.HandleRowsResult("Posts", rows)
}

func (s *blogService) DeleteUserAccount(ctx context.Context, userID uuid.UUID) error {
    return repositories.RetryOperation(ctx, repositories.DefaultRetryConfig, "delete_user_account", func(ctx context.Context) (struct{}, error) {
        // Delete user's posts first (foreign key constraint)
        deletePostsQuery := `DELETE FROM posts WHERE user_id = $1`
        if err := repositories.ExecuteNonQuery(ctx, s.postRepo.DB(), "delete_user_posts", "Posts", deletePostsQuery, userID); err != nil {
            return struct{}{}, err
        }
        
        // Delete user
        if err := s.userRepo.Delete(ctx, userID); err != nil {
            return struct{}{}, err
        }
        
        return struct{}{}, nil
    })
}
```

## Pattern 4: Repository Wrapper

### When You Need Different Interfaces

```go
package adapters

import (
    "context"
    "your-project/repositories"
)

// External interface (e.g., required by third-party package)
type UserDatastore interface {
    Save(ctx context.Context, user UserData) error
    Load(ctx context.Context, id string) (UserData, error)
    List(ctx context.Context, offset, limit int) ([]UserData, error)
}

// External data format
type UserData struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Adapter wraps generated repository
type userDatastoreAdapter struct {
    repo *repositories.UsersRepository
}

func NewUserDatastore(repo *repositories.UsersRepository) UserDatastore {
    return &userDatastoreAdapter{repo: repo}
}

func (a *userDatastoreAdapter) Save(ctx context.Context, user UserData) error {
    id, err := uuid.Parse(user.ID)
    if err != nil {
        return err
    }
    
    _, err = a.repo.Update(ctx, id, repositories.UpdateUsersParams{
        Name:  user.Name,
        Email: user.Email,
    })
    return err
}

func (a *userDatastoreAdapter) Load(ctx context.Context, id string) (UserData, error) {
    uuid, err := uuid.Parse(id)
    if err != nil {
        return UserData{}, err
    }
    
    user, err := a.repo.GetByID(ctx, uuid)
    if err != nil {
        return UserData{}, err
    }
    
    return UserData{
        ID:    user.Id.String(),
        Name:  user.Name,
        Email: user.Email,
    }, nil
}

func (a *userDatastoreAdapter) List(ctx context.Context, offset, limit int) ([]UserData, error) {
    users, err := a.repo.List(ctx)
    if err != nil {
        return nil, err
    }
    
    // Apply offset/limit logic and convert
    var result []UserData
    for i, user := range users {
        if i < offset {
            continue
        }
        if len(result) >= limit {
            break
        }
        result = append(result, UserData{
            ID:    user.Id.String(),
            Name:  user.Name,
            Email: user.Email,
        })
    }
    
    return result, nil
}
```

## Testing Patterns

### Unit Testing with Mocks

```go
package services_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/mock"
    "your-project/services"
)

// Mock the interface, not the repository
type MockUserService struct {
    mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error) {
    args := m.Called(ctx, params)
    return args.Get(0).(*repositories.Users), args.Error(1)
}

func (m *MockUserService) GetActiveUsers(ctx context.Context) ([]repositories.Users, error) {
    args := m.Called(ctx)
    return args.Get(0).([]repositories.Users), args.Error(1)
}

func TestSomeBusinessLogic(t *testing.T) {
    mockUserService := new(MockUserService)
    
    // Setup expectations
    mockUserService.On("GetActiveUsers", mock.Anything).Return([]repositories.Users{}, nil)
    
    // Test your code that depends on UserService interface
    // ...
}
```

### Integration Testing

```go
package services_test

import (
    "testing"
    "github.com/nhalm/pgxkit"
    "your-project/repositories"
    "your-project/services"
)

func TestUserService_Integration(t *testing.T) {
    // Get test database connection
    conn := pgxkit.RequireTestDB(t, func(db *pgxkit.DB) interface{} {
        return db // Adjust based on your generated repositories
    })
    
    // Clean up test data
    pgxkit.CleanupTestData(conn,
        "DELETE FROM profiles WHERE user_id IS NOT NULL",
        "DELETE FROM users WHERE email LIKE 'test_%'",
    )
    
    // Create service with real repositories
    userRepo := repositories.NewUsersRepository(conn)
    profileRepo := repositories.NewProfilesRepository(conn)
    userService := services.NewUserService(userRepo, profileRepo)
    
    // Test business logic
    user, err := userService.CreateUserWithProfile(context.Background(), 
        repositories.CreateUsersParams{
            Name:  "Test User",
            Email: "test_user@example.com",
        },
        "Test bio",
    )
    
    require.NoError(t, err)
    assert.Equal(t, "Test User", user.Name)
    
    // Verify profile was created
    // ... additional assertions
}
```

## Key Benefits

### For Teams
- **Interface-driven design** - define contracts that match your domain
- **Incremental adoption** - start with embedding, add extensions as needed
- **Testing flexibility** - mock interfaces, not repositories
- **Clean architecture** - business logic separated from data access

### For Generated Code
- **Designed for embedding** - repositories work perfectly as embedded structs
- **Shared utilities** - consistent patterns for custom extensions
- **Zero runtime overhead** - all composition happens at compile time
- **Full type safety** - Go's type system ensures correctness

### For Maintenance
- **Regeneration-safe** - custom code unaffected by repository regeneration
- **Consistent patterns** - same utilities used in generated and custom code
- **Easy refactoring** - interfaces make it easy to change implementations
- **Clear separation** - database concerns separated from business logic

This approach gives you the productivity of generated code with the flexibility to implement complex business requirements. 