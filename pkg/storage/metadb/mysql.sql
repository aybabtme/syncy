CREATE DATABASE IF NOT EXISTS syncy;
USE syncy;
DROP TABLE accounts;
DROP TABLE projects;
DROP TABLE dirs;
DROP TABLE files;
DROP TABLE pending_files;

CREATE TABLE accounts (
    `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `public_id` VARCHAR(255) NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (`public_id`)
);

CREATE TABLE projects (
    `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `account_id` BIGINT NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `created_at` TIMESTAMP DEFAULT NOW(),
    UNIQUE (`account_id`, `name`)
);

CREATE TABLE dirs (
    `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `project_id` BIGINT NOT NULL,
    `parent_id` BIGINT,
    `name` VARCHAR(255) NOT NULL,
    `mod_time` BIGINT NOT NULL,
    `mode` INT NOT NULL,
    UNIQUE (`project_id`, `parent_id`, `name`)
);

CREATE TABLE files (
    `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `project_id` BIGINT NOT NULL,
    `dir_id` BIGINT,
    `name` VARCHAR(255) NOT NULL,

    `size` BIGINT NOT NULL,
    `mod_time` BIGINT NOT NULL,
    `mode` INT NOT NULL,

    `blake3_64_256_sum` BINARY(32) NOT NULL,

    UNIQUE (`project_id`, `dir_id`, `name`)
);

CREATE TABLE pending_files (
    `file_id` BIGINT NOT NULL PRIMARY KEY,
    `created_at` TIMESTAMP NOT NULL DEFAULT NOW()
);