-- +goose Up
CREATE TABLE customers (
    id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    kode_owner          VARCHAR(50),
    nama_owner          VARCHAR(255),
    nama_brand          VARCHAR(255),
    nama_outlet         VARCHAR(255),
    kontak_owner        VARCHAR(255),
    kontak_outlet       VARCHAR(255),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE customers;
