-- Migrasi untuk menambahkan kolom data dan is_base64 ke tabel ssh_logs
-- Menambahkan kolom untuk menyimpan data lengkap SSH log dalam format base64

USE tunnel;

-- Tambahkan kolom data untuk menyimpan output/input lengkap
ALTER TABLE ssh_logs ADD COLUMN data TEXT AFTER command;

-- Tambahkan kolom is_base64 untuk menandai apakah data di-encode base64
ALTER TABLE ssh_logs ADD COLUMN is_base64 BOOLEAN DEFAULT FALSE AFTER data;

-- Verifikasi struktur tabel setelah migrasi
DESCRIBE ssh_logs;