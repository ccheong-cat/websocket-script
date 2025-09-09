package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type GraphQLMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Id      uuid.UUID              `json:"id,omitempty"`
}

func main() {
	const numConnections = 100
	var wg sync.WaitGroup
	wg.Add(numConnections)

	for i := 0; i < numConnections; i++ {
		go func(idx int) {
			defer wg.Done()
			runConnection(idx)
		}(i)
	}

	wg.Wait()
}

func runConnection(idx int) {
	url := "wss://api.wiv.ew1.tc.production.catapult.com/federation/api/graphql"
	headers := http.Header{}
	headers.Add("Cookie", "access_token=YOUR_TOKEN")
	headers.Add("Sec-WebSocket-Protocol", "graphql-transport-ws")

	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		log.Printf("[Conn %d] Error connecting to WebSocket: %v", idx, err)
		return
	}
	defer conn.Close()

	fmt.Printf("[Conn %d] Connected to WebSocket\n", idx)

	initMessage := GraphQLMessage{Type: "connection_init"}
	if err := conn.WriteJSON(initMessage); err != nil {
		log.Printf("[Conn %d] Error sending connection_init: %v", idx, err)
		return
	}

	for {
		var msg GraphQLMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("[Conn %d] Error reading WebSocket message: %v", idx, err)
			return
		}
		if msg.Type == "connection_ack" {
			fmt.Printf("[Conn %d] Received connection_ack, proceeding with subscriptions\n", idx)
			break
		}
	}

	subscribe(conn, "createdPlaylists", `subscription { createdPlaylists { playlists { playlistId }}}`)
	subscribe(conn, "createdAssets", `subscription { createdAssets { assets { assetId }}}`)
	subscribe(conn, "updatedAssets", `subscription { updatedAssets { assets { assetId }}}`)
	subscribe(conn, "createdTags", `subscription { createdTags { tags { tagId }}}`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pingRoutine(ctx, conn, idx)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[Conn %d] Error reading message: %v", idx, err)
			break
		}
		fmt.Printf("[Conn %d] %s: Received: %s\n", idx, time.Now().Format(time.RFC3339), string(message))

		var msgMap map[string]interface{}
		if err := json.Unmarshal(message, &msgMap); err == nil {
			if msgType, ok := msgMap["type"].(string); ok && msgType == "complete" {
				fmt.Printf("[Conn %d] Received complete message, stopping ping and exiting loop.\n", idx)
				cancel()
				break
			}
		}
	}
}

func subscribe(conn *websocket.Conn, idPrefix, query string) {
	subID := uuid.New()
	subMsg := GraphQLMessage{
		Id:   subID,
		Type: "subscribe",
		Payload: map[string]interface{}{
			"query": query,
		},
	}
	if err := conn.WriteJSON(subMsg); err != nil {
		log.Printf("Error sending %s subscription: %v", idPrefix, err)
	}
	fmt.Printf("%s: Sent %s subscription with id %s\n", time.Now().Format(time.RFC3339), idPrefix, subID.String())
}

func pingRoutine(ctx context.Context, conn *websocket.Conn, idx int) {
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
				log.Printf("[Conn %d] Failed to marshal ping message: %v", idx, err)
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
				log.Printf("[Conn %d] Failed to send ping: %v", idx, err)
				return
			}
			fmt.Printf("[Conn %d] Ping sent: %s\n", idx, pingID)
		}
	}
}
