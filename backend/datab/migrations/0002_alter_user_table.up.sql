-- Step 1: Create the new table with an additional column
CREATE TABLE IF NOT EXISTS User_New (
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
  CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
  ProfilePrivacy VARCHAR(255)
);

-- Step 2: Copy data from the old table to the new table
INSERT INTO User_New (UserID, Email, PasswordHash, FirstName, LastName, DateOfBirth, ProfilePicture, Nickname, AboutMe, Gender, CreatedAt)
SELECT UserID, Email, PasswordHash, FirstName, LastName, DateOfBirth, ProfilePicture, Nickname, AboutMe, Gender, CreatedAt FROM User;

-- Step 3: Rename the old table and then rename the new table
ALTER TABLE User RENAME TO User_Old;
ALTER TABLE User_New RENAME TO User;

-- Step 4: Drop the old table (be sure everything is correct before doing this)
DROP TABLE User_Old;