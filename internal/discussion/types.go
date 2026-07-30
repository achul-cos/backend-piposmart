package discussion

import "time"

type DiscussionReplyResponse struct {
	ID         int64     `json:"id"`
	ThreadID   int64     `json:"thread_id"`
	AuthorID   int64     `json:"author_id"`
	AuthorName string    `json:"authorName"`
	AuthorRole string    `json:"authorRole"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"createdAt"`
}

type DiscussionThreadResponse struct {
	ID         int64                     `json:"id"`
	Channel    string                    `json:"channel"`
	Title      string                    `json:"title"`
	AuthorID   int64                     `json:"author_id"`
	AuthorName string                    `json:"authorName"`
	AuthorRole string                    `json:"authorRole"`
	Content    string                    `json:"content"`
	Tags       []string                  `json:"tags"`
	Likes      int                       `json:"likes"`
	IsLiked    bool                      `json:"isLiked"`
	IsSolved   bool                      `json:"solved"`
	CreatedAt  time.Time                 `json:"createdAt"`
	Replies    []DiscussionReplyResponse `json:"replies"`
}

type CreateThreadRequest struct {
	Channel string   `json:"channel" binding:"required"`
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type CreateReplyRequest struct {
	Content string `json:"content" binding:"required"`
}
