package discussion

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type DBThread struct {
	ID           int64
	Channel      string
	Title        string
	Content      string
	AuthorUserID int64
	AuthorName   string
	AuthorRole   string
	TagsStr      string
	IsSolved     bool
	LikesCount   int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type DBReply struct {
	ID           int64
	ThreadID     int64
	AuthorUserID int64
	AuthorName   string
	AuthorRole   string
	Content      string
	CreatedAt    time.Time
}

func (r *Repository) CreateThread(ctx context.Context, authorID int64, authorName, authorRole, channel, title, content string, tags []string) (int64, error) {
	tagsBytes, _ := json.Marshal(tags)
	tagsStr := string(tagsBytes)

	query := `
		INSERT INTO discussion_threads (channel, title, content, author_user_id, author_name, author_role, tags, likes_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0)
	`
	res, err := r.db.ExecContext(ctx, query, channel, title, content, authorID, authorName, authorRole, tagsStr)
	if err != nil {
		return 0, fmt.Errorf("create thread query: %w", err)
	}

	return res.LastInsertId()
}

func (r *Repository) ListThreads(ctx context.Context, currentUserID int64, channel, query string) ([]DiscussionThreadResponse, error) {
	var whereClause []string
	var args []interface{}

	whereClause = append(whereClause, "t.deleted_at IS NULL")

	if channel != "" && channel != "all" {
		whereClause = append(whereClause, "t.channel = ?")
		args = append(args, channel)
	}

	if query != "" {
		q := "%" + strings.ToLower(query) + "%"
		whereClause = append(whereClause, "(LOWER(t.title) LIKE ? OR LOWER(t.content) LIKE ? OR LOWER(t.author_name) LIKE ? OR LOWER(t.tags) LIKE ?)")
		args = append(args, q, q, q, q)
	}

	whereStmt := strings.Join(whereClause, " AND ")

	sqlStmt := fmt.Sprintf(`
		SELECT 
			t.id, t.channel, t.title, t.content, t.author_user_id, t.author_name, t.author_role, t.tags, t.is_solved, t.likes_count, t.created_at,
			EXISTS(SELECT 1 FROM discussion_thread_likes l WHERE l.thread_id = t.id AND l.user_id = ?) AS is_liked
		FROM discussion_threads t
		WHERE %s
		ORDER BY t.created_at DESC
	`, whereStmt)

	queryArgs := append([]interface{}{currentUserID}, args...)

	rows, err := r.db.QueryContext(ctx, sqlStmt, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list threads query: %w", err)
	}
	defer rows.Close()

	var result []DiscussionThreadResponse
	for rows.Next() {
		var t DBThread
		var isLiked bool
		if err := rows.Scan(
			&t.ID, &t.Channel, &t.Title, &t.Content, &t.AuthorUserID, &t.AuthorName, &t.AuthorRole, &t.TagsStr, &t.IsSolved, &t.LikesCount, &t.CreatedAt,
			&isLiked,
		); err != nil {
			return nil, fmt.Errorf("scan thread row: %w", err)
		}

		var tags []string
		_ = json.Unmarshal([]byte(t.TagsStr), &tags)
		if tags == nil {
			tags = []string{"Diskusi"}
		}

		replies, err := r.GetRepliesByThreadID(ctx, t.ID)
		if err != nil {
			replies = []DiscussionReplyResponse{}
		}

		result = append(result, DiscussionThreadResponse{
			ID:         t.ID,
			Channel:    t.Channel,
			Title:      t.Title,
			AuthorID:   t.AuthorUserID,
			AuthorName: t.AuthorName,
			AuthorRole: t.AuthorRole,
			Content:    t.Content,
			Tags:       tags,
			Likes:      t.LikesCount,
			IsLiked:    isLiked,
			IsSolved:   t.IsSolved,
			CreatedAt:  t.CreatedAt,
			Replies:    replies,
		})
	}

	if result == nil {
		result = []DiscussionThreadResponse{}
	}

	return result, nil
}

func (r *Repository) GetThreadByID(ctx context.Context, threadID, currentUserID int64) (*DiscussionThreadResponse, error) {
	query := `
		SELECT 
			t.id, t.channel, t.title, t.content, t.author_user_id, t.author_name, t.author_role, t.tags, t.is_solved, t.likes_count, t.created_at,
			EXISTS(SELECT 1 FROM discussion_thread_likes l WHERE l.thread_id = t.id AND l.user_id = ?) AS is_liked
		FROM discussion_threads t
		WHERE t.id = ? AND t.deleted_at IS NULL
	`

	var t DBThread
	var isLiked bool
	err := r.db.QueryRowContext(ctx, query, currentUserID, threadID).Scan(
		&t.ID, &t.Channel, &t.Title, &t.Content, &t.AuthorUserID, &t.AuthorName, &t.AuthorRole, &t.TagsStr, &t.IsSolved, &t.LikesCount, &t.CreatedAt,
		&isLiked,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var tags []string
	_ = json.Unmarshal([]byte(t.TagsStr), &tags)
	if tags == nil {
		tags = []string{"Diskusi"}
	}

	replies, err := r.GetRepliesByThreadID(ctx, t.ID)
	if err != nil {
		replies = []DiscussionReplyResponse{}
	}

	return &DiscussionThreadResponse{
		ID:         t.ID,
		Channel:    t.Channel,
		Title:      t.Title,
		AuthorID:   t.AuthorUserID,
		AuthorName: t.AuthorName,
		AuthorRole: t.AuthorRole,
		Content:    t.Content,
		Tags:       tags,
		Likes:      t.LikesCount,
		IsLiked:    isLiked,
		IsSolved:   t.IsSolved,
		CreatedAt:  t.CreatedAt,
		Replies:    replies,
	}, nil
}

func (r *Repository) ToggleLike(ctx context.Context, threadID, userID int64) (bool, int, error) {
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM discussion_thread_likes WHERE thread_id = ? AND user_id = ?)`
	_ = r.db.QueryRowContext(ctx, checkQuery, threadID, userID).Scan(&exists)

	if exists {
		_, _ = r.db.ExecContext(ctx, `DELETE FROM discussion_thread_likes WHERE thread_id = ? AND user_id = ?`, threadID, userID)
		_, _ = r.db.ExecContext(ctx, `UPDATE discussion_threads SET likes_count = GREATEST(0, likes_count - 1) WHERE id = ?`, threadID)
	} else {
		_, _ = r.db.ExecContext(ctx, `INSERT INTO discussion_thread_likes (thread_id, user_id) VALUES (?, ?)`, threadID, userID)
		_, _ = r.db.ExecContext(ctx, `UPDATE discussion_threads SET likes_count = likes_count + 1 WHERE id = ?`, threadID)
	}

	var newLikesCount int
	_ = r.db.QueryRowContext(ctx, `SELECT likes_count FROM discussion_threads WHERE id = ?`, threadID).Scan(&newLikesCount)

	return !exists, newLikesCount, nil
}

func (r *Repository) DeleteThread(ctx context.Context, threadID int64) error {
	query := `UPDATE discussion_threads SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, threadID)
	return err
}

func (r *Repository) AddReply(ctx context.Context, threadID, authorID int64, authorName, authorRole, content string) (int64, error) {
	query := `
		INSERT INTO discussion_replies (thread_id, author_user_id, author_name, author_role, content)
		VALUES (?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, query, threadID, authorID, authorName, authorRole, content)
	if err != nil {
		return 0, fmt.Errorf("add reply query: %w", err)
	}

	return res.LastInsertId()
}

func (r *Repository) GetRepliesByThreadID(ctx context.Context, threadID int64) ([]DiscussionReplyResponse, error) {
	query := `
		SELECT id, thread_id, author_user_id, author_name, author_role, content, created_at
		FROM discussion_replies
		WHERE thread_id = ? AND deleted_at IS NULL
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var replies []DiscussionReplyResponse
	for rows.Next() {
		var rep DBReply
		if err := rows.Scan(&rep.ID, &rep.ThreadID, &rep.AuthorUserID, &rep.AuthorName, &rep.AuthorRole, &rep.Content, &rep.CreatedAt); err != nil {
			return nil, err
		}
		replies = append(replies, DiscussionReplyResponse{
			ID:         rep.ID,
			ThreadID:   rep.ThreadID,
			AuthorID:   rep.AuthorUserID,
			AuthorName: rep.AuthorName,
			AuthorRole: rep.AuthorRole,
			Content:    rep.Content,
			CreatedAt:  rep.CreatedAt,
		})
	}

	if replies == nil {
		replies = []DiscussionReplyResponse{}
	}

	return replies, nil
}

func (r *Repository) GetReplyByID(ctx context.Context, replyID int64) (*DiscussionReplyResponse, error) {
	query := `
		SELECT id, thread_id, author_user_id, author_name, author_role, content, created_at
		FROM discussion_replies
		WHERE id = ? AND deleted_at IS NULL
	`
	var rep DBReply
	err := r.db.QueryRowContext(ctx, query, replyID).Scan(&rep.ID, &rep.ThreadID, &rep.AuthorUserID, &rep.AuthorName, &rep.AuthorRole, &rep.Content, &rep.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &DiscussionReplyResponse{
		ID:         rep.ID,
		ThreadID:   rep.ThreadID,
		AuthorID:   rep.AuthorUserID,
		AuthorName: rep.AuthorName,
		AuthorRole: rep.AuthorRole,
		Content:    rep.Content,
		CreatedAt:  rep.CreatedAt,
	}, nil
}

func (r *Repository) DeleteReply(ctx context.Context, replyID int64) error {
	query := `UPDATE discussion_replies SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, replyID)
	return err
}
