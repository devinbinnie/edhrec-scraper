package main

type EDHRecJson struct {
	Container Container
}

type Container struct {
	JsonDict JsonDict `json:"json_dict"`
}

type JsonDict struct {
	CardLists []CardList `json:"cardlists"`
	Card      Card
}

type CardList struct {
	Header    string
	CardViews []CardView `json:"cardviews"`
	Tag       string
}

type CardView struct {
	Name           string
	Inclusion      int
	PotentialDecks int      `json:"potential_decks"`
	ColorIdentity  []string `json:"color_identity"`
}

type Card struct {
	Prices map[string]CardPrice
}

type CardPrice struct {
	Url   string
	Price interface{}
}

type CardOccurance struct {
	CommanderName   string
	CardList        []CardCandidate
	TotalOccurances int
	InclusionFactor float64
	PriceFactor     float64
}

type CardCandidate struct {
	Name          string
	Price         float64
	InclusionRate float64
}
