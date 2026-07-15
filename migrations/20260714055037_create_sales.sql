-- +goose Up
CREATE TABLE sales(
    id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    nama_sales          VARCHAR(255) NOT NULL,
    kontak_sales        VARCHAR(255) UNIQUE NOT NULL,
    email_sales         VARCHAR(255),
    password            VARCHAR(255) NOT NULL,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE sales;
