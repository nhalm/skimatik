// Package service implements the example-app's business logic layer, mediating
// between the HTTP handlers in api and the repository layer.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nhalm/pgxkit/v2"
	"github.com/nhalm/skimatik/v2/example-app/internal/repository/generated"
)

// BlogService demonstrates transactional operations using the per-method executor pattern.
// Each repository method accepts a pgxkit.Executor, allowing the same repository instance
// to work with either *pgxkit.DB (normal operations) or *pgxkit.Tx (transactional operations).
type BlogService struct {
	db           *pgxkit.DB
	usersRepo    *generated.UsersRepository
	postsRepo    *generated.PostsRepository
	commentsRepo *generated.CommentsRepository
}

func NewBlogService(
	db *pgxkit.DB,
	usersRepo *generated.UsersRepository,
	postsRepo *generated.PostsRepository,
	commentsRepo *generated.CommentsRepository,
) *BlogService {
	return &BlogService{
		db:           db,
		usersRepo:    usersRepo,
		postsRepo:    postsRepo,
		commentsRepo: commentsRepo,
	}
}

// CreatePostWithInitialComment demonstrates transactional usage with skimatik-generated repositories.
// It creates a post and its first comment atomically - if either operation fails, both are rolled back.
//
// This showcases the per-method executor pattern where:
//   - Each repository method accepts a pgxkit.Executor parameter
//   - The same repository instance works with *pgxkit.DB or *pgxkit.Tx
//   - No need for separate "transactional repository" types
func (s *BlogService) CreatePostWithInitialComment(
	ctx context.Context,
	authorID uuid.UUID,
	title string,
	content string,
	initialComment string,
) (*generated.Posts, *generated.Comments, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	post, err := s.postsRepo.Create(ctx, tx, generated.CreatePostsParams{
		Title:    title,
		Content:  content,
		AuthorId: authorID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create post: %w", err)
	}

	comment, err := s.commentsRepo.Create(ctx, tx, generated.CreateCommentsParams{
		PostId:   post.Id,
		AuthorId: authorID,
		Content:  initialComment,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create initial comment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit transaction: %w", err)
	}

	return post, comment, nil
}

// TransferPostOwnership demonstrates a multi-step transaction that validates
// and updates data across tables. It transfers a post from one user to another,
// ensuring both users exist before making the transfer.
func (s *BlogService) TransferPostOwnership(
	ctx context.Context,
	postID uuid.UUID,
	fromUserID uuid.UUID,
	toUserID uuid.UUID,
) (*generated.Posts, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fromUser, err := s.usersRepo.Get(ctx, tx, fromUserID)
	if err != nil {
		return nil, fmt.Errorf("get source user: %w", err)
	}
	if !fromUser.IsActive {
		return nil, fmt.Errorf("source user is not active")
	}

	toUser, err := s.usersRepo.Get(ctx, tx, toUserID)
	if err != nil {
		return nil, fmt.Errorf("get target user: %w", err)
	}
	if !toUser.IsActive {
		return nil, fmt.Errorf("target user is not active")
	}

	post, err := s.postsRepo.Get(ctx, tx, postID)
	if err != nil {
		return nil, fmt.Errorf("get post: %w", err)
	}
	if post.AuthorId != fromUserID {
		return nil, fmt.Errorf("post does not belong to source user")
	}

	updatedPost, err := s.postsRepo.Update(ctx, tx, postID, generated.UpdatePostsParams{
		Title:       post.Title,
		Content:     post.Content,
		AuthorId:    toUserID,
		IsPublished: post.IsPublished,
		PublishedAt: post.PublishedAt,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		ViewCount:   post.ViewCount,
	})
	if err != nil {
		return nil, fmt.Errorf("update post ownership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return updatedPost, nil
}

// DeleteUserWithContent demonstrates cascading deletes within a transaction.
// It removes all of a user's comments, posts, and finally the user account.
// If any step fails, all changes are rolled back.
func (s *BlogService) DeleteUserWithContent(
	ctx context.Context,
	userID uuid.UUID,
) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = s.usersRepo.Get(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	comments, err := s.commentsRepo.List(ctx, tx)
	if err != nil {
		return fmt.Errorf("list comments: %w", err)
	}
	for i := range comments {
		comment := &comments[i]
		if comment.AuthorId == userID {
			if err := s.commentsRepo.Delete(ctx, tx, comment.Id); err != nil {
				return fmt.Errorf("delete comment %s: %w", comment.Id, err)
			}
		}
	}

	posts, err := s.postsRepo.List(ctx, tx)
	if err != nil {
		return fmt.Errorf("list posts: %w", err)
	}
	for i := range posts {
		post := &posts[i]
		if post.AuthorId == userID {
			if err := s.postsRepo.Delete(ctx, tx, post.Id); err != nil {
				return fmt.Errorf("delete post %s: %w", post.Id, err)
			}
		}
	}

	if err := s.usersRepo.Delete(ctx, tx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
