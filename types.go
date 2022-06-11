package main

type EDHRecJson struct {
	Container Container
	Panels    Panels
}

type Container struct {
	JsonDict JsonDict `json:"json_dict"`
}

type Panels struct {
	TribeLinks TribeLinks `json:"tribelinks"`
}

type TribeLinks struct {
	Budget []Variant
	Themes []Variant
}

type Variant struct {
	Count      int
	HrefSuffix string `json:"href-suffix"`
	Value      string
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
	PotentialDecks int `json:"potential_decks"`
	Sanitized      string
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
