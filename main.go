package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var allowedColorIdentities []string = []string{
	"W",
	// "U",
	// "B",
	// "R",
	// "G",

	// "WU",
	// "UB",
	// "BR",
	// "RW",
	// "RG",
	// "GW",
	// "WB",
	// "UR",
	// "BG",
	// "GU",

	// "WUB",
	"UBR",
	// "BRG",
	// "RGW",
	// "GWU",
	"WBG",
	// "URW",
	// "BGU",
	// "RWB",
	// "GUR",

	"WUBR",
	"UBRG",
	"BRGW",
	"RGWU",
	"GWUB",

	// "WUBRG",
}

func main() {
	FindTopCardsByInventory()
	//FindCommandersByInventory()
	//ListInventoryCardPrices()
}

func ListInventoryCardPrices() {
	cards := ReadCardsAndRemoveDupes()
	reg, err := regexp.Compile("[^a-zA-Z0-9\\s\\-]+")
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	cardList := []CardCandidate{}

	for _, card := range cards {
		edhRecJson := GetEDHRecJsonForPath(GetPathForCard(reg, card), client)
		cardList = append(cardList, MakeCardCandidate(card, GetCardPrice(*edhRecJson), 0))
	}

	sort.Slice(cardList, func(a, b int) bool {
		return cardList[b].Price < cardList[a].Price
	})

	for _, card := range cardList {
		fmt.Println("1", card.Name)
	}
}

func SortAndPrintCardOccurances(cardOccurances map[string][]CardCandidate) {
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
		return len(sortedCardOccurances[b].CardList) < len(sortedCardOccurances[a].CardList)
		//return sortedCardOccurances[b].InclusionFactor < sortedCardOccurances[a].InclusionFactor
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

func FindTopCardsByInventory() {
	cards := ReadCardsAndRemoveDupes()
	cardOccurances := make(map[string][]CardCandidate)
	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9\\s\\-]+")
	if err != nil {
		log.Fatal(err)
	}

	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	for _, color := range allowedColorIdentities {
		PopulateCardsForCommandersByColor(color, cards, reg, client, cardOccurances)
	}

	SortAndPrintCardOccurances(cardOccurances)
}

func FindCommandersByInventory() {
	cards := ReadCardsAndRemoveDupes()
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

	SortAndPrintCardOccurances(cardOccurances)
}

func ReadCardsAndRemoveDupes() []string {
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

	// Remove duplicates
	keys := make(map[string]bool)
	list := []string{}

	// If the key(values of the slice) is not equal
	// to the already present value in new slice (list)
	// then we append it. else we jump on another element.
	for _, entry := range cards {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	cards = list

	return cards
}

func GetPathForCard(reg *regexp.Regexp, cardName string) string {
	return fmt.Sprintf("cards/%s.json", GetProcessedCardName(reg, cardName))
}

func GetPathForCommander(reg *regexp.Regexp, cardName string) string {
	return fmt.Sprintf("commanders/%s.json", GetProcessedCardName(reg, cardName))
}

func GetPathForCommanderVariant(reg *regexp.Regexp, cardName string, subpath string) string {
	return fmt.Sprintf("commanders/%s%s.json", GetProcessedCardName(reg, cardName), subpath)
}

func GetPathForCommandersByColor(colors string) string {
	return fmt.Sprintf("commanders/%s.json", strings.ToLower(colors))
}

func GetEDHRecJsonURL(path string) string {
	return fmt.Sprintf("https://json.edhrec.com/v2/%s", path)
}

func GetProcessedCardName(reg *regexp.Regexp, cardName string) string {
	return strings.ToLower(strings.ReplaceAll(reg.ReplaceAllString(cardName, ""), " ", "-"))
}

func GetEDHRecJsonForPath(path string, client http.Client) *EDHRecJson {
	edhRecJson := EDHRecJson{}
	outputPath := fmt.Sprintf("cache/%s", path)

	file, err := os.Open(outputPath)
	if err != nil {
		file.Close()

		// Download file
		url := GetEDHRecJsonURL(path)

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

		dirErr := os.MkdirAll(filepath.Dir(outputPath), 0644)
		if dirErr != nil {
			log.Println(dirErr, url)
			return nil
		}
		writeErr := ioutil.WriteFile(outputPath, body, 0644)
		if writeErr != nil {
			log.Println(writeErr, "Failed to write", outputPath)
		}
	} else {
		defer file.Close()

		body, readErr := ioutil.ReadAll(file)
		if readErr != nil {
			log.Println(readErr, path)
			return nil
		}

		jsonErr := json.Unmarshal(body, &edhRecJson)
		if jsonErr != nil {
			log.Println(jsonErr, path)
			return nil
		}
	}

	return &edhRecJson
}

func PopulateCardsForCommandersByColor(color string, cards []string, reg *regexp.Regexp, client http.Client, cardOccurances map[string][]CardCandidate) {
	colorJson := GetEDHRecJsonForPath(GetPathForCommandersByColor(color), client)
	if colorJson == nil {
		return
	}

	for _, commander := range colorJson.Container.JsonDict.CardLists[0].CardViews {
		if !strings.Contains(commander.Name, "Tymna") { // I don't want to play Tymna
			commanderJson := GetEDHRecJsonForPath(GetPathForCommander(reg, commander.Sanitized), client)
			PopulateCardsForCommander(cards, reg, client, commander.Name, *commanderJson, cardOccurances)
			for _, variant := range commanderJson.Panels.TribeLinks.Budget {
				commanderVariantJson := GetEDHRecJsonForPath(GetPathForCommanderVariant(reg, commander.Sanitized, variant.HrefSuffix), client)
				PopulateCardsForCommander(cards, reg, client, fmt.Sprintf("%s - %s", commander.Name, variant.Value), *commanderVariantJson, cardOccurances)
			}
			for _, variant := range commanderJson.Panels.TribeLinks.Themes {
				commanderVariantJson := GetEDHRecJsonForPath(GetPathForCommanderVariant(reg, commander.Sanitized, variant.HrefSuffix), client)
				PopulateCardsForCommander(cards, reg, client, fmt.Sprintf("%s - %s", commander.Name, variant.Value), *commanderVariantJson, cardOccurances)
			}
		}
	}
}

func PopulateCardsForCommander(cards []string, reg *regexp.Regexp, client http.Client, commanderName string, commanderJson EDHRecJson, cardOccurances map[string][]CardCandidate) {
	for _, cardList := range commanderJson.Container.JsonDict.CardLists {
		if cardList.Tag == "highsynergycards" || cardList.Tag == "topcards" {
			for _, card := range cardList.CardViews {
				if contains(cards, card.Name) {
					inclusionFactor := float64(card.Inclusion) / float64(card.PotentialDecks)
					cardCandidate := MakeCardCandidate(card.Name, 0.0, inclusionFactor)
					if _, ok := cardOccurances[commanderName]; ok {
						cardOccurances[commanderName] = append(cardOccurances[commanderName], cardCandidate)
					} else {
						cardOccurances[commanderName] = []CardCandidate{cardCandidate}
					}
				}
			}
		}
	}
}

func PopulateCommandersForCard(reg *regexp.Regexp, cardName string, client http.Client, cardOccurances map[string][]CardCandidate) {
	edhRecJson := GetEDHRecJsonForPath(GetPathForCard(reg, cardName), client)
	if edhRecJson == nil {
		return
	}

	for _, cardlist := range edhRecJson.Container.JsonDict.CardLists {
		if cardlist.Tag == "topcommanders" {
			for _, cardView := range cardlist.CardViews {
				inclusionFactor := float64(cardView.Inclusion) / float64(cardView.PotentialDecks)
				colorIdentity := strings.Join(cardView.ColorIdentity, "")
				if contains(allowedColorIdentities, colorIdentity) {
					cardCandidate := MakeCardCandidate(cardName, GetCardPrice(*edhRecJson), inclusionFactor)
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

func MakeCardCandidate(cardName string, price float64, inclusionFactor float64) CardCandidate {
	return CardCandidate{
		Name:          cardName,
		Price:         price,
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
