package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var allowedColorIdentities []string = []string{"W", "WU", "WR", "UR", "BR", "WUB", "WBG", "UBR", "WUBR"}

func main() {
	FindCommandersByInventory()

	//ListInventoryCardPrices()
}

func ListInventoryCardPrices() {
	cards := ReadCards()
	reg, err := regexp.Compile("[^a-zA-Z0-9\\s\\-]+")
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	cardList := []CardCandidate{}

	for _, card := range cards {
		edhRecJson := GetEDHRecJsonForCard(reg, card, client)
		cardList = append(cardList, MakeCardCandidate(card, *edhRecJson, 0))
	}

	sort.Slice(cardList, func(a, b int) bool {
		return cardList[b].Price < cardList[a].Price
	})

	for _, card := range cardList {
		fmt.Println("1", card.Name)
	}
}

func FindCommandersByInventory() {
	cards := ReadCards()
	cardOccurances := make(map[string][]CardCandidate)
	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9\\s\\-]+")
	if err != nil {
		log.Fatal(err)
	}

	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	for _, card := range cards {
		PopulateCommandersForCard(reg, card, client, cardOccurances)
	}

	sortedCardOccurances := []CardOccurance{}
	for key, value := range cardOccurances {
		// Sort by inclusion rate
		sort.Slice(value, func(a, b int) bool {
			return value[b].InclusionRate < value[a].InclusionRate
		})

		// Sum inclusion rate
		inclusionFactor := float64(0)
		priceFactor := float64(0)
		for _, card := range value {
			inclusionFactor += card.InclusionRate
			priceFactor += card.Price
		}

		sortedCardOccurances = append(sortedCardOccurances, CardOccurance{
			CommanderName:   key,
			CardList:        value,
			TotalOccurances: len(value),
			InclusionFactor: inclusionFactor,
			PriceFactor:     priceFactor,
		})
	}

	sort.Slice(sortedCardOccurances, func(a, b int) bool {
		return sortedCardOccurances[b].InclusionFactor < sortedCardOccurances[a].InclusionFactor
	})

	for _, cardOccurance := range sortedCardOccurances {
		fmt.Println(cardOccurance.CommanderName, ":", len(cardOccurance.CardList), "cards,", cardOccurance.InclusionFactor, "inclusion factor,", cardOccurance.PriceFactor, "price factor,")
		for _, card := range cardOccurance.CardList {
			fmt.Println("- ", card.Name, card.InclusionRate, card.Price)
		}
	}

	// for i := 0; i < 20; i++ {
	// 	fmt.Println(sortedCardOccurances[i].CommanderName, ":", len(sortedCardOccurances[i].CardList), "cards,", sortedCardOccurances[i].InclusionFactor, "inclusion factor,", sortedCardOccurances[i].PriceFactor, "price factor,")
	// 	for _, card := range sortedCardOccurances[i].CardList {
	// 		fmt.Println("- ", card.Name, card.InclusionRate, card.Price)
	// 	}
	// }
}

func ReadCards() []string {
	cards := []string{}
	file, err := os.Open("cardlist.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cards = append(cards, strings.Replace(scanner.Text(), "1 ", "", 1))
	}

	return cards
}

func GetEDHRecJsonURL(reg *regexp.Regexp, cardName string) string {
	return fmt.Sprintf("https://edhrec-json.s3.amazonaws.com/en/cards/%s.json", GetProcessedCardName(reg, cardName))
}

func GetProcessedCardName(reg *regexp.Regexp, cardName string) string {
	return strings.ToLower(strings.ReplaceAll(reg.ReplaceAllString(cardName, ""), " ", "-"))
}

func GetEDHRecJsonForCard(reg *regexp.Regexp, cardName string, client http.Client) *EDHRecJson {
	edhRecJson := EDHRecJson{}

	file, err := os.Open(fmt.Sprintf("cache/%s.json", GetProcessedCardName(reg, cardName)))
	if err != nil {
		file.Close()

		// Download file
		url := GetEDHRecJsonURL(reg, cardName)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Println(err, url)
			return nil
		}

		res, getErr := client.Do(req)
		if getErr != nil {
			log.Println(getErr, url)
			return nil
		}

		if res.Body != nil {
			defer res.Body.Close()
		}

		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			log.Println(readErr, url)
			return nil
		}
		jsonErr := json.Unmarshal(body, &edhRecJson)
		if jsonErr != nil {
			log.Println(jsonErr, url)
			return nil
		}

		log.Println("Successfully read", url)

		// TODO: write out JSON to file
		writeErr := ioutil.WriteFile(fmt.Sprintf("cache/%s.json", GetProcessedCardName(reg, cardName)), body, 0644)
		if writeErr != nil {
			log.Println(writeErr, "Failed to write", fmt.Sprintf("cache/%s.json", GetProcessedCardName(reg, cardName)))
		}
	} else {
		defer file.Close()

		body, readErr := ioutil.ReadAll(file)
		if readErr != nil {
			log.Println(readErr, cardName)
			return nil
		}

		jsonErr := json.Unmarshal(body, &edhRecJson)
		if jsonErr != nil {
			log.Println(jsonErr, cardName)
			return nil
		}
	}

	return &edhRecJson
}

func PopulateCommandersForCard(reg *regexp.Regexp, cardName string, client http.Client, cardOccurances map[string][]CardCandidate) {
	edhRecJson := GetEDHRecJsonForCard(reg, cardName, client)
	if edhRecJson == nil {
		return
	}

	for _, cardlist := range edhRecJson.Container.JsonDict.CardLists {
		if cardlist.Tag == "topcommanders" {
			for _, cardView := range cardlist.CardViews {
				inclusionFactor := float64(cardView.Inclusion) / float64(cardView.PotentialDecks)
				colorIdentity := strings.Join(cardView.ColorIdentity, "")
				if inclusionFactor >= 0.5 && contains(allowedColorIdentities, colorIdentity) {
					cardCandidate := MakeCardCandidate(cardName, *edhRecJson, inclusionFactor)
					if _, ok := cardOccurances[cardView.Name]; ok {
						cardOccurances[cardView.Name] = append(cardOccurances[cardView.Name], cardCandidate)
					} else {
						cardOccurances[cardView.Name] = []CardCandidate{cardCandidate}
					}
				}

			}
		}
	}
}

func MakeCardCandidate(cardName string, edhRecJson EDHRecJson, inclusionFactor float64) CardCandidate {
	return CardCandidate{
		Name:          cardName,
		Price:         GetCardPrice(edhRecJson),
		InclusionRate: inclusionFactor,
	}
}

func GetCardPrice(edhRecJson EDHRecJson) float64 {
	price, ok := edhRecJson.Container.JsonDict.Card.Prices["tcgplayer"].Price.(float64)
	if !ok {
		stringPrice, ok := edhRecJson.Container.JsonDict.Card.Prices["tcgplayer"].Price.(string)
		if !ok {
			log.Fatalln("Cannot convert type, exiting")
		}
		convertedPrice, err := strconv.ParseFloat(stringPrice, 64)
		if err != nil {
			log.Fatalln(err, "Cannot parse float, exiting")
		}
		price = convertedPrice
	}
	return price
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
