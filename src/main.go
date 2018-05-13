package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message, 10)       // broadcast channel
// TODO: in-memory cache of messages should be replaced with database
var allMessages []Message

// Configure the upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Define our message object
type Message struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

func main() {
	// Create a simple file server
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Convenience method for clearing in-memory message cache if things get out of control
	http.HandleFunc("/clear-messages", clearMessages)

	// Convenience method to view all messages for testing
	http.HandleFunc("/view-all", viewAllMessages)

	// Start go routine listening for incoming chat messages
	go handleMessages()

	// Start the server on localhost port 8000 and log any errors
	log.Println("HTTP server started on :8000")
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
	log.Printf("New client registered")

	// populate message history for new connection
	loadMessages(ws)

	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading json: %v", err)
			delete(clients, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- msg

		// add message to broadcast slice
		saveMessage(msg)
		log.Printf("%v\n", allMessages)

	}
}

// TODO: replace with MySQL
func loadMessages(ws *websocket.Conn) {
	if len(allMessages) > 10 {
		log.Printf(string(len(allMessages)))
		for _, messageHistory := range allMessages[len(allMessages)-10 : len(allMessages)] {
			sendMessage(messageHistory, ws)
		}
	} else {
		for _, messageHistory := range allMessages {
			sendMessage(messageHistory, ws)
		}
	}
}

func sendMessage(messageHistory Message, ws *websocket.Conn) {
	err := ws.WriteJSON(messageHistory)
	if err != nil {
		log.Printf("Error writing json: %v", err)
		ws.Close()
		delete(clients, ws)
	}
}

// TODO: replace with MySQL
func saveMessage(message Message) {
	allMessages = append(allMessages, message)
}

func clearMessages(w http.ResponseWriter, r *http.Request) {
	log.Printf("Clearing messages")
	allMessages = allMessages[len(allMessages)-10 : len(allMessages)]
}

func viewAllMessages(w http.ResponseWriter, r *http.Request) {
	log.Printf("Viewing all messages")
	json.NewEncoder(w).Encode(allMessages)
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast

		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("Error writing json: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
