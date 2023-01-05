package main

import (
	"crypto/rsa"
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
	ServiceName string `envconfig:"SERVICE_NAME" required:"true"`
	ServiceURL  string `envconfig:"SERVICE_URL" required:"true"`
	RelayURL    string
	Port        string `envconfig:"PORT" required:"true"`
	PostgresURL string `envconfig:"DATABASE_URL" required:"true"`
	IconSVG     string `envconfig:"ICON"`
	Secret      string `envconfig:"SECRET"`

	PrivateKey   *rsa.PrivateKey
	PublicKeyPEM string
}

func (s *Settings) generateKeys() (err error) {
	// key stuff (needed for the activitypub integration)
	var seed [4]byte
	copy(seed[:], []byte(s.Secret))

	if s.PrivateKey, err = litepub.GeneratePrivateKey(seed); err != nil {
		log.Fatal().Err(err).Msg("error deriving private key")
		return err
	}
	if s.PublicKeyPEM, err = litepub.PublicKeyToPEM(&s.PrivateKey.PublicKey); err != nil {
		log.Fatal().Err(err).Msg("error deriving public key")
		return err
	}
	return nil
}

func (s *Settings) setRelayUrl() {
	s.RelayURL = strings.Replace(s.ServiceURL, "http", "ws", 1)
}

var (
	s   Settings
	pg  *sqlx.DB
	log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
)

func main() {
	// logger
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log = log.With().Timestamp().Logger()
	var err error

	if err = envconfig.Process("", &s); err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
		return
	}
	if err = s.generateKeys(); err != nil {
		return
	}
	if pg, err = initDB(s.PostgresURL); err != nil {
		return
	}
	cacheExpirer()

	rl := Relay{}
	if err = relayer.Start(&rl); err != nil {
		log.Fatal().Err(err).Msg("!!server termiinate!!")
	}
}
