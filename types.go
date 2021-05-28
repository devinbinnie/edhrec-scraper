package main

type EDHRecJson struct {
	Container Container
}

type Container struct {
	JsonDict JsonDict `json:"json_dict"`
}

type JsonDict struct {
	CardLists []CardList `json:"cardlists"`
}

type CardList struct {
	Header    string
	CardViews []CardView `json:"cardviews"`
	Tag       string
}

type CardView struct {
	Name string
}

type CardOccurance struct {
	Name       string
	Occurances int
}
