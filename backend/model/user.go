package model

import (
	"database/sql"
	"log"
	"time"
)

type User struct {
	UserID         int       `json:"userID"`
	Email          string    `json:"email"`
	Password       string    `json:"-"`
	PasswordHash   string    `json:"-"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	DateOfBirth    time.Time `json:"dateOfBirth"`
	ProfilePicture string    `json:"profilePicture,omitempty"`
	Nickname       string    `json:"nickname,omitempty"`
	AboutMe        string    `json:"aboutMe,omitempty"`
	Gender         string    `json:"gender"`
	CreatedAt      time.Time `json:"createdAt"`
	ProfilePrivacy string    `json:"profilePrivacy"`
}

type UserRelation struct {
	Nickname       string
	RoomID         string
	UserID         int
	UnreadCount    int
	FirstName      string
	LastName       string
	ProfilePicture string `json:"profilePicture"`
}

func RegisterUser(db *sql.DB, user *User) error {
	query := `INSERT INTO User (Email, PasswordHash, FirstName, LastName, DateOfBirth, ProfilePicture, Nickname, AboutMe, Gender, ProfilePrivacy) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query, user.Email, user.PasswordHash, user.FirstName, user.LastName, user.DateOfBirth, user.ProfilePicture, user.Nickname, user.AboutMe, user.Gender, user.ProfilePrivacy)
	if err != nil {
		log.Printf("Failed to insert user data to datab: %v", err)
		return err
	}
	log.Println("Inserted user data to datab")
	return nil
}

func GetUserByCredential(db *sql.DB, credential string) (*User, error) {
	var user User
	query := `SELECT UserID, Email, PasswordHash, FirstName, LastName, DateOfBirth, ProfilePicture, Nickname, AboutMe, Gender, CreatedAt, ProfilePrivacy
	FROM User 
	WHERE Email = ? OR Nickname = ?`
	err := db.QueryRow(query, credential, credential).Scan(
		&user.UserID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.DateOfBirth, &user.ProfilePicture, &user.Nickname, &user.AboutMe, &user.Gender, &user.CreatedAt, &user.ProfilePrivacy,
	)
	if err != nil {
		log.Printf("Error querying user with credential '%s': %v", credential, err)
		return nil, err
	}
	return &user, nil
}

func UserExists(db *sql.DB, email, nickname string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM User WHERE Email = ? OR Nickname = ?)`
	err := db.QueryRow(query, email, nickname).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func GetUserFollowRelations(db *sql.DB, userID int) (map[int]UserRelation, error) {
	userMap := make(map[int]UserRelation)

	// Updated query to fetch user relations even if no room exists
	query := `
	SELECT u.UserID, u.Nickname, u.FirstName, u.LastName, u.ProfilePicture,
				 r.RoomID, 
				 IFNULL((SELECT COUNT(*) FROM Message m WHERE m.RoomID = r.RoomID AND m.ReceiverUserID = ? AND m.Read = FALSE), 0) AS UnreadCount
	FROM User u
	LEFT JOIN Rooms r ON (u.UserID = r.User1ID OR u.UserID = r.User2ID) AND (r.User1ID = ? OR r.User2ID = ?)
	WHERE u.UserID IN (SELECT FollowingUserID FROM UserFollowers WHERE FollowerUserID = ?)
	OR u.UserID IN (SELECT FollowerUserID FROM UserFollowers WHERE FollowingUserID = ?)
	`

	rows, err := db.Query(query, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var nickname, firstName, lastName, profilePicture string
		var roomID sql.NullString // Use sql.NullString to handle NULL values
		var unreadCount int

		// Scan the datab row into the variables
		if err := rows.Scan(&id, &nickname, &firstName, &lastName, &profilePicture, &roomID, &unreadCount); err != nil {
			return nil, err
		}

		// Convert sql.NullString to string, if not NULL
		var roomIDString string
		if roomID.Valid {
			roomIDString = roomID.String
		} else {
			roomIDString = "" // or some default value that makes sense in your application
		}

		userMap[id] = UserRelation{
			Nickname:       nickname,
			FirstName:      firstName,
			LastName:       lastName,
			RoomID:         roomIDString, // Use the converted string
			UserID:         id,
			UnreadCount:    unreadCount,
			ProfilePicture: profilePicture,
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return userMap, nil
}

func GetFollowRelationships(db *sql.DB) (map[int][]int, map[int][]int, error) {
	followingMap := make(map[int][]int) // Key: UserID, Value: []FollowingUserID
	followersMap := make(map[int][]int) // Key: UserID, Value: []FollowerUserID

	query := `
	SELECT FollowerUserID, FollowingUserID
	FROM UserFollowers
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var followerUserID, followingUserID int
		if err := rows.Scan(&followerUserID, &followingUserID); err != nil {
			return nil, nil, err
		}
		// Update followingMap
		followingMap[followerUserID] = append(followingMap[followerUserID], followingUserID)
		// Update followersMap
		followersMap[followingUserID] = append(followersMap[followingUserID], followerUserID)
	}

	return followingMap, followersMap, nil
}

func GetPendingFollowRequests(db *sql.DB, userID int) (map[int]string, error) {
	pendingRequestsMap := make(map[int]string) // UserID -> "pending" for outgoing requests

	query := `
	SELECT FollowingUserID
	FROM FollowRequests
	WHERE FollowerUserID = ? AND Accepted = FALSE
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var followingUserID int
		if err := rows.Scan(&followingUserID); err != nil {
			return nil, err
		}
		pendingRequestsMap[followingUserID] = "pending"
	}

	return pendingRequestsMap, nil
}

func FetchAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`SELECT UserID, FirstName, LastName, ProfilePicture, ProfilePrivacy FROM User`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		// Corrected: Removed user.Email from the Scan method
		if err := rows.Scan(&user.UserID, &user.FirstName, &user.LastName, &user.ProfilePicture, &user.ProfilePrivacy); err != nil {
			// Adding detailed error log
			log.Printf("Error scanning user row: %v", err)
			return nil, err
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		// Adding detailed error log
		log.Printf("Error iterating through user rows: %v", err)
		return nil, err
	}

	return users, nil
}

func GetUserDetails(db *sql.DB, userID int) (string, string, error) {
	var firstName, lastName string
	query := `SELECT FirstName, LastName FROM User WHERE UserID = ?`
	row := db.QueryRow(query, userID)
	err := row.Scan(&firstName, &lastName)
	if err != nil {
		log.Println("Error fetching user details:", err)
		return "", "", err
	}
	return firstName, lastName, nil
}

func FollowUser(db *sql.DB, followerId, followingId int) error {
	_, err := db.Exec("INSERT INTO UserFollowers (FollowerUserID, FollowingUserID) VALUES (?, ?)", followerId, followingId)
	return err
}

func UnfollowUser(db *sql.DB, followerId, followingId int) error {
	_, err := db.Exec("DELETE FROM UserFollowers WHERE FollowerUserID = ? AND FollowingUserID = ?", followerId, followingId)
	return err
}
