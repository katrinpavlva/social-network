CREATE TABLE IF NOT EXISTS Message_new (
  MessageID TEXT PRIMARY KEY,
  SenderUserID INTEGER,
  ReceiverUserID INTEGER,
  RoomID TEXT,
  Content TEXT,
  Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
  Read BOOLEAN DEFAULT FALSE,
  FOREIGN KEY (SenderUserID) REFERENCES User(UserID),
  FOREIGN KEY (ReceiverUserID) REFERENCES User(UserID),
  FOREIGN KEY (RoomID) REFERENCES Rooms(RoomID)
);

ALTER TABLE Message RENAME TO Message_Old;
ALTER TABLE Message_new RENAME TO Message;

DROP TABLE Message_Old;
