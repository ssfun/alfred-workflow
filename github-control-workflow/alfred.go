package main

type AlfredItem struct {
	Title    string               `json:"title"`
	Subtitle string               `json:"subtitle"`
	Arg      string               `json:"arg,omitempty"`
	Valid    bool                 `json:"valid"`
	Match    string               `json:"match,omitempty"`
	Mods     map[string]AlfredMod `json:"mods,omitempty"`
}

type AlfredMod struct {
	Arg      string `json:"arg,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
}
