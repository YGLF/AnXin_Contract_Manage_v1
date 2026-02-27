CREATE DATABASE IF NOT EXISTS contract_manage CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE contract_manage;

GRANT ALL PRIVILEGES ON contract_manage.* TO 'contract_user'@'%';
FLUSH PRIVILEGES;