package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"text/template"

	_ "embed"

	"github.com/gorilla/websocket"
	"google.golang.org/genai"
)

// To run this sample with a Gemini API key:
//
// $ export GOOGLE_API_KEY=xxxxxxxxxx
// $ go run . -n=8

func sample8_forbiddenWords(ctx context.Context) error {
	log.SetFlags(0)
	http.HandleFunc("/", serveSample8Webapp)
	http.HandleFunc("/live", sample8Live)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	return http.ListenAndServe(":"+port, nil)
}

const sample8Prompt = `
	You are playing the "guessing word" game where the human player with their microphone
	is describing a word. Your job is to listen to the description and say only one word as
	your guess, every few seconds. You have only 3 guesses.
	Don't say anything else than the word you're guessing.
`

//go:embed sample8_forbiddenwords.html
var sample8Webapp string

func serveSample8Webapp(w http.ResponseWriter, r *http.Request) {
	// Parse the embedded HTML template.
	tmpl, err := template.New("home").Parse(sample8Webapp)
	if err != nil {
		// Return an internal server error if the template parsing fails.
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	// Execute the template, passing the WebSocket URL to it.
	err = tmpl.Execute(w, "ws://"+r.Host+"/live")
	if err != nil {
		// Return an internal server error if executing the template fails.
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

var sample8Upgrader = websocket.Upgrader{} // use default options

func sample8Live(w http.ResponseWriter, r *http.Request) {
	// Attempt to upgrade the HTTP connection to a WebSocket connection.
	c, err := sample8Upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Log fatal error if the WebSocket upgrade fails (e.g., invalid request headers).
		log.Fatal("upgrade error: ", err)
		return
	}
	defer c.Close()

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal("create client error: ", err)
		return
	}

	var model string
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		model = "gemini-2.0-flash-live-preview-04-09"
	} else {
		model = "gemini-live-2.5-flash-preview"
	}

	// Establish the live WebSocket connection with the specified GenAI model.
	config := &genai.LiveConnectConfig{} // empty config
	config.SystemInstruction = &genai.Content{
		Parts: []*genai.Part{
			{Text: sample8Prompt},
		},
	}
	// config.ResponseModalities = []genai.Modality{genai.ModalityAudio}
	config.InputAudioTranscription = &genai.AudioTranscriptionConfig{}
	config.OutputAudioTranscription = &genai.AudioTranscriptionConfig{}
	session, err := client.Live.Connect(ctx, model, config)
	if err != nil {
		// Log fatal error if connecting to the model fails (e.g., network issues, invalid model name).
		log.Fatal("connect to model error: ", err)
	}
	defer session.Close() // Ensure session is closed when the handler exits

	// Goroutine to receive messages from the GenAI service and send to the client
	go func() {
		for {
			// Receive the next message from the GenAI service session.
			message, err := session.Receive()
			if err != nil {
				// Log fatal error if receiving from the GenAI service fails (e.g., connection closed, network error).
				log.Fatal("receive model response error: ", err)
			}
			if message.ServerContent != nil {
				if message.ServerContent.InputTranscription != nil && message.ServerContent.InputTranscription.Text != "" {
					log.Printf("Human player says: %s", message.ServerContent.InputTranscription.Text)
				}
				if message.ServerContent.OutputTranscription != nil && message.ServerContent.OutputTranscription.Text != "" {
					log.Printf("Model player guesses: %s", message.ServerContent.OutputTranscription.Text)
				}
			}
			// Marshal the received message into JSON format.
			messageBytes, err := json.Marshal(message)
			if err != nil {
				// Log fatal error if marshaling the message to JSON fails.
				log.Fatal("marhal model response error: ", message, err)
			}
			{
				tmpfile, err := os.CreateTemp("", "livestream")
				if err != nil {
					log.Fatalln(err)
				}
				//fmt.Printf("Received JSON from model, writing to %s\n", tmpfile.Name())
				tmpfile.Write(messageBytes)
				tmpfile.Close()
			}
			// Send the JSON message to the client WebSocket.
			err = c.WriteMessage(websocket.TextMessage, messageBytes) // Use TextMessage type for JSON
			if err != nil {
				// Log error and break the loop if writing to the client WebSocket fails (e.g., client disconnected).
				log.Println("write message error: ", err)
				break
			}
		}
	}()

	// Main loop to read messages from the client and send to the GenAI service
	for {
		// Read the next message from the client WebSocket.
		_, message, err := c.ReadMessage()
		if err != nil {
			// Log error and break the loop if reading from the client WebSocket fails (e.g., client disconnected).
			log.Println("read from client error: ", err)
			break // Exit loop on error
		}
		if len(message) > 0 {
			// log.Printf(" bytes size received from client: %d", len(message))
		}

		var realtimeInput genai.LiveRealtimeInput
		// Unmarshal the received client message into a LiveRealtimeInput struct.
		if err := json.Unmarshal(message, &realtimeInput); err != nil {
			// Log fatal error if unmarshaling the client message fails (e.g., invalid JSON format).
			log.Fatal("unmarshal message error ", string(message), err)
		}
		// Send the unmarshaled realtime input to the GenAI service session.
		// Note: This currently doesn't handle potential errors from SendRealtimeInput.
		// Consider adding error handling here if needed.
		session.SendRealtimeInput(realtimeInput)
	}
}
