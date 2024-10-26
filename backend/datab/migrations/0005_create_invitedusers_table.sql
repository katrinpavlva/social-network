CREATE TABLE IF NOT EXISTS InvitedUsers (
  GroupID INTEGER,
  UserID INTEGER,
  Accepted BOOLEAN DEFAULT 0,
  FOREIGN KEY (GroupID) REFERENCES Cluster(GroupID),
  FOREIGN KEY (UserID) REFERENCES User(UserID)
);