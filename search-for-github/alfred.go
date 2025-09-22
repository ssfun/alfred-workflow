package main

type AlfredItem struct {
	Title      string               `json:"title"`
	Subtitle   string               `json:"subtitle"`
	Arg        string               `json:"arg,omitempty"`
	Valid      bool                 `json:"valid"`
	Match      string               `json:"match,omitempty"`
	Mods       map[string]AlfredMod `json:"mods,omitempty"`
	Variables  map[string]string    `json:"variables,omitempty"`
}

type AlfredMod struct {
	Arg      string `json:"arg,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
}
