package main

import (
	"context"
	"encoding/json"
	"flag"
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

func logMsg(idx int, format string, args ...interface{}) {
	timestamp := time.Now().Format(time.RFC3339)
	prefix := fmt.Sprintf("[Conn %d] %s: ", idx, timestamp)
	log.Printf(prefix+format, args...)
}

func main() {
	const numConnections = 100
	var wg sync.WaitGroup
	wg.Add(numConnections)

	logPing := flag.Bool("logPing", false, "Log ping/pong messages")
	flag.Parse()

	for i := 0; i < numConnections; i++ {
		go func(idx int) {
			defer wg.Done()
			runConnection(idx, *logPing)
		}(i)
	}

	wg.Wait()
}

func runConnection(idx int, logPing bool) {
	url := "wss://api.wiv.ew1.tc.production.catapult.com/federation/api/graphql"
	headers := http.Header{}
	headers.Add("Cookie", "access_token=eyJraWQiOiJKSGhkUjVYeUtTOHN6NVB1VElSQTdhc2NCRzNSUWUzTTlidEZIT3ZpOWVFPSIsImFsZyI6IlJTMjU2In0.eyJjdXN0b206Y3VzdG9tZXJfaWQiOiI4MTA2MjQwYS0yODZmLTRjZjUtODllZi1lMjEyYWE4YjFlNzAiLCJzdWIiOiI3MmU1MjRhNC1lMDQxLTcwYWQtMWY5ZC0yMWYwM2FiZGFmYTUiLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAuZXUtd2VzdC0xLmFtYXpvbmF3cy5jb21cL2V1LXdlc3QtMV9tSEZsTTBiY3UiLCJjdXN0b206b2ZfYXRobGV0ZV9pZCI6IiIsImNsaWVudF9pZCI6IjVhODc1ZGUxbGlsdG1rbmpranVzOHByMm5uIiwib3JpZ2luX2p0aSI6IjMxNzc2YTI2LWFhMzMtNDM3MS1iMDNkLTE4NTQ4MjYxZWMyZCIsImN1c3RvbTp0b2tlbl9vcmlnaW4iOiJjdXN0b21lci1hcGkiLCJldmVudF9pZCI6IjQ1ZDgwY2QwLWNjZDUtNDg3Ni04ZDY0LTg5NDA0MWRiM2U0OSIsInRva2VuX3VzZSI6ImFjY2VzcyIsInNjb3BlIjoiYXdzLmNvZ25pdG8uc2lnbmluLnVzZXIuYWRtaW4iLCJhdXRoX3RpbWUiOjE3NTc0MjAxODUsImN1c3RvbTpvZl9jdXN0b21lcl9pZCI6IiIsImV4cCI6MTc1NzQyMzc4NSwiaWF0IjoxNzU3NDIwMTg1LCJqdGkiOiIzNmI2NGI5ZS1mMTY2LTQ5MTktOTg5Yy03NDE3MTkwOWIwMjEiLCJ1c2VybmFtZSI6ImZiYjE5Mzk3LWI3MjItNGM5ZC1iNGU1LTRjNmVmYTA3ZDNiZCJ9.FpC8Wui7RWTEiS-4abLO5bEroRPxZnQzYCgwrhVFwHO08xju_zyC3pTU6nOSL7KMC6hBUtTWkRNTRbaHtn2qUi7qeZjF9HJ94fe7hVCKdzgGL6E-N8Ob7uh7Z5iWOXCQnwWCsqG3ADu6lnU0CJMaC0lemADi2v1C6CzRXuQSJywu8n6k70y9tWzkE3dfWMcOLTjoDfLyfU5WXAsAA9_rCjzgSYC-lA0FfS6vWTB6u9A06MBMY3oXYYbedJjFviRrm51U-QTQyluEnhFc0hn18ue6YhC39yHTIZTXB2k6nV5ZoY-X02ouIZmKvpDceU24gimw_fzbzmCq2Z_s-BPBCg")
	headers.Add("Sec-WebSocket-Protocol", "graphql-transport-ws")

	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		logMsg(idx, "Error connecting to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	logMsg(idx, "Connected to WebSocket")

	initMessage := GraphQLMessage{Type: "connection_init"}
	if err := conn.WriteJSON(initMessage); err != nil {
		logMsg(idx, "Error sending connection_init: %v", err)
		return
	}

	for {
		var msg GraphQLMessage
		if err := conn.ReadJSON(&msg); err != nil {
			logMsg(idx, "Error reading WebSocket message: %v", err)
			return
		}
		if msg.Type == "connection_ack" {
			logMsg(idx, "Received connection_ack, proceeding with subscriptions")
			break
		}
	}

	subscribe(conn, "createdPlaylists", `subscription { createdPlaylists { playlists { playlistId }}}`, idx)
	//subscribe(conn, "createdAssets", `subscription { createdAssets { assets { assetId }}}`, idx)
	//subscribe(conn, "updatedAssets", `subscription { updatedAssets { assets { assetId }}}`, idx)
	//subscribe(conn, "createdTags", `subscription { createdTags { tags { tagId }}}`, idx)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pingRoutine(ctx, conn, idx, logPing)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logMsg(idx, "Error reading message: %v", err)
			break
		}

		var msgMap map[string]interface{}
		if err := json.Unmarshal(message, &msgMap); err == nil {
			if msgType, ok := msgMap["type"].(string); ok {
				if msgType == "ping" || msgType == "pong" {
					if logPing {
						logMsg(idx, "Received: %s", string(message))
					}
					continue
				}
				if msgType == "complete" {
					logMsg(idx, "Received: %s", string(message))
					cancel()
					break
				}
			}
		}
		logMsg(idx, "Received: %s", string(message))
	}
}

func subscribe(conn *websocket.Conn, idPrefix, query string, idx int) {
	subID := uuid.New()
	subMsg := GraphQLMessage{
		Id:   subID,
		Type: "subscribe",
		Payload: map[string]interface{}{
			"query": query,
		},
	}
	if err := conn.WriteJSON(subMsg); err != nil {
		logMsg(idx, "Error sending %s subscription: %v", idPrefix, err)
	}
	logMsg(idx, "Sent %s subscription with id %s", idPrefix, subID.String())
}

func pingRoutine(ctx context.Context, conn *websocket.Conn, idx int, logPing bool) {
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
				logMsg(idx, "Failed to marshal ping message: %v", err)
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
				logMsg(idx, "Failed to send ping: %v", err)
				return
			}
			if logPing {
				logMsg(idx, "Ping sent: %s", pingID)
			}
		}
	}
}
