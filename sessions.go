package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
)

func createAndDeleteSession(conn *websocket.Conn) {
	createQuery := `mutation createSessions($input: [CreateSessionInput!]!) { createSessions(input: $input) { sessions { id name } } }`
	createVars := `{"input": [{"name": "CreateSession"}]}`

	send(conn, OperationMutation, "createSession", createQuery, createVars)

	var sessionID string
	for {
		message, err := readResponse(conn)
		if err != nil {
			log.Fatalf("Error reading createSessions response: %v", err)
		}
		var wsMsg GraphQLMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Error unmarshalling GraphQLMessage: %v", err)
			continue
		}
		if wsMsg.Type == "next" {
			id, err := parseCreateSessionsResponse(wsMsg.Payload)
			if err != nil {
				log.Printf("Error parsing 'next' payload: %v", err)
				continue
			}
			sessionID = id
			fmt.Printf("createSessions returned id: %s\n", sessionID)
		} else if wsMsg.Type == "complete" {
			break
		}
	}

	if sessionID == "" {
		log.Printf("No session ID found, skipping deleteSession")
		return
	}

	deleteQuery := `mutation($input: [DeleteSessionInput!]!) { deleteSessions(input: $input) { success } }`
	deleteVars := fmt.Sprintf(`{"input": [{"id": "%s"}]}`, sessionID)

	send(conn, OperationMutation, "deleteSession", deleteQuery, deleteVars)

	for {
		message, err := readResponse(conn)
		if err != nil {
			log.Fatalf("Error reading deleteSessions response: %v", err)
		}
		var wsMsg GraphQLMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Error unmarshalling GraphQLMessage: %v", err)
			continue
		}
		if wsMsg.Type == "next" {
			fmt.Printf("deleteSessions response: %s\n", string(wsMsg.Payload))
		} else if wsMsg.Type == "complete" {
			break
		}
	}
}

type CreateSessionsPayload struct {
	Data struct {
		CreateSessions struct {
			Sessions []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"sessions"`
		} `json:"createSessions"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func parseCreateSessionsResponse(payload json.RawMessage) (string, error) {
	if payload == nil || len(payload) == 0 {
		return "", fmt.Errorf("payload is nil or empty")
	}

	var respPayload CreateSessionsPayload
	if err := json.Unmarshal(payload, &respPayload); err != nil {
		return "", fmt.Errorf("error unmarshalling payload into structs: %w", err)
	}

	if len(respPayload.Errors) > 0 {
		return "", fmt.Errorf("GraphQL query returned an error: %s", respPayload.Errors[0].Message)
	}

	if len(respPayload.Data.CreateSessions.Sessions) == 0 {
		return "", fmt.Errorf("sessions array not found or is empty in payload")
	}

	id := respPayload.Data.CreateSessions.Sessions[0].ID
	if id == "" {
		return "", fmt.Errorf("session ID was empty in payload")
	}

	return id, nil
}
