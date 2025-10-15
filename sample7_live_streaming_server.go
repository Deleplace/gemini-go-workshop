package main

import (
	"context"
	"encoding/json"
	"io"
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
// $ export GOOGLE_GENAI_USE_VERTEXAI=false
// $ go run . -n=7

func sample7_liveStreamingServer(ctx context.Context) error {
	log.SetFlags(0)
	http.HandleFunc("/", homePage)
	http.HandleFunc("/live", live)
	http.HandleFunc("/proxyVideo", proxyVideo)

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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allows connections from any origin
		// This help with testing in Cloud Shell,
		// however we should not do this in a production system.
		return true
	},
}

//go:embed sample7_live_streaming.html
var homeTemplate string

// live handles incoming WebSocket requests for the live streaming example.
// It upgrades the HTTP connection to a WebSocket connection, establishes a
// connection with the configured GenAI model (Gemini API or Vertex AI),
// and then proxies messages between the client WebSocket and the GenAI service.
// It runs two goroutines: one to receive messages from the GenAI service and
// forward them to the client, and another to read messages from the client
// and send them to the GenAI service.
func live(w http.ResponseWriter, r *http.Request) {
	// Attempt to upgrade the HTTP connection to a WebSocket connection.
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Log fatal error if the WebSocket upgrade fails (e.g., invalid request headers).
		log.Fatal("upgrade error: ", err)
		return
	}
	defer c.Close()

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		// Log fatal error if client creation fails (e.g., invalid config, authentication issues).
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
			// {Text: "Always answer in German, only in German."},
			// {Text: "Start the conversation by greeting Marc and Valentin."},
		},
	}
	// voiceName := "Zephyr"
	// voiceName := "Gacrux" // not available for model models/gemini-live-2.5-flash-preview
	// voiceName := "Achird" // not available for model models/gemini-live-2.5-flash-preview
	// voiceName := "Kore" // very very asian
	// config.SpeechConfig = &genai.SpeechConfig{
	// 	VoiceConfig: &genai.VoiceConfig{
	// 		PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
	// 			VoiceName: voiceName,
	// 		},
	// 	},
	// }
	config.ResponseModalities = []genai.Modality{genai.ModalityAudio}
	//config.ResponseModalities = []genai.Modality{genai.ModalityAudio, genai.ModalityText}
	// config.ResponseModalities = []genai.Modality{genai.ModalityText, genai.ModalityAudio}
	// config.ResponseModalities = []genai.Modality{genai.ModalityText}
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
					log.Printf("Input Transcript: %s", message.ServerContent.InputTranscription.Text)
				}
				if message.ServerContent.OutputTranscription != nil && message.ServerContent.OutputTranscription.Text != "" {
					log.Printf("Output Transcript: %s", message.ServerContent.OutputTranscription.Text)
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

func homePage(w http.ResponseWriter, r *http.Request) {
	// Parse the embedded HTML template.
	tmpl, err := template.New("home").Parse(homeTemplate)
	if err != nil {
		// Return an internal server error if the template parsing fails.
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	// fmt.Println("ws://" + r.Host + "/live")
	// Execute the template, passing the WebSocket URL to it.
	err = tmpl.Execute(w, "ws://"+r.Host+"/live")
	if err != nil {
		// Return an internal server error if executing the template fails.
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func proxyVideo(w http.ResponseWriter, r *http.Request) {
	// Fetch the video from Google Cloud Storage.
	resp, err := http.Get("http://storage.googleapis.com/cloud-samples-data/video/animals.mp4")
	if err != nil {
		// Return an internal server error if fetching the video fails.
		http.Error(w, "Error fetching video", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Set CORS header to allow requests from the frontend origin.
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080") // Adjust if your frontend runs elsewhere
	// Set the Content-Type header based on the original video response.
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	// Copy the video content from the GCS response to the client response.
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		// Log error if copying the video content fails.
		log.Printf("Error copying video content: %v", err)
		// It might be too late to send an HTTP error header here if data was already sent.
	}
}
