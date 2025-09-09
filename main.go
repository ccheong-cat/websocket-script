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

func main() {
	url := "wss://api.wiv.ew1.tc.production.catapult.com/federation/api/graphql"
	fmt.Printf("Connected to %v \n", url)

	headers := http.Header{}
	headers.Add("Cookie", "access_token=eyJraWQiOiJKSGhkUjVYeUtTOHN6NVB1VElSQTdhc2NCRzNSUWUzTTlidEZIT3ZpOWVFPSIsImFsZyI6IlJTMjU2In0.eyJjdXN0b206Y3VzdG9tZXJfaWQiOiI4MTA2MjQwYS0yODZmLTRjZjUtODllZi1lMjEyYWE4YjFlNzAiLCJzdWIiOiI3MmU1MjRhNC1lMDQxLTcwYWQtMWY5ZC0yMWYwM2FiZGFmYTUiLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAuZXUtd2VzdC0xLmFtYXpvbmF3cy5jb21cL2V1LXdlc3QtMV9tSEZsTTBiY3UiLCJjdXN0b206b2ZfYXRobGV0ZV9pZCI6IiIsImNsaWVudF9pZCI6IjVhODc1ZGUxbGlsdG1rbmpranVzOHByMm5uIiwib3JpZ2luX2p0aSI6IjhmNGNlNDZiLTI5N2MtNDlmNi05ZDkwLTA0NjEwOGZhN2UxNCIsImN1c3RvbTp0b2tlbl9vcmlnaW4iOiJjdXN0b21lci1hcGkiLCJldmVudF9pZCI6Ijc3Yjc5NDE4LWJiZTEtNDMwZi1hYTg1LWM3ZDMwYzQ4Y2MxZiIsInRva2VuX3VzZSI6ImFjY2VzcyIsInNjb3BlIjoiYXdzLmNvZ25pdG8uc2lnbmluLnVzZXIuYWRtaW4iLCJhdXRoX3RpbWUiOjE3NTczNDg2MzEsImN1c3RvbTpvZl9jdXN0b21lcl9pZCI6IiIsImV4cCI6MTc1NzM1MjIzMSwiaWF0IjoxNzU3MzQ4NjMxLCJqdGkiOiI2YjZlYjgyYi02YTU2LTQ5OGQtYWY3My1hNmU3NjllYjcxMzYiLCJ1c2VybmFtZSI6ImZiYjE5Mzk3LWI3MjItNGM5ZC1iNGU1LTRjNmVmYTA3ZDNiZCJ9.whe42dxEKu645qXIYSglcMOa35E2U-Kaf4hxNHEj-A70J07nFxTu22Det1cdax5gVxkz5RaBAm8KpIVCs91Yq664vZf8eB7Kx1uQXY6_gEq6CjRCJG3wID5wohIzBoHoQ-ywrh5HjL2b7jEWCiQRxTR2zVuWlOJDNZJjUKKOIwRB_x2QNw8AZLf-2qK1eF93z1KnGgfJ9uF4NZuTvJbi88gJAIEAkimBLd_XEj4hMAW5DgGUatRiDbYK4yNO-SVEZGaJZzbzjXdT8PNAu_ciQFfIvSSuV1WiabM-FYJC9gLWf0YPq35TRWN8oWK1oK6GJ7CF3Sv563HHxs_6o85PrA")
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
	subscribe(conn, "createdPlaylists", `subscription { createdPlaylists { playlists { playlistId }}}`)
	subscribe(conn, "createdAssets", `subscription { createdAssets { assets { assetId }}}`)
	subscribe(conn, "updatedAssets", `subscription { updatedAssets { assets { assetId }}}`)
	subscribe(conn, "createdTags", `subscription { createdTags { tags { tagId }}}`)

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
		log.Fatalf("Error sending %s subscription: %v", idPrefix, err)
	}
	fmt.Printf("%s: Sent %s subscription with id %s\n", time.Now().Format(time.RFC3339), idPrefix, subID.String())
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
			fmt.Printf("Ping sent: %s\n", pingID)
		}
	}
}
