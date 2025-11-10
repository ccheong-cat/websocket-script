package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type GraphQLMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type OperationType string

const (
	OperationSubscription OperationType = "subscription"
	OperationMutation     OperationType = "mutation"
)

func send(conn *websocket.Conn, opType OperationType, prefix, query string, variables string) {
	msg := GraphQLMessage{
		ID:      uuid.New().String(),
		Type:    "subscribe",
		Payload: marshalPayload(query, variables),
	}
	if err := conn.WriteJSON(msg); err != nil {
		log.Fatalf("Error sending %s: %v", opType, err)
	}
	fmt.Printf("Sent [%s] %s, query: %s, vars: %s \n", prefix, opType, query, variables)
}

func marshalPayload(query, variables string) json.RawMessage {
	payload := map[string]interface{}{"query": query}
	if variables != "" {
		var vars map[string]interface{}
		if err := json.Unmarshal([]byte(variables), &vars); err == nil {
			payload["variables"] = vars
		}
	}
	b, _ := json.Marshal(payload)
	return b
}

func readResponse(conn *websocket.Conn) ([]byte, error) {
	_, message, err := conn.ReadMessage()
	return message, err
}

func pingRoutine(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingMsg := GraphQLMessage{Type: "ping"}
			if err := conn.WriteJSON(pingMsg); err != nil {
				log.Println("Failed to send ping:", err)
				return
			}
			fmt.Println("Ping sent")
		}
	}
}
