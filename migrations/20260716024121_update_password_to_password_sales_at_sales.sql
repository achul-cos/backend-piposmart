-- +goose Up
ALTER TABLE sales
RENAME COLUMN password TO password_sales;

-- +goose Down
RENAME COLUMN password_sales TO password;
