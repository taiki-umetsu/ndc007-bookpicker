CREATE TABLE IF NOT EXISTS books (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    isbn           VARCHAR(20)   NOT NULL DEFAULT ''  UNIQUE,
    title          VARCHAR(255)  NOT NULL DEFAULT '',
    subtitle       VARCHAR(255)  NOT NULL DEFAULT '',
    authors        VARCHAR(255)  NOT NULL DEFAULT '',
    publisher      VARCHAR(255)  NOT NULL DEFAULT '',
    published_date VARCHAR(10)   NOT NULL DEFAULT '',
    description    TEXT          NOT NULL DEFAULT '',
    book_url       VARCHAR(255)  NOT NULL DEFAULT '',
    image_url      VARCHAR(255)  NOT NULL DEFAULT '',
    created_at     TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);
