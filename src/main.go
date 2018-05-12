package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message, 10)       // broadcast channel
var broadcastMessages []Message

// Configure the upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Define our message object
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

func main() {
	// Create a simple file server
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Start listening for incoming chat messages
	go handleMessages()

	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	// Register our new client
	clients[ws] = true
	log.Printf("new client registered... ")

  // populate message history for new connection
	loadMessages(ws)

  // enters loop until??
	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error reading json: %v", err)
			delete(clients, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		log.Printf("handleConnections again...")
		broadcast <- msg

		// add message to broadcast slice
		saveMessage(msg)
		log.Printf("%v\n", broadcastMessages)

	}
}

func loadMessages(ws *websocket.Conn) {
	if len(broadcastMessages) > 10 {
		for _, messageHistory := range broadcastMessages[len(broadcastMessages) - 10 : len(broadcastMessages)] {
			err := ws.WriteJSON(messageHistory)
			if err != nil {
				log.Printf("error writing json: %v", err)
				ws.Close()
				delete(clients, ws)
			}
		}
	} else {
		for _, messageHistory := range broadcastMessages {
			err := ws.WriteJSON(messageHistory)
			if err != nil {
				log.Printf("error writing json: %v", err)
				ws.Close()
				delete(clients, ws)
			}
		}
	}
}

func saveMessage(message Message) {
	broadcastMessages = append(broadcastMessages, message)
}

func handleMessages() {
	for {
		log.Printf("handleMessages again...")
		// Grab the next message from the broadcast channel
		// broadcast <- Message{"test", "test", "messageTest"}
		msg := <-broadcast
		// msg2 := <-broadcast
		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error writing json: %v", err)
				client.Close()
				delete(clients, client)
			}

		}
	}
}
