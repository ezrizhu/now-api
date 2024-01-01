package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
)

type UpdownResp []struct {
	URL       string `json:"url"`
	Down      bool   `json:"down"`
	DownSince string `json:"down_since"`
}

type Cloud struct {
	IsDown    bool       `json:"is_down"`
	DownSince string     `json:"down_since"`
	Mu        sync.Mutex `json:"-"`
}

func (c *Cloud) check() {
	resp, err := http.Get(updownUrl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error getting updown.io status")
		return
	}

	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error reading updown.io response")
		return
	}

	updownResp := UpdownResp{}
	err = json.Unmarshal(respBytes, &updownResp)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error unmarshalling updown.io response")
		return
	}

	for _, check := range updownResp {
		if check.Down {
			c.Mu.Lock()
			log.Info().
				Str("url", check.URL).
				Bool("down", check.Down).
				Str("down_since", check.DownSince).
				Msg("updown.io status")
			c.IsDown = true
			c.DownSince = check.DownSince
			c.Mu.Unlock()
			return
		}
	}

	c.Mu.Lock()
	c.IsDown = false
	c.Mu.Unlock()
}
