package main

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"social-network/backend/chat"
	"social-network/backend/datab"
	"social-network/backend/handler"
	"social-network/backend/model"
)

func main() {

	db, err := datab.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to the datab: %v", err)
	}
	defer db.Close()

	err = datab.CreateTables(db)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	storageClient, err := storage.NewClient(context.Background(), option.WithCredentialsFile("datab/private/social-network-KEY.json"))
	if err != nil {
		log.Fatalf("Failed to create storage client: %v", err)
	}

	go model.CleanExpiredSessions(db)

	wsServer := chat.NewWSServer()
	go wsServer.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		chat.ServeWs(db, wsServer, w, r)
	})

	http.HandleFunc("/api/register", handler.RegisterH(db, storageClient, "social-network-bucket"))
	http.HandleFunc("/api/login", handler.LoginH(db))
	http.HandleFunc("/api/logout", handler.LogoutH(db))

	http.HandleFunc("/api/createPost", handler.CreatePH(db, storageClient, "social-network-bucket"))
	http.HandleFunc("/api/posts", handler.GetPH(db))

	http.HandleFunc("/api/createComment", handler.CrComHandler(db, storageClient, "social-network-bucket"))
	http.HandleFunc("/api/getComments", handler.GePostComH(db))

	http.HandleFunc("/api/createGroup", handler.CreateGrH(db))
	http.HandleFunc("/api/groups", handler.GetGrH(db))
	http.HandleFunc("/api/group/details", handler.FetchGrDetailH(db))
	http.HandleFunc("/api/groupMembers", handler.FetchGrMemH(db))
	http.HandleFunc("/api/createEvent", handler.CreateEvH(db))
	http.HandleFunc("/api/events", handler.GetEvH(db))
	http.HandleFunc("/api/joinGroup", handler.JoinGrH(db))
	http.HandleFunc("/api/leaveGroup", handler.LeaveGrH(db))
	http.HandleFunc("/api/inviteUsers", handler.InviteUserH(db))
	http.HandleFunc("/api/invitedUsers", handler.GetInvUserH(db))

	http.HandleFunc("/api/users", handler.FetchUseH(db))
	http.HandleFunc("/api/following", handler.FollowH(db))
	http.HandleFunc("/api/profilePosts", handler.GetUserPH(db))
	http.HandleFunc("/api/userFollowing", handler.GetFollowH(db))
	http.HandleFunc("/api/userDetails", handler.GetUserDetH(db))
	http.HandleFunc("/api/toggleProfilePrivacy", handler.ToggleProPrivH(db))

	http.Handle("/", http.FileServer(http.Dir("frontend/dist")))

	url := "http://localhost:8091"
	fmt.Println("Listening on", url)

	http.ListenAndServe(":8091", nil)
	fmt.Println("Listening on :8091...")
}


