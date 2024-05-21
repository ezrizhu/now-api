package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Workstation struct {
	Status     string     `json:"status"`
	LastUpdate time.Time  `json:"lastUpdate"`
	Mu         sync.Mutex `json:"-"`
}

var (
	k = koanf.New(".")

	discordToken string
	discordId    string
	steamUrl     string
	updownUrl    string
	ideKey       string
	matchesUrl   string
	mmrUrl       string
	valKey       string
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	debug := flag.Bool("debug", false, "sets log level to debug")

	flag.Parse()

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Debug().Msg("Debug mode enabled")

	// load config
	if err := k.Load(file.Provider("config.toml"), toml.Parser()); err != nil {
		log.Error().Err(err).Msg("Could not load config")
	}

	discordToken = k.String("discord.token")
	discordId = k.String("discord.id")
	steamUrl = k.String("steam.url")
	updownUrl = k.String("updown.url")
	ideKey = k.String("ide.key")
	matchesUrl = k.String("valorant.matchesUrl")
	mmrUrl = k.String("valorant.mmrUrl")
	valKey = k.String("valorant.key")

	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Error().Err(err).Msg("Could not create dg session")
	}
	defer dg.Close()

	dg.Identify.Intents |= discordgo.IntentGuildPresences
	dg.Identify.Intents |= discordgo.IntentGuildMembers

	dg.AddHandler(presenceUpdateHandler)

	err = dg.Open()
	if err != nil {
		log.Error().Err(err).Msg("Could not open dg session")
	}

	log.Info().Msg("Discord session opened")

	log.Info().Msg("Initializing")
	steamProfile := SteamProfile{}
	cloud := Cloud{}
	workstation := Workstation{}
	valProfile := ValProfile{}

	cloud.check()
	steamProfile.update()
	valProfile.update()
	log.Info().Msg("Initialized")

	// run steamProfile.update() every 5 minutes
	go func() {
		for {
			log.Debug().Msg("Updating")
			steamProfile.update()
			log.Debug().Msg("steam updated")
			cloud.check()
			log.Debug().Msg("updown updated")
			valProfile.update()
			log.Debug().Msg("valorant updated")
			time.Sleep(5 * time.Minute)
		}
	}()

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/steam", func(w http.ResponseWriter, r *http.Request) {
		// output steamProfile in json
		steamProfile.Mu.Lock()
		defer steamProfile.Mu.Unlock()
		out, err := json.Marshal(steamProfile)
		if err != nil {
			log.Error().Err(err).Msg("Could not marshal steamProfile")
		}
		fmt.Fprintf(w, string(out))
	})

	r.Get("/steam.text", func(w http.ResponseWriter, r *http.Request) {
		// output steamProfile in json
		steamProfile.Mu.Lock()
		defer steamProfile.Mu.Unlock()
		out := ""
		out += "Status: " + steamProfile.PersonaState + "\n"
		if steamProfile.IsGaming {
			out += "Game: " + steamProfile.GameExtraInfo + "\n"
		}
		out += "Last logoff: " + steamProfile.LastLogoff + "\n"

		fmt.Fprintf(w, out)
	})

	r.Get("/ezricloud", func(w http.ResponseWriter, r *http.Request) {
		// output cloud in json
		cloud.Mu.Lock()
		defer cloud.Mu.Unlock()
		out, err := json.Marshal(cloud)
		if err != nil {
			log.Error().Err(err).Msg("Could not marshal cloud")
		}
		fmt.Fprintf(w, string(out))
	})

	r.Get("/ezricloud.text", func(w http.ResponseWriter, r *http.Request) {
		// output cloud in json
		cloud.Mu.Lock()
		defer cloud.Mu.Unlock()
		out := ""
		if cloud.IsDown {
			out += "EzriCloud: Outage since " + cloud.DownSince + "\n"
		} else {
			out += "EzriCloud: All Systems Operational\n"
		}
		fmt.Fprintf(w, out)
	})

	r.Post("/ide0", func(w http.ResponseWriter, r *http.Request) {
		// verify key
		key := r.Header.Get("Authorization")
		if key != ideKey {
			log.Error().Msg("Invalid key")
			return
		}

		// receive body into ide0
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error().Err(err).Msg("Could not read body")
		}
		ide0Bytes := body
		ide0 := string(ide0Bytes)
		workstation.Mu.Lock()
		defer workstation.Mu.Unlock()
		workstation.Status = ide0
		workstation.LastUpdate = time.Now()
		log.Debug().Msg("ide0: " + workstation.Status)
	})

	r.Get("/ide0", func(w http.ResponseWriter, r *http.Request) {
		// output ide0 in json
		workstation.Mu.Lock()
		defer workstation.Mu.Unlock()
		out, err := json.Marshal(workstation)
		if err != nil {
			log.Error().Err(err).Msg("Could not marshal ide0")
		}
		fmt.Fprintf(w, string(out))
	})

	r.Get("/ide0.text", func(w http.ResponseWriter, r *http.Request) {
		workstation.Mu.Lock()
		defer workstation.Mu.Unlock()

		out := ""
		if workstation.Status != "" {
			out += workstation.Status + "\n"
		}
		out += "Last Update: " + workstation.LastUpdate.Format("15:04:05 MST")

		fmt.Fprintf(w, out)
	})

	r.Get("/val.text", func(w http.ResponseWriter, r *http.Request) {
		valProfile.Mu.Lock()
		defer valProfile.Mu.Unlock()

		out := ""
		out += "Username: " + valProfile.Name + "#" + valProfile.Tag + "\n"
		out += "Region: NA \n"
		out += "Elo: " + strconv.Itoa(valProfile.Elo) + "\n"
		out += "Current Rank: " + valProfile.CurrentRank + "\n"
		out += "Highest Rank: " + valProfile.HighestRank + "\n"

		fmt.Fprintf(w, out)
	})

	r.Get("/discord", func(w http.ResponseWriter, r *http.Request) {
		// output discord in json
		discord.Mu.Lock()
		defer discord.Mu.Unlock()
		out, err := json.Marshal(discord.Status)
		if err != nil {
			log.Error().Err(err).Msg("Could not marshal discord")
		}
		fmt.Fprintf(w, string(out))
	})

	r.Get("/discord.text", func(w http.ResponseWriter, r *http.Request) {
		discord.Mu.Lock()
		defer discord.Mu.Unlock()

		out := ""
		status := discord.Status
		discordOut := ""
		if status.StatusDesk != "" {
			discordOut += "Desktop: " + status.StatusDesk + "\n"
		}
		if status.StatusWeb != "" {
			discordOut += "Web: " + status.StatusWeb + "\n"
		}
		if status.StatusMobile != "" {
			discordOut += "Mobile: " + status.StatusMobile + "\n"
		}
		if discordOut == "" {
			discordOut = "Currently offline" + "\n"
		}
		if status.CustomStatus != "" {
			discordOut += "Custom Status: " + status.CustomStatus + "\n"
		}
		out += discordOut

		out += "Last Update: " + status.UpdatedAt
		fmt.Fprintf(w, out)
	})

	http.ListenAndServe(":8080", r)

	// Handling SIGINT
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	dg.Close()

}
