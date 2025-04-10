package client

type Subscription struct {
	Command    string `json:"command"`
	Identifier string `json:"identifier"`
}

type Identifier struct {
	Channel string `json:"channel"`
	Tunnel  string `json:"tunnel"`
}

type Message struct {
	Command    string `json:"command"`
	Identifier string `json:"identifier"`
	Data       string `json:"data"`
}