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
// $ go run . -n=8fr

func sample9_forbiddenWords_fr(ctx context.Context) error {
	log.SetFlags(0)
	http.HandleFunc("/fr", servesample9Webapp_fr)
	http.HandleFunc("/live-fr", sample9Live_fr)
	http.Handle("/forbiddenwords/", http.StripPrefix("/forbiddenwords/", http.FileServer(http.Dir("testdata/forbiddenwords"))))

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	log.Printf("Access the French version at http://localhost:%s/fr", port)
	return http.ListenAndServe(":"+port, nil)
}

const sample9Prompt_fr = `
	Vous jouez au jeu du "mot à deviner" où le joueur humain avec son microphone
	décrit un mot. Votre travail consiste à écouter la description et à ne dire qu'un seul mot comme
	votre suggestion, toutes les quelques secondes. Vous n'avez que 3 essais.
	Ne dites rien d'autre que le mot que vous devinez.
`

//go:embed sample9_forbiddenwords_fr.html
var sample9Webapp_fr string

func servesample9Webapp_fr(w http.ResponseWriter, r *http.Request) {
	// Parse the embedded HTML template.
	tmpl, err := template.New("home").Parse(sample9Webapp_fr)
	if err != nil {
		// Return an internal server error if the template parsing fails.
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	// Execute the template, passing the WebSocket URL to it.
	err = tmpl.Execute(w, "ws://"+r.Host+"/live-fr")
	if err != nil {
		// Return an internal server error if executing the template fails.
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

var sample9Upgrader_fr = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allows connections from any origin
		// This help with testing in Cloud Shell,
		// however we should not do this in a production system.
		return true
	},
}

func sample9Live_fr(w http.ResponseWriter, r *http.Request) {
	// Attempt to upgrade the HTTP connection to a WebSocket connection.
	c, err := sample9Upgrader_fr.Upgrade(w, r, nil)
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

	// NOTE: Model IDs are subject to change. Always consult the official
	// Google Cloud Vertex AI and Google AI Studio Gemini model documentation for the latest versions.
	var model string
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		// Use the latest Vertex AI Live API model with Native Audio Preview (as of Oct 2025)
		model = "gemini-live-2.5-flash-preview-native-audio-09-2025"
	} else {
		// Use the latest Gemini API (Google AI Studio) model with Native Audio Preview (as of Oct 2025)
		// This replaces the soon-to-be-discontinued 'gemini-live-2.5-flash-preview'.
		model = "gemini-2.5-flash-native-audio-preview-09-2025"
	}
	// TODO: Consider updating to the Generally Available (GA) version of the
	// Live API Native Audio models when they are released (expected Nov 2025).

	// Establish the live WebSocket connection with the specified GenAI model.
	config := &genai.LiveConnectConfig{} // empty config
	config.SystemInstruction = &genai.Content{
		Parts: []*genai.Part{
			{Text: sample9Prompt_fr},
		},
	}
	voiceName := "Puck"
	config.SpeechConfig = &genai.SpeechConfig{
		VoiceConfig: &genai.VoiceConfig{
			PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
				VoiceName: voiceName,
			},
		},
	}
	config.ResponseModalities = []genai.Modality{genai.ModalityAudio}
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
		var humanSpeechBuffer []byte
		var modelGuessBuffer []byte
		for {
			// Receive the next message from the GenAI service session.
			message, err := session.Receive()
			if err != nil {
				// Log fatal error if receiving from the GenAI service fails (e.g., connection closed, network error).
				log.Println("deconnected: ", err)
				// The conversation is now stopped, but avoid crashing
				return
			}
			if message.ServerContent != nil {
				if message.ServerContent.InputTranscription != nil {
					inputText := message.ServerContent.InputTranscription.Text
					if inputText != "" {
						log.Printf("Human player says: %s", inputText)
						humanSpeechBuffer = append(humanSpeechBuffer, inputText...)
						log.Printf("Human player says buffer: %s", humanSpeechBuffer)
					}
				}
				if message.ServerContent.OutputTranscription != nil {
					outputText := message.ServerContent.OutputTranscription.Text
					if outputText != "" {
						modelGuessBuffer = append(modelGuessBuffer, outputText...)
						log.Printf("Model player guesses: %s", outputText)
						log.Printf("Model player guesses buffer: %s", modelGuessBuffer)
					}
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
