package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
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

const sample8Prompt_fr = `
	Vous jouez au jeu du "mot à deviner" où le joueur humain avec son microphone
	décrit un mot. Votre travail consiste à écouter la description et à ne dire qu'un seul mot comme
	votre suggestion, toutes les quelques secondes. Vous n'avez que 3 essais.
	Ne dites rien d'autre que le mot que vous devinez.
`

func sample8_forbidden_words(ctx context.Context) error {
	log.SetFlags(0)
	http.HandleFunc("/", serveGame)
	http.HandleFunc("/live/", liveGame)
	http.HandleFunc("/sample8_words.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "sample8_words.json")
	})
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

//go:embed sample8_forbiddenwords.html
var gameWebapp string

func serveGame(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("game").Parse(gameWebapp)
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
		prompt = sample8Prompt_fr
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

	gameID := randomString(4)
	forbiddenWords := r.URL.Query()["forbidden"]
	log.Printf("Starting game %s in %s with forbidden words %q", gameID, lang, forbiddenWords)

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

	// Gemini Live session 1 : model listens to the human and guesses the secret word
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
	var shortDuration int32 = 100
	config.RealtimeInputConfig = &genai.RealtimeInputConfig{
		AutomaticActivityDetection: &genai.AutomaticActivityDetection{

			StartOfSpeechSensitivity: "START_SENSITIVITY_HIGH",
			EndOfSpeechSensitivity:   "END_SENSITIVITY_HIGH",
			PrefixPaddingMs:          &shortDuration,
			SilenceDurationMs:        &shortDuration,
		},
	}
	session, err := client.Live.Connect(ctx, model, config)
	if err != nil {
		log.Fatal("connect to model error: ", err)
	}
	defer session.Close()

	// Gemini Live session 2 : model listens to the human and guesses the secret word
	configJudge := &genai.LiveConnectConfig{}
	configJudge.SystemInstruction = &genai.Content{
		Parts: []*genai.Part{
			{Text: `
				You're a judge listening to a human player of Forbidden Words, who is not allowed to
				say any of the words from the forbidden list. If the human player says any of them,
				or a very close word with the same radical, or one of the words translated in aother
				language, then pronounce only the phrase from the human that violated the rule.

				The forbiddens words are: ` + strings.Join(forbiddenWords, ", ")},
		},
	}
	configJudge.ResponseModalities = []genai.Modality{genai.ModalityAudio}
	configJudge.OutputAudioTranscription = &genai.AudioTranscriptionConfig{}
	sessionJudge, err := client.Live.Connect(ctx, model, configJudge)
	if err != nil {
		log.Fatal("connect to model error: ", err)
	}
	defer sessionJudge.Close()

	go func() {
		// Guessing Loop:
		// Receive audio data from the Gemini Live session.
		// Forward it to the player browser, via WebSocket.
		for {
			message, err := session.Receive()
			if err != nil {
				log.Println("guesser model deconnected: ", err)
				return
			}
			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Fatal("marshal guesser model response error: ", message, err)
			}
			err = c.WriteMessage(websocket.TextMessage, messageBytes)
			if err != nil {
				log.Println("write message error: ", err)
				break
			}
		}
	}()

	for {
		// Human speech Loop:
		// Receive audio  and transcript data from player browser, via WebSocket.
		// Forward it to the model guesser player's Gemini Live session.
		// Also forward it to the model judge's Gemini Live session.
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
		sessionJudge.SendRealtimeInput(realtimeInput)
	}

	go func() {
		// Judge Loop:
		// Receive audio and transcript data from the Gemini Live session.
		// Signal to the browser to end the game.
		for {
			message, err := sessionJudge.Receive()
			if err != nil {
				log.Println("judge deconnected: ", err)
				return
			}
			sc := message.ServerContent
			if sc != nil {
				ot := sc.OutputTranscription
				if ot != nil {
					log.Printf("Game %s Judge says %q", gameID, ot.Text)
					// TODO err = c.WriteMessage(websocket.TextMessage, messageBytes)
				}
			}
		}
	}()
}

const alphanum = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func randomString(n int) string {
	a := make([]byte, n)
	for i := range a {
		a[i] = alphanum[rand.Intn(len(alphanum))]
	}
	return string(a)
}
