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
	headers.Add("Cookie", "access_token=eyJraWQiOiJ1bmlmOVp6cExwSmRPZElPNlArWjlVZFdSWTh6N2JWVFNqNmFNWlVwYTBnPSIsImFsZyI6IlJTMjU2In0.eyJjdXN0b206Y3VzdG9tZXJfaWQiOiI1MTNhMTUxYS02ODViLTQyNzEtODgxNy04ZThhZjNmODI3OWYiLCJzdWIiOiIyMDE5ZTVkYS03NzkyLTQ4YTktYjRhNC0zYzM4ODAzN2Q5NGYiLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAuZXUtd2VzdC0xLmFtYXpvbmF3cy5jb21cL2V1LXdlc3QtMV9mdHVoTVZLU0wiLCJjdXN0b206b2ZfYXRobGV0ZV9pZCI6IiIsImNsaWVudF9pZCI6IjRla2VrdTdlczgwNWM3cWk5dTA3Z2t2aW9pIiwib3JpZ2luX2p0aSI6ImE4OTQ2YTJlLWYwZDgtNGMxOC05ODAxLWNlZmYyZTQ2YzIzYyIsImN1c3RvbTp0b2tlbl9vcmlnaW4iOiJjdXN0b21lci1hcGkiLCJldmVudF9pZCI6ImJjNjY4NWFhLTIxYmEtNDJjNS1iZTBkLTAyNmYyZTY3N2ZkYyIsInRva2VuX3VzZSI6ImFjY2VzcyIsInNjb3BlIjoiYXdzLmNvZ25pdG8uc2lnbmluLnVzZXIuYWRtaW4iLCJhdXRoX3RpbWUiOjE3NjIzNDAyMzYsImN1c3RvbTpvZl9jdXN0b21lcl9pZCI6IiIsImV4cCI6MTc2MjM0MzgzNiwiaWF0IjoxNzYyMzQwMjM2LCJqdGkiOiI2NmQwOTM3ZS1hYmVkLTQzMmQtOWUyZi0xNjU1N2VhMjQ4ZmMiLCJ1c2VybmFtZSI6IjQwNzgxYjUyLTY5Y2YtNDFlNC1hMGY5LWI0MzBjNGE1N2YyNSJ9.Zz5dI65LJ7EhipyNDxant9n5kV7WVyzqvSnFGFJuOhyX7qOUfkuWb2qTk9lrm8NIEhqEESniopJeF5Xt71GD0IzAn7wY72SfPAjnKS0F3mfIXDEXcgJtYoilGp1kTvgMuvmm3k7WdhIo54uIm9l7yHoqFSPlfR42VyhVFjA-Ty0FFcAuJyeONsTgNdqhzn36W_LhmLEwsiEgd30QIRZE9yVC2jTzF9NVNXkxb728MkZhRFSi3L5wVWdboU6AD60nqoX0Q5QHJ_kP6z4Sjq3JMzNm2xQl8TkwanhOxpMd9Ggy1iTxcaMPLVXwyM1WipZ4aydCHkL6OmIbbsZxb4ynTA")
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
