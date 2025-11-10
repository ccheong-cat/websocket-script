package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	url := "wss://api.wiv.ew1.tc.development.catapult.com/federation/api/graphql"
	fmt.Printf("Connected to %v \n", url)

	headers := http.Header{}
	headers.Add("Cookie", "access_token=YOUR_ACCESS_TOKEN_HERE")
	headers.Add("Sec-WebSocket-Protocol", "graphql-transport-ws")

	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		log.Fatalf("Error connecting to WebSocket: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected to WebSocket")

	initMsg := GraphQLMessage{Type: "connection_init"}
	if err := conn.WriteJSON(initMsg); err != nil {
		log.Fatalf("Error sending connection_init: %v", err)
	}

	for {
		var msg GraphQLMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Fatalf("Error reading WebSocket message: %v", err)
		}
		if msg.Type == "connection_ack" {
			fmt.Println("Received connection_ack, proceeding with workflow")
			break
		}
	}

	for i := 0; i < 10; i++ {
		createAndDeleteSession(conn)
		time.Sleep(2 * time.Second)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pingRoutine(ctx, conn)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		fmt.Printf("%s: Received: %s\n", time.Now().Format(time.RFC3339), string(message))
	}
}
