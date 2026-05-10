package main

import (
	"database/sql"
	"errors"
	"log"
	"strings"
)

var ErrFriendRequestExists = errors.New("friend request already exists")

func createFriendRequest(playerID, friendID string) error {
	query := "INSERT INTO friend_requests (sender_id, receiver_id) VALUES (?, ?)"

	_, err := db.Exec(query, playerID, friendID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrFriendRequestExists
		}

		log.Printf("Error creating friend request: %v", err)
		return err
	}

	return nil
}

func getPendingFriendRequests(playerID string) []PlayerDTO {
	query := `
		SELECT fr.sender_id, p.username
		FROM friend_requests fr
		JOIN players p ON fr.sender_id = p.id
		WHERE fr.receiver_id = ? AND fr.status = 'pending'
	`

	rows, err := db.Query(query, playerID)
	if err != nil {
		log.Printf("Error fetching friend requests: %v", err)
		return []PlayerDTO{}
	}
	defer rows.Close()

	requests := make([]PlayerDTO, 0)

	for rows.Next() {
		var fr PlayerDTO
		err := rows.Scan(&fr.ID, &fr.Name)
		if err != nil {
			log.Printf("Error scanning friend request: %v", err)
			continue
		}
		requests = append(requests, fr)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
	}

	return requests
}

func handleFriendRequest(senderID, receiverID string, acceptRequest bool) error {
	deleteQuery := `
		DELETE FROM friend_requests
		WHERE sender_id = ? AND receiver_id = ? AND status = 'pending'
	`

	result, err := db.Exec(deleteQuery, senderID, receiverID)
	if err != nil {
		log.Printf("Error deleting friend request: %v", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("friend request not found")
	}

	if !acceptRequest {
		return nil
	}

	firstID := senderID
	secondID := receiverID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	insertQuery := `
		INSERT INTO friend_lists (first_player_id, second_player_id)
		VALUES (?, ?)
	`

	_, err = db.Exec(insertQuery, firstID, secondID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil
		}

		log.Printf("Error creating friendship: %v", err)
		return err
	}

	return nil
}

func getFriendsList(playerID string) []PlayerDTO {
	query := `
		SELECT fl.second_player_id AS friend_id, p.username
		FROM friend_lists fl
		JOIN players p ON p.id = fl.second_player_id
		WHERE fl.first_player_id = ?

		UNION

		SELECT fl.first_player_id AS friend_id, p.username
		FROM friend_lists fl
		JOIN players p ON p.id = fl.first_player_id
		WHERE fl.second_player_id = ?
	`

	rows, err := db.Query(query, playerID, playerID)
	if err != nil {
		log.Printf("Error fetching friends list: %v", err)
		return []PlayerDTO{}
	}
	defer rows.Close()

	friends := make([]PlayerDTO, 0)

	for rows.Next() {
		var p PlayerDTO
		err := rows.Scan(&p.ID, &p.Name)
		if err != nil {
			log.Printf("Error scanning friend: %v", err)
			continue
		}
		friends = append(friends, p)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
	}

	return friends
}

func areFriends(playerID, otherPlayerID string) bool {
	firstID := playerID
	secondID := otherPlayerID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	query := `
		SELECT 1
		FROM friend_lists
		WHERE first_player_id = ? AND second_player_id = ?
	`

	var exists int
	err := db.QueryRow(query, firstID, secondID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		log.Printf("Error checking friendship: %v", err)
		return false
	}

	return true
}

func removeFriend(playerID, friendID string) error { //Falls Chats hinzugefügt werden, muss hier auch die Chat-Beziehung gelöscht werden
	firstID := playerID
	secondID := friendID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	query := `
		DELETE FROM friend_lists
		WHERE first_player_id = ? AND second_player_id = ?
	`

	result, err := db.Exec(query, firstID, secondID)
	if err != nil {
		log.Printf("Error removing friend: %v", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("friendship not found")
	}

	return nil
}
