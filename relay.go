package main

import (
	"fmt"

	"encoding/json"
	"net/http"

	"github.com/fiatjaf/relayer"
	"github.com/nbd-wtf/go-nostr"
)

type Relay struct {
	storage Storage
}

func (r *Relay) Name() string {
	return "no-fed"
}

func (r *Relay) Storage() relayer.Storage {
	return r.storage
}

func (r *Relay) OnInitialized(server *relayer.Server) {
	// define routes
	server.Router().Path("/icon.svg").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/svg+xml")
			fmt.Fprint(w, s.IconSVG)
			return
		})

	server.Router().Path("/pub").Methods("POST").HandlerFunc(pubInbox)
	server.Router().Path("/pub/user/{pubkey:[A-Fa-f0-9]{64}}").Methods("GET").HandlerFunc(pubUserActor)
	server.Router().Path("/pub/user/{pubkey:[A-Fa-f0-9]{64}}/following").Methods("GET").HandlerFunc(pubUserFollowing)
	server.Router().Path("/pub/user/{pubkey:[A-Fa-f0-9]{64}}/followers").Methods("GET").HandlerFunc(pubUserFollowers)
	server.Router().Path("/pub/user/{pubkey:[A-Fa-f0-9]{64}}/outbox").Methods("GET").HandlerFunc(pubOutbox)
	server.Router().Path("/pub/note/{id:[A-Fa-f0-9]{64}}").Methods("GET").HandlerFunc(pubNote)
	server.Router().Path("/.well-known/webfinger").HandlerFunc(webfinger)
	server.Router().Path("/.well-known/nostr.json").HandlerFunc(handleNip05)

	server.Router().PathPrefix("/").Methods("GET").Handler(http.FileServer(http.Dir("./static")))
}

func (relay Relay) Init() error {
	filters := relayer.GetListeningFilters()
	for _, filter := range filters {
		log.Print(filter)
	}

	return nil
}

func (r *Relay) AcceptEvent(evt *nostr.Event) bool {
	// block events that are too large
	jsonb, _ := json.Marshal(evt)
	if len(jsonb) > 10000 {
		return false
	}

	return true
}
