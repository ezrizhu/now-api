package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type SteamProfileResp struct {
	Response struct {
		Players []struct {
			PersonaState  int    `json:"personastate"`
			PersonaName   string `json:"personaname"`
			ProfileUrl    string `json:"profileurl"`
			Avatar        string `json:"avatarfull"`
			LastLogoff    int64  `json:"lastlogoff"`
			GameExtraInfo string `json:"gameextrainfo"`
			GameId        string `json:"gameid"`
		} `json:"players"`
	} `json:"response"`
}

type SteamProfile struct {
	PersonaState  string     `json:"personastate"`
	PersonaName   string     `json:"personaname"`
	ProfileUrl    string     `json:"profileurl"`
	Avatar        string     `json:"avatar"`
	LastLogoff    string     `json:"lastlogoff"`
	IsGaming      bool       `json:"isgaming"`
	GameExtraInfo string     `json:"gameextrainfo"`
	GameUrl       string     `json:"gameurl"`
	Mu            sync.Mutex `json:"-"`
}

func (profile *SteamProfile) update() {
	resp, err := http.Get(steamUrl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error fetching steam status")
	}

	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error reading steam status response body")
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status", resp.StatusCode).
			Str("body", string(respBytes)).
			Msg("Error fetching steam status")
	}

	steamStatusResp := SteamProfileResp{}
	err = json.Unmarshal(respBytes, &steamStatusResp)

	if err != nil {
		log.Error().
			Err(err).
			Msg("Error unmarshalling steam status response")
	}

	profile.Mu.Lock()
	defer profile.Mu.Unlock()

	profile.PersonaName = steamStatusResp.Response.Players[0].PersonaName
	profile.ProfileUrl = steamStatusResp.Response.Players[0].ProfileUrl
	profile.Avatar = steamStatusResp.Response.Players[0].Avatar
	profile.LastLogoff = time.Unix(steamStatusResp.Response.Players[0].LastLogoff, 0).Format("2006-01-02 15:04:05")

	if steamStatusResp.Response.Players[0].GameId != "" {
		profile.IsGaming = true
		profile.GameExtraInfo = steamStatusResp.Response.Players[0].GameExtraInfo
		profile.GameUrl = "https://store.steampowered.com/app/" + steamStatusResp.Response.Players[0].GameId
	}

	switch steamStatusResp.Response.Players[0].PersonaState {
	case 0:
		profile.PersonaState = "Offline"
	case 1:
		profile.PersonaState = "Online"
	case 2:
		profile.PersonaState = "Busy"
	case 3:
		profile.PersonaState = "Away"
	case 4:
		profile.PersonaState = "Snooze"
	case 5:
		profile.PersonaState = "Looking to trade"
	case 6:
		profile.PersonaState = "Looking to play"
	default:
		profile.PersonaState = "Unknown"
	}

}
