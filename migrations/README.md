# Panduan menggunakan migration serta query SQL dasar
by: Achul

## 01. Migration Up
Kalau mau migration Up

```bash
cd 'C:\magang\piposmart\backend_crm_piposmart\migrations' #command prompt haruslah di folder migrations
goose mysql "root:root@tcp(localhost:3306)/crm_piposmart?parseTime=true" up
```

---

## 02. Migration Down
Kalau mau migration Down

```bash
goose mysql "root:root@tcp(localhost:3306)/crm_piposmart?parseTime=true" down
```

---

## 03. Migration Reset
Kalau mau migration reset

```bash
goose mysql "root:root@tcp(localhost:3306)/crm_piposmart?parseTime=true" reset
```

---

## 04. Create Migration
Kalau mau membuat migration baru

```bash
goose create create_new_migration sql #ganti create_new_migration menjadi nama migration (biasanya nama table)
```

---

## 05. Create Table
Cara membuat table baru menggunakan migration

```sql
CREATE TABLE table_a (
    -- Kolom primary key
    id                  BIGINT AUTO_INCREMENT PRIMARY KEY,

    -- Isi kolom-kolom tabelnya berdasarkan tipe datanya, sebagai contoh:
    data_angka          INTEGER,
    data_string         VARCHAR(255),
    data_waktu          DATETIME,

    -- jika kolomnya berupa data yang unique
    data_angka          INTEGER UNIQUE,
    data_string         VARCHAR(255) UNIQUE,

    -- jika kolomnya berupa data yang tidak boleh kosong atau not nullable
    data_angka          INTEGER NOT NULL,
    data_string         VARCHAR(255) NOT NULL, 

    -- jika ingin menambahkan foreign key
    data_foreign_key     BIGINT UNSIGNED NOT NULL,

    CONSTRAINT fk_foreign_key
        FOREIGN KEY (data_foreign_key)
        REFERENCES table_b(id),

    -- jika ingin menambahkan data index
    data_index          VARCHAR(255),

    INDEX idx_data_index(data_index),

    -- jika ingin menambahkan data timestamp
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

---

## 06. Remove Table
Cara menghapus table

```sql
DROP TABLE table_a;
```

---

## 07. Remove Table Column
Cara menghapus suatu kolom table

```sql
ALTER TABLE table_a
DROP COLUMN data_kolom_dihapus;
```

---

## 08. Rename Table Column
Cara mengubah nama suatu kolom table

```sql
ALTER TABLE table_a
RENAME COLUMN nama_column_lama TO nama_column_baru;
```

---

## 09. Menambahkan Column
Jika ingin menambahkan column baru pada table

```sql
-- Sebelum:
-- table_a
-- |- id    BIGINT  PRIMARY_KEY

ALTER TABLE table_a
ADD COLUMN name VARCHAR(50);

-- Setelah:
-- table_a
-- |- id    BIGINT      PRIMARY_KEY
-- |- name  VARCHAT(50)
```

---

## 10. Mengubah Column Dengan Tipe Data Baru
Mengubah tipe data dari suatu column. Ini fitur sebenarnya tidak direkomendasi
jika ingin mengubah tipe dari dari VARCHAR menjadi INT, tidak direkomendasikan,.
lebih baik digunakan contohnya untuk VARCHAR(20) menjadi VARCHAR(30)

```sql

-- Sebelumnya
-- table_a
-- |- column_yang_ingin_diubah VARCHAR(20)

ALTER TABLE table_a
MODIFY COLUMN column_yang_ingin_diubah VARCHAR(30)

-- Sesudah
-- table_a
-- |- column_yang_ingin_diubah VARCHAR(30)
```

---

## 11. Mengubah Column Dengan Memiliki Index
Mengubah column yang sudah ada (telah dibuat sebelumnya) menjadi memiliki index

```sql
-- Sebelum:
-- table_a
-- |- email VARCHAR(100)

CREATE INDEX idx_table_a_email
ON table_a(email)

-- Sesudah:
-- table_a
-- |- email VARCHAR(100)    INDEX
```

Jika ingin membatalkan perubahannya, maka dapat jalankan sql query,

```sql
-- Sebelum:
-- table_a
-- |- email VARCHAR(100)    INDEX

DROP INDEX idx_table_a_email
ON table_a;

-- Setelah:
-- table_a
-- |- email VARCHAR(100)
```

---

## 12. Mengubah Column Dengan Menjadi Unique
Mengubah Column yang sudah ada (telah dibuat sebelumnya) menjadi column yang unique

```sql
-- Sebelum:
-- table_a
-- |- telephone VARCHAR(100)

ALTER TABLE table_a
ADD CONSTRAINT uq_table_a_telephone
UNIQUE(email);

-- Setelah:
-- table_a
-- |- telephone VARCHAR(100)    UNIQUE
```

Jika ingin membatalkan perubahannya:

```sql
-- Sebelum:
-- table_a
-- |- telephone VARCHAR(100)    UNIQUE

ALTER TABLE table_a
DROP INDEX uq_table_a_telephone;

-- Setelah:
-- table_a
-- |- telephone VARCHAR(100)
```

---

## 13. Mengubah Column Dengan Menjadi Foreign Key
Mengubah Column yang sudah ada (telah dibuat sebelumnya) menjadi column berupa foreign key

```sql
-- Sebelum
-- table_a
-- |- sales_id  BIGINT

ALTER TABLE table_a
ADD CONSTRAINT fk_table_a_sales
FOREIGN KEY (sales_id)
REFERENCES sales(id)    -- nama_table_foreign_key(nama_kolom_id_tablenya), contoh user(id)

-- Setelah
-- table_a
-- |- sales_id  BIGINT FOREIGN KEY (sales(id))
```

Jika ingin membatalkan perubahannya:

```sql
-- Sebelum
-- table_a
-- |- sales_id  BIGINT FOREIGN KEY (sales(id))

ALTER TABLE table_a
DROP FOREIGN KEY fk_table_a_sales

-- Setelah
-- table_a
-- |- sales_id  BIGINT
```

---

## 14. Mengisi Migration untuk pertama kali (create)
Untuk mengisi migration pada table yang pertama kali di buat,
maka mengikuti format seperti ini,

```sql
-- +goose Up
CREATE TABLE tables (
    -- isi kolomnya disini, sebagai contoh:
    id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
    -- ...
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE customers;
```

dan format penamananya seharusnya create_table.sql
menyesuaikan dengan nama tablenya.

---

## 15. Mengisi Migration untuk update selanjutnya 
Jika kedepannya terdapat perubahan kolom atau tablenya maka gunakan format seperti ini

```sql
-- +goose Up
-- Misal aku ingin menambahkan column baru pada table customer
ALTER TABLE customer
ADD COLUMN alamat VARCHAR(255);

-- +goose Down
-- Maka disini berisi kebalikannya yaitu menghapus kolomnya
ALTER TABLE customer
DROP COLUMN alamat;
```

dan format penamaanya seharusnya add_alamat_to_customer.sql
pola penamaan harus menjelaskan menambahkan apa dan ke table apa.

---