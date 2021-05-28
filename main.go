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
	"strings"
	"time"
)

func main() {
	cards := ReadCards()
	cardOccurances := make(map[string]int)
	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9\\s\\-]+")
	if err != nil {
		log.Fatal(err)
	}

	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	for _, card := range cards {
		url := GetEDHRecJsonURL(reg, card)
		GetCards(url, client, cardOccurances)
	}

	sortedCardOccurances := []CardOccurance{}
	for key, value := range cardOccurances {
		sortedCardOccurances = append(sortedCardOccurances, CardOccurance{
			Name:       key,
			Occurances: value,
		})
	}

	sort.Slice(sortedCardOccurances, func(a, b int) bool {
		return sortedCardOccurances[b].Occurances < sortedCardOccurances[a].Occurances
	})

	for i := 0; i < 10; i++ {
		fmt.Println(sortedCardOccurances[i].Name, sortedCardOccurances[i].Occurances)
	}
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
	processedCardName := strings.ToLower(strings.ReplaceAll(reg.ReplaceAllString(cardName, ""), " ", "-"))
	return fmt.Sprintf("https://edhrec-json.s3.amazonaws.com/en/cards/%s.json", processedCardName)
}

func GetCards(url string, client http.Client, cardOccurances map[string]int) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(err, url)
		return
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Println(getErr, url)
		return
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Println(readErr, url)
		return
	}

	edhRecJson := EDHRecJson{}
	jsonErr := json.Unmarshal(body, &edhRecJson)
	if jsonErr != nil {
		log.Println(jsonErr, url)
		return
	}

	fmt.Println("Successfully read", url)

	for _, cardlist := range edhRecJson.Container.JsonDict.CardLists {
		if cardlist.Tag == "topcommanders" {
			for _, cardview := range cardlist.CardViews {
				if val, ok := cardOccurances[cardview.Name]; ok {
					cardOccurances[cardview.Name] = val + 1
				} else {
					cardOccurances[cardview.Name] = 1
				}
			}
		}
	}
}
