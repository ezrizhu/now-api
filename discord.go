package main

import (
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

type Discord struct {
	Mu     sync.Mutex `json:"-"`
	Status Status     `json:"status"`
}

type Status struct {
	StatusDesk   string `json:"status_desk"`
	StatusWeb    string `json:"status_web"`
	StatusMobile string `json:"status_mobile"`
	CustomStatus string `json:"custom_status"`
	StatusEmoji  string `json:"status_emoji"`
	UpdatedAt    string `json:"updated_at"`
}

type Activity struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Details string `json:"details"`
}

var discord Discord

func presenceUpdateHandler(s *discordgo.Session, p *discordgo.PresenceUpdate) {
	if p.User.ID != discordId {
		return
	}

	discord.Mu.Lock()
	defer discord.Mu.Unlock()

	status := &discord.Status

	status.StatusDesk = string(p.ClientStatus.Desktop)
	status.StatusWeb = string(p.ClientStatus.Web)
	status.StatusMobile = string(p.ClientStatus.Mobile)

	for _, activity := range p.Activities {
		if activity.Type == discordgo.ActivityTypeGame {
			status.CustomStatus = "Playing " + activity.Name
			break
		}

		if activity.Type == discordgo.ActivityTypeCustom {
			if activity.Name == "Custom Status" {
				status.CustomStatus = activity.State
				emoji := "https://cdn.discordapp.com/emojis/"
				emoji += activity.Emoji.ID
				if activity.Emoji.Animated {
					emoji += ".gif"
				} else {
					emoji += ".png"
				}
				status.StatusEmoji = emoji

				break
			}
		}

		status.CustomStatus = ""
		status.StatusEmoji = ""
	}
	status.UpdatedAt = time.Now().Format("15:04:05 MST")
	log.Debug().Msg("Discord status updated")
}
