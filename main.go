package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	nfctype4 "github.com/hsanjuan/go-nfctype4"
	"github.com/hsanjuan/go-nfctype4/drivers/libnfc"
	yubigo "github.com/hsanjuan/yubigo"
)

var waitNoTargets = 400 * time.Millisecond
var waitError = 5 * time.Second

var yubikeyPayloadURLPrefixes = []string{
	"https://my.yubico.com/neo/",
}

// Config sets the necessary parameters to operate the Nuki lock
// and allow certain Yubikeys to open the door.
type Config struct {
	NukiBridgeAddr     string
	NukiBridgeToken    string
	NukiLockID         string
	AuthorizedYubikeys []string
	YubicloudClientID  string
	YubicloudSecretKey string
}

type YubiNuki struct {
	cfg      *Config
	yubiauth *yubigo.YubiAuth
	device   *nfctype4.Device
}

func New(cfg *Config) (*YubiNuki, error) {
	yauth, err := yubigo.NewYubiAuth(cfg.YubicloudClientID, cfg.YubicloudSecretKey)
	if err != nil {
		return nil, err
	}

	device := nfctype4.New(&libnfc.Driver{})

	return &YubiNuki{
		cfg:      cfg,
		yubiauth: yauth,
		device:   device,
	}, nil
}

// ReadAndAuthorizeYubikey reads a token from the NFC Reader,
// check the ID is among the AuthorizedYubikeys, checks the
// validity of the token against the Yubicloud servers and,
// if successful, opens the Nuki lock.
func (yn *YubiNuki) ReadAndAuthorizeYubikey() error {
	token, err := yn.readToken()
	if err != nil {
		return err
	}
	ok := yn.authorizeTokenID(token)
	if !ok {
		return fmt.Errorf("unknown token ID for: %s", token)
	}
	ok2, err := yn.verifyToken(token)
	if err != nil {
		return err
	}
	if !ok2 {
		return fmt.Errorf("BAD YUBIKEY STATUS: %s", token)
	}
	log.Println("Verified token: ", token)

	return yn.openDoor()
}

func (yn *YubiNuki) readToken() (string, error) {
	ndefMessage, err := yn.device.Read()
	if err != nil {
		return "", err
	}

	if len(ndefMessage.Records) < 1 {
		return "", errors.New("no ndef records present")
	}

	url := ndefMessage.Records[0].Payload.String()
	token := parseYubikeyURL(url)
	if token == "" {
		return "", errors.New("unknown token url")
	}
	log.Println("Read token: ", token)
	return token, nil
}

func (yn *YubiNuki) authorizeTokenID(token string) bool {
	if len(token) <= 32 {
		return false
	}

	id := token[0 : len(token)-32]
	for _, user := range yn.cfg.AuthorizedYubikeys {
		if user == id {
			log.Println("authorized user: ", id)
			return true
		}
	}
	log.Println("Not authorized token ID: ", id)
	return false
}

func (yn *YubiNuki) verifyToken(token string) (bool, error) {
	_, ok, err := yn.yubiauth.Verify(token)
	return ok, err
}

type nukiResp struct {
	Success         bool `json:"success"`
	BatteryCritical bool `json:"batteryCritical"`
}

func (yn *YubiNuki) openDoor() error {
	q := url.Values{}
	q.Set("nukiId", yn.cfg.NukiLockID)
	q.Set("noWait", "0")
	q.Set("action", "3") // unlatch
	q.Set("token", yn.cfg.NukiBridgeToken)

	u, err := url.Parse(fmt.Sprintf("http://%s/lockAction", yn.cfg.NukiBridgeAddr))
	if err != nil {
		return err
	}

	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	js, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	status := &nukiResp{}
	err = json.Unmarshal(js, status)
	if err != nil {
		return err
	}

	fmt.Println("Opened door. Success: ", status.Success)
	return nil
}

// parseYubikeyURL returns the token extracted from the Yubikey URL
func parseYubikeyURL(ykURL string) string {
	//log.Println(ykURL)
	for _, prefix := range yubikeyPayloadURLPrefixes {
		if tok := strings.TrimPrefix(ykURL, prefix); tok != ykURL {
			return tok
		}
	}
	return ""
}

// Command line flags
var (
	configFlag string
)

func init() {
	flag.StringVar(&configFlag, "config", "yubinuki.json", "Path to config file")
	flag.Parse()
}

func main() {
	f, err := os.Open(configFlag)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	cfg := &Config{}
	fbytes, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	err = json.Unmarshal(fbytes, cfg)
	if err != nil {
		log.Fatal(err)
	}

	yn, err := New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	for {
		err = yn.ReadAndAuthorizeYubikey()
		if err == libnfc.ErrNoTargetsDetected {
			time.Sleep(waitNoTargets)
			continue
		}
		if err != nil {
			log.Println(err)
			time.Sleep(waitError)
			continue
		}
		time.Sleep(10 * time.Second)
	}
}
