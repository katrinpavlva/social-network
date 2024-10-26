package model

import (
	"database/sql"
	"log"
	"time"
)

type Post struct {
	PostID         int           `json:"postID"`
	UserID         int           `json:"userID"`
	Content        string        `json:"content"`
	ImageURL       string        `json:"imageURL"`
	Timestamp      time.Time     `json:"timestamp"`
	PrivacySetting string        `json:"privacySetting"`
	AllowedViewers string        `json:"allowedViewers"`
	Nickname       string        `json:"nickname"`
	FirstName      string        `json:"firstName"`
	LastName       string        `json:"lastName"`
	ProfilePicture string        `json:"profilePicture"`
	GroupID        sql.NullInt64 `json:"groupID,omitempty"`
}

// CreatePost inserts a new post into the datab and returns the post with user details
func CreatePost(db *sql.DB, post Post) (*Post, error) {
	// Insert the new post into the datab
	statement := `INSERT INTO Post (UserID, Content, PrivacySetting, ImageURL, AllowedViewers, GroupID) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := db.Exec(statement, post.UserID, post.Content, post.PrivacySetting, post.ImageURL, post.AllowedViewers, post.GroupID)
	if err != nil {
		log.Printf("Error creating post with image: %v", err)
		return nil, err
	}

	// Get the ID of the newly created post
	postID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting last insert ID: %v", err)
		return nil, err
	}
	post.PostID = int(postID)

	// Retrieve the full post with user details from the datab
	err = db.QueryRow(`
		SELECT p.PostID, p.UserID, p.Content, p.PrivacySetting, p.ImageURL, p.Timestamp, p.AllowedViewers, p.GroupID,
					u.Nickname, u.FirstName, u.LastName, u.ProfilePicture
		FROM Post p
		JOIN User u ON p.UserID = u.UserID
		WHERE p.PostID = ?`, post.PostID).Scan(
		&post.PostID, &post.UserID, &post.Content, &post.PrivacySetting, &post.ImageURL, &post.Timestamp, &post.AllowedViewers,
		&post.GroupID, // Correctly handle as sql.NullInt64
		&post.Nickname, &post.FirstName, &post.LastName, &post.ProfilePicture,
	)
	if err != nil {
		log.Printf("Error retrieving new post: %v", err)
		return nil, err
	}

	log.Printf("Post with image created successfully with PostID: %d", post.PostID)
	return &post, nil // Return the full post object
}

func GetPosts(db *sql.DB) ([]Post, error) {
	var posts []Post
	query := `SELECT p.PostID, p.UserID, p.Content, p.ImageURL, p.Timestamp, p.PrivacySetting, p.AllowedViewers,
	u.Nickname, u.FirstName, u.LastName, u.ProfilePicture
	FROM Post p
	JOIN User u ON p.UserID = u.UserID
	WHERE p.GroupID IS NULL
	ORDER BY p.Timestamp DESC`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.PostID, &post.UserID, &post.Content, &post.ImageURL, &post.Timestamp, &post.PrivacySetting, &post.AllowedViewers,
			&post.Nickname, &post.FirstName, &post.LastName, &post.ProfilePicture); err != nil {
			log.Printf("Error scanning post: %v", err)
			continue
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func GetGroupPosts(db *sql.DB, groupID string) ([]Post, error) {
	var posts []Post
	query := `SELECT p.PostID, p.UserID, p.Content, p.ImageURL, p.Timestamp, p.PrivacySetting, p.AllowedViewers,
			  u.Nickname, u.FirstName, u.LastName, u.ProfilePicture
			  FROM Post p
			  JOIN User u ON p.UserID = u.UserID
			  WHERE p.GroupID = ?
			  ORDER BY p.Timestamp DESC`

	rows, err := db.Query(query, groupID)
	if err != nil {
		log.Printf("Error querying group posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.PostID, &post.UserID, &post.Content, &post.ImageURL, &post.Timestamp, &post.PrivacySetting, &post.AllowedViewers,
			&post.Nickname, &post.FirstName, &post.LastName, &post.ProfilePicture); err != nil {
			log.Printf("Error scanning group post: %v", err)
			continue
		}
		posts = append(posts, post)
	}

	log.Println("Fetching posts for group %v", groupID)
	return posts, nil
}
