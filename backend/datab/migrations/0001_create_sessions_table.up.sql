-- +migrate Up
CREATE TABLE IF NOT EXISTS Sessions (
    SessionID TEXT PRIMARY KEY,
    UserID INTEGER NOT NULL,
    ExpiresAt DATETIME NOT NULL,
    FOREIGN KEY (UserID ) REFERENCES User(UserID )
);
