package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	_ "embed"

	"github.com/gorilla/websocket"
	"google.golang.org/genai"
)

// To run this sample with a Gemini API key:
//
// $ export GOOGLE_API_KEY=xxxxxxxxxx
// $ go run . -n=8

const sample8Prompt = `
	You are playing the "guessing word" game where the human player with their microphone
	is describing a word. Your job is to listen to the description and say only one word as
	your guess, every few seconds. You have only 3 guesses.
	Don't say anything else than the word you're guessing.
`

const sample9Prompt_fr = `
	Vous jouez au jeu du "mot à deviner" où le joueur humain avec son microphone
	décrit un mot. Votre travail consiste à écouter la description et à ne dire qu'un seul mot comme
	votre suggestion, toutes les quelques secondes. Vous n'avez que 3 essais.
	Ne dites rien d'autre que le mot que vous devinez.
`

func sample8_forbidden_words(ctx context.Context) error {
	log.SetFlags(0)
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/en", serveGame("en"))
	http.HandleFunc("/fr", serveGame("fr"))
	http.HandleFunc("/live/", liveGame)
	http.Handle("/forbiddenwords/", http.StripPrefix("/forbiddenwords/", http.FileServer(http.Dir("testdata/forbiddenwords"))))

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

//go:embed sample8_lang_choice.html
var indexWebapp string

//go:embed sample8_forbiddenwords.html
var gameEnWebapp string

//go:embed sample9_forbiddenwords_fr.html
var gameFrWebapp string

func serveIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index").Parse(indexWebapp)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func serveGame(lang string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var webapp string
		switch lang {
		case "en":
			webapp = gameEnWebapp
		case "fr":
			webapp = gameFrWebapp
		default:
			http.NotFound(w, r)
			return
		}

		tmpl, err := template.New("game").Parse(webapp)
		if err != nil {
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, "ws://"+r.Host+"/live/"+lang)
		if err != nil {
			http.Error(w, "Error executing template", http.StatusInternalServerError)
			return
		}
	}
}

var sample8Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func liveGame(w http.ResponseWriter, r *http.Request) {
	lang := strings.TrimPrefix(r.URL.Path, "/live/")
	var prompt string
	switch lang {
	case "en":
		prompt = sample8Prompt
	case "fr":
		prompt = sample9Prompt_fr
	default:
		log.Printf("unsupported language: %q", lang)
		http.NotFound(w, r)
		return
	}

	c, err := sample8Upgrader.Upgrade(w, r, nil)
	if err != nil {
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
		model = "gemini-live-2.5-flash-preview-native-audio-09-2025"
	} else {
		model = "gemini-2.5-flash-native-audio-preview-09-2025"
	}

	config := &genai.LiveConnectConfig{}
	config.SystemInstruction = &genai.Content{
		Parts: []*genai.Part{
			{Text: prompt},
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
		log.Fatal("connect to model error: ", err)
	}
	defer session.Close()

	go func() {
		for {
			message, err := session.Receive()
			if err != nil {
				log.Println("deconnected: ", err)
				return
			}
			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Fatal("marhal model response error: ", message, err)
			}
			err = c.WriteMessage(websocket.TextMessage, messageBytes)
			if err != nil {
				log.Println("write message error: ", err)
				break
			}
		}
	}()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read from client error: ", err)
			break
		}

		var realtimeInput genai.LiveRealtimeInput
		if err := json.Unmarshal(message, &realtimeInput); err != nil {
			log.Fatal("unmarshal message error ", string(message), err)
		}
		session.SendRealtimeInput(realtimeInput)
	}
}
