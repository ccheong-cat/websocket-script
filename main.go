package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type GraphQLMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Id      uuid.UUID              `json:"id,omitempty"`
}

type OperationType string

const (
	OperationSubscription OperationType = "subscription"
	OperationMutation     OperationType = "mutation"
)

func parseVariables(jsonStr string) (map[string]interface{}, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var variables map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &variables); err != nil {
		return nil, err
	}
	return variables, nil
}

func NewGraphQLMessage(opType OperationType, query string, variables map[string]interface{}) (GraphQLMessage, error) {
	if opType == OperationSubscription && variables != nil {
		return GraphQLMessage{}, fmt.Errorf("subscriptions cannot have variables")
	}
	id := uuid.New()
	payload := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}
	return GraphQLMessage{
		Id:      id,
		Type:    "subscribe",
		Payload: payload,
	}, nil
}

func send(conn *websocket.Conn, opType OperationType, prefix, query string, variables string) {
	parsedVars, err := parseVariables(variables)
	if err != nil {
		log.Fatalf("Error parsing variables for %s: %v", prefix, err)
	}
	msg, err := NewGraphQLMessage(opType, query, parsedVars)
	if err != nil {
		log.Fatalf("Error creating GraphQL message for %s: %v", prefix, err)
	}
	if err := conn.WriteJSON(msg); err != nil {
		log.Fatalf("Error sending %s: %v", opType, err)
	}
	fmt.Printf("%s: Sent [%s] %s with id %s\n", time.Now().Format(time.RFC3339), prefix, opType, msg.Id.String())

	if opType == OperationSubscription {
		return
	}
	// For mutations, log the next response from the server
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("Error reading response for %s: %v", prefix, err)
		return
	}
	fmt.Printf("%s: Response for [%s]: %s\n", time.Now().Format(time.RFC3339), prefix, string(message))
}

func main() {
	url := "wss://api.wiv.ew1.tc.development.catapult.com/federation/api/graphql"
	fmt.Printf("Connected to %v \n", url)

	headers := http.Header{}
	headers.Add("Cookie", "access_token=eyJraWQiOiJ1bmlmOVp6cExwSmRPZElPNlArWjlVZFdSWTh6N2JWVFNqNmFNWlVwYTBnPSIsImFsZyI6IlJTMjU2In0.eyJjdXN0b206Y3VzdG9tZXJfaWQiOiI1MTNhMTUxYS02ODViLTQyNzEtODgxNy04ZThhZjNmODI3OWYiLCJzdWIiOiIyMDE5ZTVkYS03NzkyLTQ4YTktYjRhNC0zYzM4ODAzN2Q5NGYiLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAuZXUtd2VzdC0xLmFtYXpvbmF3cy5jb21cL2V1LXdlc3QtMV9mdHVoTVZLU0wiLCJjdXN0b206b2ZfYXRobGV0ZV9pZCI6IiIsImNsaWVudF9pZCI6IjRla2VrdTdlczgwNWM3cWk5dTA3Z2t2aW9pIiwib3JpZ2luX2p0aSI6ImRiMzk0OGQ0LWY1MjgtNDQwZi04M2Y3LTk2NjNhMTI0YzkzNyIsImN1c3RvbTp0b2tlbl9vcmlnaW4iOiJjdXN0b21lci1hcGkiLCJldmVudF9pZCI6IjQ4OTBmNTk3LWE2MWQtNDg2OS05ZDdiLTRiNjRmMTVkZDNlOCIsInRva2VuX3VzZSI6ImFjY2VzcyIsInNjb3BlIjoiYXdzLmNvZ25pdG8uc2lnbmluLnVzZXIuYWRtaW4iLCJhdXRoX3RpbWUiOjE3NjIzMzQ5MDMsImN1c3RvbTpvZl9jdXN0b21lcl9pZCI6IiIsImV4cCI6MTc2MjMzODUwMywiaWF0IjoxNzYyMzM0OTAzLCJqdGkiOiI2ZDQ0NzAyMC1hOTg4LTQ4YmMtYTgxNy0wNmJmNDE0NDllZTIiLCJ1c2VybmFtZSI6IjQwNzgxYjUyLTY5Y2YtNDFlNC1hMGY5LWI0MzBjNGE1N2YyNSJ9.G737P2Ce3fIPRjvADbq8855z3aj9-pWU4eMDBxmJSQydiPTybJ2_BEEw64PdI_txxgx84pEJjNvNa8a5xuW36ON2Z_HksSmpNh5UWg8J-6f3EVvVuSrYKXBustidufBsjrKhHg46SApCelDBKORLTpch3Qny7qO6lGWxxaKLqWapZSmmRvHGw4BPM0esQHZOabUcCpJxcYyIkvWeM2f7WoUrglOWEbJQtmyUDSoSgmCp3Fnxr9INaazz1AZH5DzrHFTYBupC9CiE0ndr0W9r2AiJRcg0qCm7zOsup_xLgK5f1ED7TI2MsX3nhtDnYWxzILWT_HQ6CIGtB4CLB07TPA")
	headers.Add("Sec-WebSocket-Protocol", "graphql-transport-ws")

	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		log.Fatalf("Error connecting to WebSocket: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected to WebSocket")

	initMessage := GraphQLMessage{Type: "connection_init"}
	if err := conn.WriteJSON(initMessage); err != nil {
		log.Fatalf("Error sending connection_init: %v", err)
	}

	for {
		var msg GraphQLMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Fatalf("Error reading WebSocket message: %v", err)
		}
		if msg.Type == "connection_ack" {
			fmt.Println("Received connection_ack, proceeding with subscriptions")
			break
		}
	}

	// Subscribe to createdTags and deletedTags
	send(conn, OperationSubscription, "createdSessions", `subscription { createdSessions { sessions { sessionId }}}`, "")
	createAndDeleteSession(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pingRoutine(ctx, conn)

	// Log all responses
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		fmt.Printf("%s: Received: %s\n", time.Now().Format(time.RFC3339), string(message))
	}
}

type GraphQLWSMessage struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func parseCreateSessionsResponse(payload json.RawMessage) (string, error) {
	if payload == nil || len(payload) == 0 {
		return "", fmt.Errorf("payload is nil or empty")
	}
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return "", fmt.Errorf("error unmarshalling payload: %w", err)
	}
	// Defensive: check for exact structure
	data, ok := payloadMap["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("data not found in payload")
	}
	createSessions, ok := data["createSessions"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("createSessions not found in data")
	}
	sessions, ok := createSessions["sessions"].([]interface{})
	if !ok || len(sessions) == 0 {
		return "", fmt.Errorf("sessions not found or empty in createSessions")
	}
	firstSession, ok := sessions[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("first session is not a map")
	}
	id, ok := firstSession["id"].(string)
	if !ok {
		return "", fmt.Errorf("id not found in first session")
	}
	return id, nil
}

func createAndDeleteSession(conn *websocket.Conn) {
	createQuery := `mutation createSessions($input: [CreateSessionInput!]!) { createSessions(input: $input) { sessions { id name } } }`
	createVars := `{"input": [{"name": "CreateSession"}]}`

	send(conn, OperationMutation, "createSession", createQuery, createVars)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("Error reading createSessions response: %v", err)
		}
		var wsMsg GraphQLWSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Error unmarshalling GraphQLWSMessage: %v", err)
			continue
		}
		id, err := parseCreateSessionsResponse(wsMsg.Payload)
		if err != nil {
			log.Printf("Error parsing createSessions payload: %v", err)
			continue
		}
		fmt.Printf("createSessions returned id: %s\n", id)
		deleteQuery := `mutation($input: DeleteSessionInput!) { deleteSession(input: $input) { success } }`
		deleteVars := fmt.Sprintf(`{"input": {"sessionId": "%s"}}`, id)
		send(conn, OperationMutation, "deleteSession", deleteQuery, deleteVars)
		return
	}
}

func pingRoutine(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingID := uuid.New().String()
			pingMsg := GraphQLMessage{
				Type: "ping",
				Payload: map[string]interface{}{
					"uuid": pingID,
				},
			}
			messageBytes, err := json.Marshal(pingMsg)
			if err != nil {
				log.Println("Failed to marshal ping message:", err)
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
				log.Println("Failed to send ping:", err)
				return
			}
			//fmt.Printf("Ping sent: %s\n", pingID)
		}
	}
}
