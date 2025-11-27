package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"
)

type forbiddenWord struct {
	Word      string   `json:"word"`
	Forbidden []string `json:"forbidden"`
}

type wordsByLang struct {
	En []forbiddenWord `json:"en"`
	Fr []forbiddenWord `json:"fr"`
}

type uiPhrases struct {
	chooseLanguage    string
	wordToDescribe    string
	forbiddenWordsAre string
	describeTheWord   string
	usedForbiddenWord string
	aiGuess           string
	aiGuessedTheWord  string
	wordWas           string
}

var phrases = map[string]uiPhrases{
	"en": {
		chooseLanguage:    "Choose your language (en/fr): ",
		wordToDescribe:    "The word to describe is: %s\n",
		forbiddenWordsAre: "The forbidden words are: %s\n",
		describeTheWord:   "\nDescribe the word.\n> ",
		usedForbiddenWord: "Oh! You used the forbidden word '%s'. You lose!\n",
		aiGuess:           "AI: %s\n",
		aiGuessedTheWord:  "\nThe AI guessed the word! You win!\n",
		wordWas:           "\nThe word was %s. You lose!\n",
	},
	"fr": {
		chooseLanguage:    "Choisissez votre langue (en/fr): ",
		wordToDescribe:    "Le mot à décrire est : %s\n",
		forbiddenWordsAre: "Les mots interdits sont : %s\n",
		describeTheWord:   "\nDécrivez le mot.\n> ",
		usedForbiddenWord: "Oh ! Vous avez utilisé le mot interdit '%s'. Vous avez perdu !\n",
		aiGuess:           "IA : %s\n",
		aiGuessedTheWord:  "\nL'IA a deviné le mot ! Vous avez gagné !\n",
		wordWas:           "\nLe mot était %s. Vous avez perdu !\n",
	},
}

func sample8_forbidden_words_cli(ctx context.Context) error {
	modelName := "gemini-2.5-flash-lite"

	// Load words from JSON file
	file, err := os.ReadFile("sample8_words.json")
	if err != nil {
		return fmt.Errorf("failed to read words file: %w", err)
	}

	var allWords wordsByLang
	if err := json.Unmarshal(file, &allWords); err != nil {
		return fmt.Errorf("failed to parse words file: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print(phrases["en"].chooseLanguage)
	lang, _ := reader.ReadString('\n')
	lang = strings.TrimSpace(lang)

	var words []forbiddenWord
	var instructions string
	var currentPhrases uiPhrases
	switch lang {
	case "fr":
		words = allWords.Fr
		currentPhrases = phrases["fr"]
		instructions = `
			Tu es le devineur dans une partie de "Mots Interdits".
			Je vais te décrire un mot. Tu dois deviner ce que c'est.
			Tu n'as que 3 essais.
			Je connais le mot à faire deviner, mais je ne peux pas te le dire.
			Je ne peux pas non plus te dire plusieurs mots interdits.
			Réponds uniquement en Français.
			Réponds uniquement le mot que tu supposes être celui que j'essaie de faire deviner.
			Commençons.
			`
	default:
		lang = "en"
		words = allWords.En
		currentPhrases = phrases["en"]
		instructions = `
			You are the guesser in a game of "Forbidden Words".
			I will describe a word to you. You have to guess what it is.
			You only have 3 guesses.
			I know the word to guess, but I cannot say it to you.
			I also cannot say several other forbidden words.
			Answer only in English.
			Answer only with the word you think is the one I'm trying to let you guess.
			Let's start.
			`
	}
	//fmt.Println(instructions)

	// Pick a random word
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	gameWord := words[r.Intn(len(words))]

	fmt.Println()
	fmt.Printf(currentPhrases.wordToDescribe, gameWord.Word)
	fmt.Printf(currentPhrases.forbiddenWordsAre, strings.Join(gameWord.Forbidden, ", "))

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
		fmt.Printf(currentPhrases.describeTheWord)
		description, _ := reader.ReadString('\n')
		description = strings.TrimSpace(description)

		// Check for forbidden words (client-side)
		for _, forbidden := range gameWord.Forbidden {
			if strings.Contains(strings.ToLower(description), strings.ToLower(forbidden)) {
				fmt.Printf(currentPhrases.usedForbiddenWord, forbidden)
				return nil
			}
		}

		result, err := chat.SendMessage(ctx, genai.Part{Text: description})
		if err != nil {
			return fmt.Errorf("error generating content: %w", err)
		}

		// AI's turn
		aiResponse := textOf(result)
		fmt.Printf(currentPhrases.aiGuess, aiResponse)

		if strings.Contains(strings.ToLower(string(aiResponse)), strings.ToLower(gameWord.Word)) {
			fmt.Println(currentPhrases.aiGuessedTheWord)
			return nil
		}
		guesses--
	}

	fmt.Printf(currentPhrases.wordWas, gameWord.Word)
	return nil
}
