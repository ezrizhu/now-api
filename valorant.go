package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
)

type ValMmr struct {
	Status int `json:"status"`
	Data   struct {
		Name        string `json:"name"`
		Tag         string `json:"tag"`
		CurrentData struct {
			Currenttier        int    `json:"currenttier"`
			Currenttierpatched string `json:"currenttierpatched"`
			Images             struct {
				Small        string `json:"small"`
				Large        string `json:"large"`
				TriangleDown string `json:"triangle_down"`
				TriangleUp   string `json:"triangle_up"`
			} `json:"images"`
			RankingInTier        int  `json:"ranking_in_tier"`
			MmrChangeToLastGame  int  `json:"mmr_change_to_last_game"`
			Elo                  int  `json:"elo"`
			GamesNeededForRating int  `json:"games_needed_for_rating"`
			Old                  bool `json:"old"`
		} `json:"current_data"`
		HighestRank struct {
			Old         bool   `json:"old"`
			Tier        int    `json:"tier"`
			PatchedTier string `json:"patched_tier"`
			Season      string `json:"season"`
			Converted   int    `json:"converted"`
		} `json:"highest_rank"`
	} `json:"data"`
}

type ValProfile struct {
	Name        string     `json:"name"`
	Tag         string     `json:"tag"`
	CurrentRank string     `json:"current_rank"`
	HighestRank string     `json:"highest_rank"`
	Elo         int        `json:"elo"`
	Mu          sync.Mutex `json:"-"`
}

func (profile *ValProfile) update() {
	client := http.Client{}
	mmrReq, err := http.NewRequest("GET", mmrUrl, nil)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error creating req for val mmr")
		return
	}
	mmrReq.Header.Set("Authorization", valKey)

	mmrResp, err := client.Do(mmrReq)
	if err != nil {
		log.Error().Err(err).Msg("Error sending mmr req")
		return
	}

	defer mmrResp.Body.Close()

	mmrRespBytes, err := ioutil.ReadAll(mmrResp.Body)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error reading steam status response body")
		return
	}

	if mmrResp.StatusCode != http.StatusOK {
		log.Error().
			Int("status", mmrResp.StatusCode).
			Str("body", string(mmrRespBytes)).
			Msg("Error fetching steam status")
		return
	}

	valMmrResp := ValMmr{}
	err = json.Unmarshal(mmrRespBytes, &valMmrResp)

	if err != nil {
		log.Error().
			Err(err).
			Msg("Error unmarshalling val mmr resp response")
		return
	}

	profile.Mu.Lock()
	defer profile.Mu.Unlock()

	profile.Name = valMmrResp.Data.Name
	profile.Tag = valMmrResp.Data.Tag
	profile.CurrentRank = valMmrResp.Data.CurrentData.Currenttierpatched
	profile.Elo = valMmrResp.Data.CurrentData.Elo
	profile.HighestRank = valMmrResp.Data.HighestRank.PatchedTier
}
