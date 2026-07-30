package discussion

import (
	"context"
	"errors"
	"fmt"

	"backend_crm_piposmart/internal/identity"
)

var (
	ErrThreadNotFound     = errors.New("thread tidak ditemukan")
	ErrReplyNotFound      = errors.New("balasan tidak ditemukan")
	ErrUnauthorizedDelete = errors.New("anda tidak memiliki akses untuk menghapus data ini")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateThread(ctx context.Context, user identity.User, req CreateThreadRequest) (*DiscussionThreadResponse, error) {
	authorRole := user.RoleCode
	if authorRole == "" {
		authorRole = "SALES"
	}

	threadID, err := s.repo.CreateThread(ctx, user.ID, user.Name, authorRole, req.Channel, req.Title, req.Content, req.Tags)
	if err != nil {
		return nil, fmt.Errorf("service create thread: %w", err)
	}

	return s.repo.GetThreadByID(ctx, threadID, user.ID)
}

func (s *Service) ListThreads(ctx context.Context, user identity.User, channel, query string) ([]DiscussionThreadResponse, error) {
	return s.repo.ListThreads(ctx, user.ID, channel, query)
}

func (s *Service) GetThread(ctx context.Context, user identity.User, threadID int64) (*DiscussionThreadResponse, error) {
	thread, err := s.repo.GetThreadByID(ctx, threadID, user.ID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}
	return thread, nil
}

func (s *Service) ToggleLike(ctx context.Context, user identity.User, threadID int64) (bool, int, error) {
	thread, err := s.repo.GetThreadByID(ctx, threadID, user.ID)
	if err != nil {
		return false, 0, err
	}
	if thread == nil {
		return false, 0, ErrThreadNotFound
	}

	return s.repo.ToggleLike(ctx, threadID, user.ID)
}

func (s *Service) DeleteThread(ctx context.Context, user identity.User, threadID int64) error {
	thread, err := s.repo.GetThreadByID(ctx, threadID, user.ID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	// Permission check: Admin OR Author of thread
	if user.RoleCode != identity.RoleAdmin && thread.AuthorID != user.ID {
		return ErrUnauthorizedDelete
	}

	return s.repo.DeleteThread(ctx, threadID)
}

func (s *Service) AddReply(ctx context.Context, user identity.User, threadID int64, req CreateReplyRequest) (*DiscussionReplyResponse, error) {
	thread, err := s.repo.GetThreadByID(ctx, threadID, user.ID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}

	authorRole := user.RoleCode
	if authorRole == "" {
		authorRole = "SALES"
	}

	replyID, err := s.repo.AddReply(ctx, threadID, user.ID, user.Name, authorRole, req.Content)
	if err != nil {
		return nil, err
	}

	return s.repo.GetReplyByID(ctx, replyID)
}

func (s *Service) DeleteReply(ctx context.Context, user identity.User, replyID int64) error {
	reply, err := s.repo.GetReplyByID(ctx, replyID)
	if err != nil {
		return err
	}
	if reply == nil {
		return ErrReplyNotFound
	}

	// Permission check: Admin OR Author of reply
	if user.RoleCode != identity.RoleAdmin && reply.AuthorID != user.ID {
		return ErrUnauthorizedDelete
	}

	return s.repo.DeleteReply(ctx, replyID)
}
