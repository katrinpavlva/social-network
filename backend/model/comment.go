package model

import (
	"database/sql"
	"log"
	"time"
)

type Comment struct {
	CommentID      int       `json:"commentID"`
	PostID         int       `json:"postID"`
	UserID         int       `json:"userID"`
	Content        string    `json:"content"`
	Timestamp      time.Time `json:"timestamp"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	ProfilePicture string    `json:"profilePicture,omitempty"`
	CommentMedia   string    `json:"commentMedia,omitempty"`
}

func CreateComment(db *sql.DB, comment Comment) (*Comment, error) {
	// Insert the new comment into the datab
	statement := `INSERT INTO Comment (PostID, UserID, Content, CommentMedia) VALUES (?, ?, ?, ?)`
	result, err := db.Exec(statement, comment.PostID, comment.UserID, comment.Content, comment.CommentMedia)
	if err != nil {
		log.Printf("Error creating comment: %v", err)
		return nil, err
	}

	// Get the ID of the newly created comment
	commentID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting last insert ID: %v", err)
		return nil, err
	}
	comment.CommentID = int(commentID)

	// Retrieve the full comment along with user data from the datab
	err = db.QueryRow(`
	SELECT c.CommentID, c.PostID, c.UserID, c.Content, c.Timestamp, c.CommentMedia,
				 u.FirstName, u.LastName, u.ProfilePicture
	FROM Comment c
	JOIN User u ON c.UserID = u.UserID
	WHERE c.CommentID = ?`, comment.CommentID).Scan(
		&comment.CommentID, &comment.PostID, &comment.UserID, &comment.Content,
		&comment.Timestamp, &comment.CommentMedia, &comment.FirstName, &comment.LastName, &comment.ProfilePicture)
	if err != nil {
		log.Printf("Error retrieving new comment with user data: %v", err)
		return nil, err
	}

	log.Printf("Comment created successfully with CommentID: %d", comment.CommentID)
	return &comment, nil
}

func GetCommentsForPost(db *sql.DB, postID string) ([]Comment, error) {
	var comments []Comment

	// Updated SQL query to join Comment and User tables
	query := `
	SELECT c.CommentID, c.PostID, c.UserID, c.Content, c.Timestamp, c.CommentMedia,
				 u.FirstName, u.LastName, u.ProfilePicture
	FROM Comment c
	JOIN User u ON c.UserID = u.UserID
	WHERE c.PostID = ?
	ORDER BY c.Timestamp DESC`

	rows, err := db.Query(query, postID)
	if err != nil {
		log.Printf("Error querying comments: %v", err)
		return nil, err
	}
	defer rows.Close()

	// Iterate over the rows and scan data into the Comment struct
	for rows.Next() {
		var comment Comment
		if err := rows.Scan(&comment.CommentID, &comment.PostID, &comment.UserID, &comment.Content,
			&comment.Timestamp, &comment.CommentMedia, &comment.FirstName, &comment.LastName, &comment.ProfilePicture); err != nil {
			log.Printf("Error scanning comment with user data: %v", err)
			continue
		}
		comments = append(comments, comment)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}
