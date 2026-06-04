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
	"unicode"

	"golang.org/x/sync/errgroup"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
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
	chooseLanguage          string
	wordToDescribe          string
	forbiddenWordsAre       string
	describeTheWord         string
	usedForbiddenWord       string
	usedForbiddenInflection string
	aiGuess                 string
	aiGuessedTheWord        string
	wordWas                 string
}

var phrases = map[string]uiPhrases{
	"en": {
		chooseLanguage:          "Choose your language (en/fr): ",
		wordToDescribe:          "The word to describe is: %s\n",
		forbiddenWordsAre:       "The forbidden words are: %s\n",
		describeTheWord:         "\nDescribe the word.\n> ",
		usedForbiddenWord:       "Oh! You used the forbidden word '%s'. You lose!\n",
		usedForbiddenInflection: "Oh! You sais '%s' which is too close to the forbidden word '%s'. You lose!\n",
		aiGuess:                 "AI: %s\n",
		aiGuessedTheWord:        "\nThe AI guessed the word! You win!\n",
		wordWas:                 "\nThe word was %s. You lose!\n",
	},
	"fr": {
		chooseLanguage:          "Choisissez votre langue (en/fr): ",
		wordToDescribe:          "Le mot à décrire est : %s\n",
		forbiddenWordsAre:       "Les mots interdits sont : %s\n",
		describeTheWord:         "\nDécrivez le mot.\n> ",
		usedForbiddenWord:       "Oh! Vous avez utilisé le mot interdit '%s'. Vous avez perdu !\n",
		usedForbiddenInflection: "Oh! Vous avez dit '%s' qui est trop proche du mot interdit '%s'. Vous avez perdu !\n",
		aiGuess:                 "IA : %s\n",
		aiGuessedTheWord:        "\nL'IA a deviné le mot ! Vous avez gagné !\n",
		wordWas:                 "\nLe mot était %s. Vous avez perdu !\n",
	},
}

const sample8ModelName = "gemini-3.5-flash"

func sample8_forbidden_words_cli(ctx context.Context) error {

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

	var words []forbiddenWord
	var instructions string
	var currentPhrases uiPhrases

	var langChosen = false
	for !langChosen {
		fmt.Print(phrases["en"].chooseLanguage)
		lang, _ := reader.ReadString('\n')
		lang = strings.TrimSpace(lang)

		switch lang {
		case "fr":
			langChosen = true
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
		case "en":
			langChosen = true
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

	chat, err := client.Chats.Create(ctx, sample8ModelName, config, nil)
	if err != nil {
		return err
	}

	guesses := 3
	for guesses > 0 {
		fmt.Printf(currentPhrases.describeTheWord)
		description, _ := reader.ReadString('\n')
		description = strings.TrimSpace(description)

		g := new(errgroup.Group)

		// Check for forbidden words
		var lost bool
		var forbiddenSaid, forbiddenMatched string
		g.Go(func() error {
			lost, forbiddenSaid, forbiddenMatched, err = gameWord.saidForbidden(ctx, description)
			return err
		})

		// Let Gemini guess, concurrently
		var result *genai.GenerateContentResponse
		g.Go(func() error {
			result, err = chat.SendMessage(ctx, genai.Part{Text: description})
			return err
		})

		err := g.Wait()
		if err != nil {
			return err
		}

		if lost {
			if normalize(forbiddenSaid) == normalize(forbiddenMatched) {
				// Exact match
				fmt.Printf(currentPhrases.usedForbiddenWord, forbiddenMatched)
			} else {
				// Fuzzy match
				fmt.Printf(currentPhrases.usedForbiddenInflection, forbiddenSaid, forbiddenMatched)
			}
			return nil
		}

		// AI's guess
		aiResponse := textOf(result)
		fmt.Printf(currentPhrases.aiGuess, aiResponse)

		winning, err := gameWord.isWinning(ctx, aiResponse)
		if err != nil {
			return err
		}

		if winning {
			fmt.Println(currentPhrases.aiGuessedTheWord)
			return nil
		}
		guesses--
	}

	fmt.Printf(currentPhrases.wordWas, gameWord.Word)
	return nil
}

func (fw *forbiddenWord) isWinning(ctx context.Context, guess string) (bool, error) {
	lowGuess := normalize(guess)
	lowGoal := normalize(fw.Word)
	return strings.Contains(lowGuess, lowGoal), nil
}

// normalize returns its argument lowercased and without diacritics
func normalize(s string) string {
	// Local transformers, not shared with other goroutines
	tr := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, err := transform.String(tr, strings.ToLower(s))
	if err != nil {
		// We do not expect string transformation to fail in general
		panic(err)
	}
	return normalized
}

func (fw *forbiddenWord) saidForbidden(ctx context.Context, said string) (lost bool, forbiddenSaid string, forbiddenMatched string, err error) {
	systemInstruction := `
		You are the judge in the Forbidden Words game.
		The human player will say a description.

		If the prompt contains any of the forbidden words, or an inflection of a forbidden
		word, or a forbidden word translated in another language, then the game is lost.

		In the field "forbiddenWord", provide exactly one of the original forbidden words.

		In the field "fragment", provide the part of the prompt that violated the rule.

		The description must be rejected as using a forbidden word only if it actually contains
		an inflection, or misspelling, or translation of a forbidden word. Synonyms of forbidden
		words must not trigger a lost game.

		E.g. "ficelle" does not match the forbidden word "Corde", because the two words have
		a similar meaning but the word "ficelle" is not an inflection of the word "corde" and
		the game is not lost.

		E.g. "orange" does not match the forbidden word "Agrume", because the two words have
		a similar meaning but the word "orange" is not an inflection of the word "Agrume" and
		the game is not lost.

		E.g. "tronc" does not match the forbidden word "Arbre", because the two words have
		related meaning but the word "tronc" is not an inflection of the word "Arbre" and
		the game is not lost.

		E.g. "poussent" matches the fodbidden word "Pousser", because "poussent" is a
		conjugation of the verb "Pousser", thus it is an inflection of "Pousser" and the game
		is lost.

		The forbidden words are:
	` + fw.Word + ", " + strings.Join(fw.Forbidden, ", ")

	// Force JSON structured output
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromParts([]*genai.Part{
			{Text: systemInstruction},
		}, genai.RoleModel),
		ResponseMIMEType: "application/json",
		ResponseJsonSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"lost": {
					Type:        genai.TypeBoolean,
					Description: "Indicates if the user has lost the game.",
				},
				"forbiddenWord": {
					Type:        genai.TypeString,
					Description: "The word that triggered the loss condition.",
				},
				"fragment": {
					Type:        genai.TypeString,
					Description: "The text fragment analyzed.",
				},
			},
			Required: []string{"lost"},
		},
	}

	prompt := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			{Text: said},
		}, genai.RoleUser),
	}

	resp, err := client.Models.GenerateContent(ctx, sample8ModelName, prompt, config)

	if err != nil {
		return false, "", "", err
	}

	structureAnswer := resp.Candidates[0].Content.Parts[0].Text

	// Parse structureAnswer to return the fields
	var result struct {
		Lost          bool   `json:"lost"`
		ForbiddenWord string `json:"forbiddenWord"`
		Fragment      string `json:"fragment"`
	}
	if err := json.Unmarshal([]byte(structureAnswer), &result); err != nil {
		return false, "", "", fmt.Errorf("failed to parse AI response: %w", err)
	}

	return result.Lost, result.Fragment, result.ForbiddenWord, nil
}
