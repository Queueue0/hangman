package main

import (
	_ "embed"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

//go:embed data/words.txt
var wordsFile string

func main() {
	var forceSingle, executioner, multiplayer bool
	var maxLength int
	var delay float64

	flag.BoolVar(&forceSingle, "g", false, "Play singleplayer as the guesser. Takes precedence over -e and -m.")
	flag.BoolVar(&executioner, "e", false, "Play singleplayer as the executioner. Takes precedence over -m.")
	flag.BoolVar(&multiplayer, "m", false, "Play multiplayer mode")
	flag.IntVar(&maxLength, "l", math.MaxInt, "The maximum word length. If no other flags or -g, will be the exact length.")
	flag.Float64Var(&delay, "d", 2, "The delay between guesses in seconds. Only applicable when used with -e.")

	flag.Parse()

	secret := flag.Arg(0)
	
	clear()

	if forceSingle {
		playGuesser(maxLength)
	} else if executioner {
		playExecutioner(secret, maxLength, delay)
	} else if multiplayer {
		playMultiplayer(secret, maxLength)
	} else {
		playGuesser(maxLength)
	}
}

func playGuesser(maxLength int) {
	words := getWords()
	var secret string
	for len(secret) != maxLength {
		randIndex := rand.Intn(len(words))
		secret = words[randIndex]
		secret = strings.TrimSpace(secret)

		if maxLength == math.MaxInt {
			maxLength = len(secret)
		}

		// Prune
		if len(secret) != maxLength {
			words[randIndex] = words[len(words)-1]
			words = words[:len(words)-1]
		}
	}

	secret = strings.ToUpper(secret)

	humanGuesserLoop(secret)
}

func playExecutioner(word string, maxLength int, delay float64) {
	secret := inputWord(word, maxLength)
	words := getWords()
	
	var guessedLetters []string
	unguessedLetters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	for drawAndCheck(guessedLetters, secret) {
		blanks := generateBlanks(secret, guessedLetters)
		words = prune(words, guessedLetters, blanks)
		fmt.Printf("Possible words: %v\n", len(words))
		fmt.Println("First few possiblilties (semi-alphabetically):")
		high := 10
		if len(words) < 10 {
			high = len(words)
		}
		for _, word := range words[:high] {
			fmt.Println(word)
		}

		maxCount := 0
		bestLetter := unguessedLetters[0]
		bestIndex := 0
		for i, letter := range unguessedLetters {
			count := 0
			for _, word := range words {
				word = strings.ToUpper(word)
				if strings.Contains(word, letter) {
					count++
				}
			}
			if count > maxCount {
				maxCount = count
				bestLetter = letter
				bestIndex = i
			}
		}

		guessedLetters = append(guessedLetters, bestLetter)
		unguessedLetters[bestIndex] = unguessedLetters[len(unguessedLetters)-1]
		unguessedLetters = unguessedLetters[:len(unguessedLetters)-1]
		time.Sleep(time.Duration(delay * float64(time.Second)))
		clear()
	}
}

func playMultiplayer(word string, maxLength int) {
	secret := inputWord(word, maxLength)	
	humanGuesserLoop(secret)
}

func humanGuesserLoop(secret string) {
	var guessedLetters []string
	for drawAndCheck(guessedLetters, secret) {
		var guess string
		fmt.Scanln(&guess)
		guess = strings.ToUpper(guess)
		if !contains(guessedLetters, guess) && len(guess) == 1 {
			guessedLetters = append(guessedLetters, guess)
		}
		clear()
	}
}

func inputWord(secret string, maxLength int) string {
	secret = strings.ToUpper(secret)
	for len(secret) == 0 || len(secret) > maxLength {
		if len(secret) > maxLength {
			fmt.Printf("Too long! Entered word must be %v letters or fewer.\n", maxLength)
		}
		fmt.Println("Enter the secret word:")
		fmt.Scanln(&secret)
		secret = strings.ToUpper(secret)
		clear()
	}

	return secret
}

func drawAndCheck(guessedLetters []string, secretWord string) bool {
	var wrongLetters []string
	for _, letter := range guessedLetters {
		if !strings.Contains(secretWord, letter) {
			wrongLetters = append(wrongLetters, letter)
		}
	}

	var blanks []string
	for _, letter := range secretWord {
		letter := string(letter)
		if contains(guessedLetters, letter) || !strings.Contains(alphabet, letter) {
			blanks = append(blanks, letter)
		} else {
			blanks = append(blanks, "_")
		}
	}
	strBlanks := strings.Join(blanks, " ")
	strGuessedLetters := strings.Join(guessedLetters, " ")

	numWrong := len(wrongLetters)

	output := ` 
  _____
 |     |
`

	if numWrong >= 1 {
		output += " |     O\n"		
	} else {
		output += " |\n"
	}

	if numWrong >= 4 {
		output += " |    /|\\\n"
	} else if numWrong >= 3 {
		output += " |     |\\\n"
	} else if numWrong >= 2 {
		output += " |     |\n"
	} else {
		output += " |\n"
	}

	if numWrong >= 6 {
		output += " |    / \\\n"
	} else if numWrong >= 5 {
		output += " |      \\\n"
	} else {
		output += " |\n"
	}
	output += " -----\n"

	output += fmt.Sprintf("\n%v\n", strBlanks)

	output += fmt.Sprintf("\nGuessed Letters: %v\n", strGuessedLetters)

	if numWrong >= 6 {
		output += fmt.Sprintf("Game Over\nThe word was %v", secretWord)
	}

	if !strings.Contains(strBlanks, "_") {
		output += "WINNER\n"
	}

	fmt.Println(output)

	return numWrong < 6 && strings.Contains(strBlanks, "_")
}

func getWords() []string {
	words := strings.Split(wordsFile, "\n")
	return words
}

func prune(words []string, guessedLetters []string, blanks string) []string {
	var pruned []string
	rBlanks := []rune(blanks)

	outer:
	for _, word := range words {
		word = strings.TrimSpace(word)
		word = strings.ToUpper(word)
		runes := []rune(word)
		if len(runes) != len(rBlanks) {
			continue
		}
		for i, c := range rBlanks {
			if c == '_' {
				if !strings.Contains(alphabet, string(runes[i])) {
					continue outer
				}
				continue
			}

			if c != runes[i] {
				continue outer
			}

			if strings.Count(blanks, string(c)) != strings.Count(word, string(c)) {
				continue outer
			}
		}
		for _, letter := range guessedLetters {
			if !strings.Contains(blanks, letter) && strings.Contains(word, letter) {
				continue outer
			}
		}
		pruned = append(pruned, word)
	}	

	return pruned
}

func generateBlanks(secret string, guessedLetters []string) string {
	var blanks string

	for _, letter := range secret {
		letter := string(letter)
		if contains(guessedLetters, letter) || !strings.Contains(alphabet, letter) {
			blanks += letter
		} else {
			blanks += "_"
		}
	}

	return blanks
}

func contains(slice []string, item string) bool {
	for _, el := range slice {
		if el == item {
			return true
		}
	} 
	return false
}

func clear() {
	clear := exec.Command("clear")
	clear.Stdout = os.Stdout
	clear.Run()
}