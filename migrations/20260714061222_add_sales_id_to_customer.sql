-- +goose Up
ALTER TABLE customers
ADD COLUMN sales_id BIGINT UNSIGNED,

ADD CONSTRAINT fk_customers_sales
FOREIGN KEY (sales_id)
REFERENCES sales(id);

-- +goose Down
ALTER TABLE customers
DROP FOREIGN KEY fk_customers_sales,

DROP COLUMN sales_id;
