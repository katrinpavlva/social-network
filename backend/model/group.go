package model

import (
	"database/sql"
	"log"
	"time"
)

type Group struct {
	GroupID       int    `json:"groupId"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	CreatorUserID int    `json:"creatorUserId"`
}

type GroupMemberRelation struct {
	UserID         int    `json:"userID"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	ProfilePicture string `json:"profilePicture"`
}

type Event struct {
	EventID       int
	GroupID       int
	Title         string
	Description   string
	EventDateTime time.Time
	CreatorID     int
	CreatedAt     time.Time
	FirstName     string
	LastName      string
	Going         []string `json:"Going"`
	NotGoing      []string `json:"NotGoing"`
}

type GroupJoinRequest struct {
	UserID  int `json:"userId"`
	GroupID int `json:"groupId"`
}

type GroupLeaveRequest struct {
	UserID  int `json:"userId"`
	GroupID int `json:"groupId"`
}

type EventCreationRequest struct {
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	EventDateTime    time.Time `json:"dateTime"`
	InvitedMemberIDs []int     `json:"invitedMembers"`
	EventCreatorID   int       `json:"eventCreator"`
	GroupID          int       `json:"groupId"`
}

// CreateGroup inserts a new group into the datab.
func CreateGroup(db *sql.DB, group Group, invitedUserIds []int) (*Group, error) {
	// Prepare the SQL statement for inserting a new group
	statement := `INSERT INTO Cluster (Name, Description, CreatorUserID) VALUES (?, ?, ?)`
	result, err := db.Exec(statement, group.Name, group.Description, group.CreatorUserID)
	if err != nil {
		log.Printf("Error creating group: %v", err)
		return nil, err
	}

	// Get the ID of the newly created group
	groupID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting last insert ID for group: %v", err)
		return nil, err
	}
	group.GroupID = int(groupID)

	// Insert the creator (CreatorID) into the GroupMembers table
	_, err = db.Exec(`INSERT INTO GroupMembers (GroupID, UserID, Accepted) VALUES (?, ?, ?)`, group.GroupID, group.CreatorUserID, true)
	if err != nil {
		log.Printf("Error adding creator (UserID: %d) to group members: %v", group.CreatorUserID, err)
		// Decide how you want to handle the error - rollback group creation, continue with other inserts, etc.
	}

	// Insert invited users into InvitedUsers table
	for _, userID := range invitedUserIds {
		_, err := db.Exec(`INSERT INTO InvitedUsers (GroupID, UserID) VALUES (?, ?)`, group.GroupID, userID)
		if err != nil {
			log.Printf("Error inviting user (ID: %d) to group: %v", userID, err)
			// Decide how you want to handle the error - rollback group creation, continue with other inserts, etc.
		}
	}

	log.Printf("Group created successfully with GroupID: %d", group.GroupID)
	return &group, nil
}

func GetGroups(db *sql.DB) ([]Group, error) {
	var groups []Group

	// Query to select all groups
	query := `SELECT GroupID, Name, Description, CreatorUserID FROM Cluster`
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying for groups: %v", err)
		return nil, err
	}
	defer rows.Close()

	// Iterate over the rows and scan data into the Group struct
	for rows.Next() {
		var group Group
		err := rows.Scan(&group.GroupID, &group.Name, &group.Description, &group.CreatorUserID)
		if err != nil {
			log.Printf("Error scanning group: %v", err)
			return nil, err
		}
		groups = append(groups, group)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over rows: %v", err)
		return nil, err
	}

	return groups, nil
}

func GetGroupByID(db *sql.DB, groupID string) (*Group, error) {
	var group Group
	row := db.QueryRow("SELECT GroupID, Name, Description, CreatorUserID FROM Cluster WHERE GroupID = ?", groupID)
	err := row.Scan(&group.GroupID, &group.Name, &group.Description, &group.CreatorUserID)
	if err != nil {
		log.Printf("Error fetching group by ID: %v", err)
		return nil, err
	}
	return &group, nil
}

func GetGroupMembers(db *sql.DB, groupID int) ([]GroupMemberRelation, error) {
	var members []GroupMemberRelation

	query := `
	SELECT u.UserID, u.FirstName, u.LastName, u.ProfilePicture
	FROM User u
	JOIN GroupMembers gm ON u.UserID = gm.UserID
	WHERE gm.GroupID = ?
	`
	rows, err := db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m GroupMemberRelation
		if err := rows.Scan(&m.UserID, &m.FirstName, &m.LastName, &m.ProfilePicture); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}

func CreateEvent(db *sql.DB, creationReq EventCreationRequest) (Event, error) {
	// Initialize an empty Event struct
	event := Event{
		GroupID:       creationReq.GroupID,
		Title:         creationReq.Title,
		Description:   creationReq.Description,
		EventDateTime: creationReq.EventDateTime,
		CreatorID:     creationReq.EventCreatorID,
		// CreatedAt will be set automatically by the datab
	}

	statement := `INSERT INTO Event (GroupID, Title, Description, EventDateTime, CreatorID) VALUES (?, ?, ?, ?, ?)`
	result, err := db.Exec(statement, event.GroupID, event.Title, event.Description, event.EventDateTime, event.CreatorID)
	if err != nil {
		return Event{}, err
	}

	eventID, err := result.LastInsertId()
	if err != nil {
		return Event{}, err
	}
	event.EventID = int(eventID)

	// Get the CreatedAt time from the datab
	err = db.QueryRow("SELECT CreatedAt FROM Event WHERE EventID = ?", event.EventID).Scan(&event.CreatedAt)
	if err != nil {
		return Event{}, err
	}

	for _, userID := range creationReq.InvitedMemberIDs {
		_, err := db.Exec(`INSERT INTO UserEventResponse (EventID, UserID) VALUES (?, ?)`, event.EventID, userID)
		if err != nil {
			// Handle the error based on your application logic
		}
	}

	return event, nil
}

func GetGroupEvents(db *sql.DB, groupID string) ([]Event, error) {
	// responses := make(map[int]map[string][]string)

	query := `
	SELECT 
			e.EventID, e.GroupID, e.Title, e.Description, e.EventDateTime, e.CreatorID, e.CreatedAt,
			u.FirstName, u.LastName, r.Response
	FROM Event e
	JOIN User u ON e.CreatorID = u.UserID 
	LEFT JOIN UserEventResponse r ON e.EventID = r.EventID AND r.UserID = u.UserID
	WHERE e.GroupID = ?
	ORDER BY e.CreatedAt DESC
	`

	log.Printf("Executing query for GroupID: %s", groupID)

	rows, err := db.Query(query, groupID)
	if err != nil {
		log.Printf("Error querying events for group %s: %v", groupID, err)
		return nil, err
	}
	defer rows.Close()

	eventMap := make(map[int]*Event)
	for rows.Next() {
		var (
			event    Event
			response sql.NullString
		)
		if err := rows.Scan(&event.EventID, &event.GroupID, &event.Title, &event.Description, &event.EventDateTime, &event.CreatorID, &event.CreatedAt, &event.FirstName, &event.LastName, &response); err != nil {
			log.Printf("Error scanning event for group %s: %v", groupID, err)
			continue
		}

		// Check if we've already started building this event's details, if not, create it
		if _, exists := eventMap[event.EventID]; !exists {
			event.Going = []string{}
			event.NotGoing = []string{}
			eventMap[event.EventID] = &event
		}

		// Check if a response exists and add the full name to the appropriate list
		fullName := event.FirstName + " " + event.LastName
		if response.Valid {
			if response.String == "Going" {
				eventMap[event.EventID].Going = append(eventMap[event.EventID].Going, fullName)
			} else if response.String == "Not Going" {
				eventMap[event.EventID].NotGoing = append(eventMap[event.EventID].NotGoing, fullName)
			}
		}
	}

	// Convert the map to a slice
	var events []Event
	for _, event := range eventMap {
		events = append(events, *event)
	}

	if len(events) == 0 {
		log.Printf("No events found for GroupID: %s", groupID)
	}

	return events, nil
}

func JoinGroup(db *sql.DB, joinReq GroupJoinRequest) error {
	// First, find the CreatorUserID for the given GroupID from the Cluster table
	var creatorUserID int
	query := `SELECT CreatorUserID FROM Cluster WHERE GroupID = ?`
	err := db.QueryRow(query, joinReq.GroupID).Scan(&creatorUserID)
	if err != nil {
		log.Printf("Error finding creator user ID from Cluster: %v", err)
		return err
	}

	// Now, insert the new join request into GroupJoinRequests with the found GroupCreatorId
	statement := `INSERT INTO GroupJoinRequests (UserID, GroupID, GroupCreatorId) VALUES (?, ?, ?)`
	_, err = db.Exec(statement, joinReq.UserID, joinReq.GroupID, creatorUserID)
	if err != nil {
		log.Printf("Error inserting join group request into datab: %v", err)
		return err
	}

	return nil
}

func LeaveGroup(db *sql.DB, leaveReq GroupLeaveRequest) error {
	statement := `DELETE FROM GroupMembers WHERE UserID = ? AND GroupID = ?`
	_, err := db.Exec(statement, leaveReq.UserID, leaveReq.GroupID)
	if err != nil {
		log.Printf("Error inserting join group request into datab: %v", err)
		return err
	}
	return nil
}

func GetUserGroupMemberships(db *sql.DB, userID int) (map[int]bool, error) {
	groupsMap := make(map[int]bool)
	query := `
		SELECT GroupID 
		FROM GroupMembers 
		WHERE UserID = ? AND Accepted = TRUE
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var groupID int
		if err := rows.Scan(&groupID); err != nil {
			return nil, err
		}
		// Set true for each group the user is a part of
		groupsMap[groupID] = true
	}

	return groupsMap, nil
}

func GetUserGroupJoinRequests(db *sql.DB, userID int) (map[int]bool, error) {
	requestsMap := make(map[int]bool)
	query := `
		SELECT GroupId 
		FROM GroupJoinRequests 
		WHERE UserId = ?
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var groupID int
		if err := rows.Scan(&groupID); err != nil {
			return nil, err
		}
		// Set true for each group the user has requested to join
		requestsMap[groupID] = true
	}

	return requestsMap, nil
}

func InviteUsersToGroup(db *sql.DB, groupId int, userIds []int) error {
	statement := `INSERT INTO InvitedUsers (GroupID, UserID) VALUES (?, ?)`

	for _, userId := range userIds {
		_, err := db.Exec(statement, groupId, userId)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetInvitedUsers(db *sql.DB, groupId int) ([]User, error) {
	var users []User
	query := `SELECT UserID FROM InvitedUsers WHERE GroupID = ?`
	rows, err := db.Query(query, groupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		users = append(users, User{UserID: userID})
	}
	return users, nil
}
