package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/davidrjonas/hipchat-addon"
	"github.com/jmoiron/jsonq"
)

var urlBase string
var host string
var port int
var stateFilename string

func init() {
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.DebugLevel)

	flag.StringVar(&urlBase, "url", "", "The base URL, defaults to the host and port")
	flag.StringVar(&host, "host", "127.0.0.1", "The IP address on which to listen")
	flag.IntVar(&port, "port", 3000, "The port on which to listen")
	flag.StringVar(&stateFilename, "state-file", "state", "The file to read/store state information")
}

func onYraQuery(a *addon.HipchatAddon, installation *addon.Installation, webhook *addon.WebHook, event map[string]interface{}) error {

	var msg string

	jq := jsonq.NewQuery(event)

	name, err := jq.String("item", "message", "from", "mention_name")

	if err != nil {
		logrus.Error(err)
	}

	switch name {
	case "kbussche":
		msg = "@kbussche You're a qwerty."
	case "djonas":
		if s, _ := jq.String("item", "message", "message"); strings.Contains(s, "debug") {
			b, err := json.MarshalIndent(event, "", "  ")
			if err != nil {
				msg = fmt.Sprintf("Error marshalling json: %q", err)
			} else {
				msg = "@djonas " + string(b[:])
			}
		} else {
			// No error and not a query!
			return nil
		}
	default:
		if name != "" {
			msg = "@" + name + " You're a queryf."
		} else {
			msg = "You're a queryf."
		}
	}

	notification := &addon.Notification{
		MessageFormat: "text",
		Notify:        true,
		Message:       msg,
	}

	if err := a.SendNotification(installation, notification); err != nil {
		logrus.Error(err)
		return err
	}

	return nil
}

func url(resource string) string {
	return urlBase + resource
}

func main() {
	flag.Parse()

	host = os.Getenv("OPENSHIFT_GO_IP")
	port, _ := strconv.Atoi(os.Getenv("OPENSHIFT_GO_PORT"))
	stateFilename = os.Getenv("OPENSHIFT_DATA_DIR") + "state"
	urlBase = "https://ohsnap-davidrjonas.rhcloud.com"

	if urlBase == "" {
		urlBase = fmt.Sprintf("http://%s:%d", host, port)
	}

	a := addon.NewWithStateFile(
		&addon.CapabilitiesDescriptor{
			Name:        "OhSnap",
			Description: "Snappy replies when necessary.",
			Key:         "ohsnap",
			Vendor: &addon.Vendor{
				Name: "davidrjonas",
				Url:  "https://github.com/davidrjonas/ohsnap",
			},
			Links: &addon.Links{
				Homepage: "https://github.com/davidrjonas/ohsnap",
				Self:     url("/capabilities.json"),
			},
			Capabilities: &addon.Capabilities{
				HipchatApiConsumer: &addon.HipchatApiConsumer{
					Scopes: []string{"send_notification"},
				},
				Installable: &addon.Installable{
					AllowGlobal: false,
					AllowRoom:   true,
					CallbackUrl: url("/install"),
				},
				WebHook: []*addon.WebHook{&addon.WebHook{
					Event:          "room_message",
					Pattern:        "(?i)quer(y|ies)",
					Authentication: "jwt",
					Name:           "Yraquery",
					Url:            url("/webhook/0"),

					Callback: onYraQuery,
				}},
			},
		},
		stateFilename,
		addon.Logger(logrus.StandardLogger()),
	)

	logrus.Infof("Saving state to file '%s'", stateFilename)
	logrus.Infof("Starting server on %s:%d for url %s", host, port, urlBase)

	a.Serve(fmt.Sprintf("%s:%d", host, port))
}
