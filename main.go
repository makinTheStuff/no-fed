package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fiatjaf/litepub"
	"github.com/fiatjaf/relayer"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

type Settings struct {
	ServiceName   string `envconfig:"SERVICE_NAME" required:"true"`
	ServiceURL    string `envconfig:"SERVICE_URL" required:"true"`
	Port          string `envconfig:"PORT" required:"true"`
	PostgresURL   string `envconfig:"DATABASE_URL" required:"true"`
	IconSVG       string `envconfig:"ICON"`
	PrivateKeyPEM string `envconfig:"PRIVATE_KEY"`
	PrivateKey    *rsa.PrivateKey
	PublicKey     rsa.PublicKey
	PublicKeyPEM  string
}

var s Settings
var pub litepub.LitePub
var pg *sqlx.DB
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})

func main() {
	err := envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	// key stuff (needed for the activitypub integration)
	if s.PrivateKeyPEM != "" {
		s.PrivateKeyPEM = strings.Replace(s.PrivateKeyPEM, "$$", "\n", -1)
		decodedskpem, _ := pem.Decode([]byte(s.PrivateKeyPEM))

		sk, err := x509.ParsePKCS1PrivateKey(decodedskpem.Bytes)
		if err != nil {
			log.Fatal().Err(err).Msg("couldn't process private key pem.")
		}

		s.PrivateKey = sk
		s.PublicKey = sk.PublicKey

		key, err := x509.MarshalPKIXPublicKey(&sk.PublicKey)
		if err != nil {
			log.Fatal().Err(err).Msg("couldn't marshal public key to pem.")
		}
		s.PublicKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: key,
		}))
	}

	pub = litepub.LitePub{
		PrivateKey: s.PrivateKey,
	}

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log = log.With().Timestamp().Logger()

	// postgres connection
	pg, err = sqlx.Connect("postgres", s.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't connect to postgres")
	}

	// define routes
	relayer.Router.Path("/icon.svg").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/svg+xml")
			fmt.Fprint(w, s.IconSVG)
			return
		})

	relayer.Router.Path("/pub").HandlerFunc(pubInbox)
	relayer.Router.Path("/pub/user/{pubkey:[\\d\\w-]+}").Methods("GET").HandlerFunc(pubUserActor)
	relayer.Router.Path("/pub/user/{pubkey:[\\d\\w-]+}/following").Methods("GET").HandlerFunc(pubUserFollowing)
	relayer.Router.Path("/pub/user/{pubkey:[\\d\\w-]+}/followers").Methods("GET").HandlerFunc(pubUserFollowers)
	relayer.Router.Path("/pub/user/{pubkey:[\\d\\w-]+}/outbox").Methods("GET").HandlerFunc(pubOutbox)
	relayer.Router.Path("/pub/note/{id}").Methods("GET").HandlerFunc(pubNote)
	relayer.Router.Path("/.well-known/webfinger").HandlerFunc(webfinger)

	relayer.Router.PathPrefix("/").Methods("GET").Handler(http.FileServer(http.Dir("./static")))

	// start the relay/http server
	relayer.Start(Relay{})
}

func switchHTMLJSON(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if strings.Contains(r.Header.Get("Accept"), "text/html") {
			http.ServeFile(w, r, "static/index.html")
		} else {
			w.Header().Set("Content-Type", "application/json")
			next(w, r)
		}
	}
}
