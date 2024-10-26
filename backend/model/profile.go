package model

import (
	"database/sql"
	"log"
	"time"
)

type FollowingUser struct {
	UserID       int    `json:"userID"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	RelationType string `json:"relationType"`
}

func FetchPostsByUserID(db *sql.DB, userID int) ([]Post, error) {
	log.Printf("Fetching posts for userID: %d", userID) // Log the start of the operation

	query := `SELECT p.PostID, p.UserID, p.Content, p.ImageURL, p.Timestamp, p.PrivacySetting, p.AllowedViewers,
		u.Nickname, u.FirstName, u.LastName, u.ProfilePicture
		FROM Post p
		JOIN User u ON p.UserID = u.UserID
		WHERE p.UserID = ?`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error executing the query for userID %d: %v", userID, err)
		return nil, err
	}
	defer rows.Close()
	log.Printf("Query executed successfully for userID: %d", userID) // Log successful execution of the query

	var posts []Post

	for rows.Next() {
		var post Post
		var timestamp time.Time
		var groupID sql.NullInt64

		err = rows.Scan(&post.PostID, &post.UserID, &post.Content, &post.ImageURL, &timestamp, &post.PrivacySetting, &post.AllowedViewers,
			&post.Nickname, &post.FirstName, &post.LastName, &post.ProfilePicture)
		if err != nil {
			log.Printf("Error scanning row for userID %d: %v", userID, err)
			return nil, err
		}

		post.Timestamp = timestamp
		if groupID.Valid {
			post.GroupID = groupID
		}

		posts = append(posts, post)
	}

	log.Printf("Successfully fetched %d posts for userID: %d", len(posts), userID) // Log the number of posts fetched

	return posts, nil
}

func FetchFollowingByUserID(db *sql.DB, userID int) ([]FollowingUser, error) {
	query := `SELECT u.UserID, u.FirstName, u.LastName
              FROM UserFollowers uf
              JOIN User u ON uf.FollowingUserID = u.UserID
              WHERE uf.FollowerUserID = ?`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error executing the query: %v", err)
		return nil, err
	}
	defer rows.Close()

	var following []FollowingUser

	for rows.Next() {
		var user FollowingUser
		if err := rows.Scan(&user.UserID, &user.FirstName, &user.LastName); err != nil {
			log.Printf("Error scanning row: %v", err)
			return nil, err
		}
		user.RelationType = "following" // Since this query fetches whom the user is following
		following = append(following, user)
	}

	return following, nil
}

func FetchFollowersByUserID(db *sql.DB, userID int) ([]FollowingUser, error) {
	query := `SELECT u.UserID, u.FirstName, u.LastName
						FROM UserFollowers uf
						JOIN User u ON uf.FollowerUserID = u.UserID
						WHERE uf.FollowingUserID = ?`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error executing the query: %v", err)
		return nil, err
	}
	defer rows.Close()

	var followers []FollowingUser

	for rows.Next() {
		var user FollowingUser
		if err := rows.Scan(&user.UserID, &user.FirstName, &user.LastName); err != nil {
			log.Printf("Error scanning row: %v", err)
			return nil, err
		}
		user.RelationType = "follower" // Since this query fetches who is following the user
		followers = append(followers, user)
	}

	return followers, nil
}
