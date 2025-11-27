package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"
)

type forbiddenWord struct {
	Word      string
	Forbidden []string
}

var englishWords = []forbiddenWord{
	{"Lion", []string{"king", "jungle", "roar", "mane", "cat"}},
	{"Computer", []string{"screen", "keyboard", "mouse", "internet", "code"}},
	{"Guitar", []string{"music", "strings", "rock", "band", "play"}},
}

var frenchWords = []forbiddenWord{
	{"Lion", []string{"roi", "jungle", "rugissement", "crinière", "chat"}},
	{"Ordinateur", []string{"écran", "clavier", "souris", "internet", "code"}},
	{"Guitare", []string{"musique", "cordes", "rock", "groupe", "jouer"}},
}

func sample8_forbidden_words_cli(ctx context.Context) error {
	modelName := "gemini-2.5-flash-lite"

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Choose your language (en/fr): ")
	lang, _ := reader.ReadString('\n')
	lang = strings.TrimSpace(lang)

	var words []forbiddenWord
	var instructions string
	switch lang {
	case "fr":
		words = frenchWords
		instructions = `
			You are the guesser in a game of "Forbidden Words".
			I will describe a word to you. You have to guess what it is.
			You only have 3 guesses.
			Let's start.
			`
	default:
		lang = "en"
		words = englishWords
		instructions = `
			You are the guesser in a game of "Forbidden Words".
			I will describe a word to you. You have to guess what it is.
			You only have 3 guesses.
			The word to guess is for me, not for you to see. I will give you the list of forbidden words.
			You must not use any of the forbidden words in your guess.
			If I use a forbidden word, you must tell me and I lose.
			Let's start.
			`
	}
	//fmt.Println(instructions)

	// Pick a random word
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	gameWord := words[r.Intn(len(words))]

	fmt.Printf("The word to describe is: %s\n", gameWord.Word)
	fmt.Printf("The forbidden words are: %s\n", strings.Join(gameWord.Forbidden, ", "))

	var config *genai.GenerateContentConfig = &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: instructions},
			},
		},
	}

	chat, err := client.Chats.Create(ctx, modelName, config, nil)
	if err != nil {
		return err
	}

	guesses := 3
	for guesses > 0 {
		fmt.Printf("\nDescribe the word.\n> ")
		description, _ := reader.ReadString('\n')
		description = strings.TrimSpace(description)

		// Check for forbidden words (client-side)
		for _, forbidden := range gameWord.Forbidden {
			if strings.Contains(strings.ToLower(description), strings.ToLower(forbidden)) {
				fmt.Printf("Oh! You used the forbidden word '%s'. You lose!\n", forbidden)
				return nil
			}
		}

		result, err := chat.SendMessage(ctx, genai.Part{Text: description})
		if err != nil {
			return fmt.Errorf("error generating content: %w", err)
		}

		// AI's turn
		aiResponse := textOf(result)
		fmt.Printf("AI: %s\n", aiResponse)

		if strings.Contains(strings.ToLower(string(aiResponse)), strings.ToLower(gameWord.Word)) {
			fmt.Println("\nThe AI guessed the word! You win!")
			return nil
		}
		guesses--
	}

	fmt.Printf("\nThe word was %s. You lose!\n", gameWord.Word)
	return nil
}
