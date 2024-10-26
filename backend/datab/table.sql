CREATE TABLE IF NOT EXISTS User (
  UserID INTEGER PRIMARY KEY AUTOINCREMENT,
  Email VARCHAR(255),
  PasswordHash VARCHAR(255),
  FirstName VARCHAR(255),
  LastName VARCHAR(255),
  DateOfBirth DATE,
  ProfilePicture VARCHAR(255),
  Nickname VARCHAR(255),
  AboutMe TEXT,
  Gender VARCHAR(255),
  CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS Post (
  PostID INTEGER PRIMARY KEY AUTOINCREMENT,
  UserID INTEGER,
  Content TEXT,
  ImageURL VARCHAR(255),
  Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
  PrivacySetting VARCHAR(255),
  AllowedViewers TEXT,
  FOREIGN KEY (UserID) REFERENCES User(UserID)
);

CREATE TABLE IF NOT EXISTS Comment (
  CommentID INTEGER PRIMARY KEY AUTOINCREMENT,
  PostID INTEGER,
  UserID INTEGER,
  Content TEXT,
  CommentMedia VARCHAR(255),
  Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (PostID) REFERENCES Post(PostID),
  FOREIGN KEY (UserID) REFERENCES User(UserID)
);

CREATE TABLE IF NOT EXISTS Cluster (
  GroupID INTEGER PRIMARY KEY AUTOINCREMENT,
  Name VARCHAR(255),
  Description TEXT,
  CreatorUserID INTEGER,
  FOREIGN KEY (CreatorUserID) REFERENCES User(UserID)
);

CREATE TABLE IF NOT EXISTS Message (
  MessageID INTEGER PRIMARY KEY AUTOINCREMENT,
  SenderUserID INTEGER,
  ReceiverUserID INTEGER,
  Content TEXT,
  Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (SenderUserID) REFERENCES User(UserID),
  FOREIGN KEY (ReceiverUserID) REFERENCES User(UserID)
);

CREATE TABLE IF NOT EXISTS Notification (
  NotificationID INTEGER PRIMARY KEY AUTOINCREMENT,
  UserID INTEGER,
  Content TEXT,
  Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
  ReadStatus BOOLEAN,
  FOREIGN KEY (UserID) REFERENCES User(UserID)
);

CREATE TABLE IF NOT EXISTS UserFollowers (
  FollowerUserID INTEGER,
  FollowingUserID INTEGER,
  FOREIGN KEY (FollowerUserID) REFERENCES User(UserID),
  FOREIGN KEY (FollowingUserID) REFERENCES User(UserID)
);

CREATE TABLE IF NOT EXISTS GroupMembers (
  GroupID INTEGER,
  UserID INTEGER,
  Accepted BOOLEAN DEFAULT FALSE,
  FOREIGN KEY (GroupID) REFERENCES Cluster(GroupID),
  FOREIGN KEY (UserID) REFERENCES User(UserID)
);