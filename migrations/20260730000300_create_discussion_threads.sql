-- +goose Up
CREATE TABLE IF NOT EXISTS discussion_threads (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    channel VARCHAR(64) NOT NULL DEFAULT '#tanya-fitur',
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    author_user_id BIGINT UNSIGNED NOT NULL,
    author_name VARCHAR(255) NOT NULL,
    author_role VARCHAR(32) NOT NULL DEFAULT 'SALES',
    tags VARCHAR(255) NOT NULL DEFAULT 'Diskusi',
    is_solved BOOLEAN NOT NULL DEFAULT FALSE,
    likes_count INT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT fk_discussion_threads_user FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_discussion_threads_channel (channel),
    INDEX idx_discussion_threads_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS discussion_replies (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    thread_id BIGINT UNSIGNED NOT NULL,
    author_user_id BIGINT UNSIGNED NOT NULL,
    author_name VARCHAR(255) NOT NULL,
    author_role VARCHAR(32) NOT NULL DEFAULT 'SALES',
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT fk_discussion_replies_thread FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE,
    CONSTRAINT fk_discussion_replies_user FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_discussion_replies_thread (thread_id),
    INDEX idx_discussion_replies_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS discussion_thread_likes (
    thread_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (thread_id, user_id),
    CONSTRAINT fk_discussion_likes_thread FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE,
    CONSTRAINT fk_discussion_likes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS discussion_thread_likes;
DROP TABLE IF EXISTS discussion_replies;
DROP TABLE IF EXISTS discussion_threads;
