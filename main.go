package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Map holding all Websocket clients and the endpoints they are subscribed to
var clients = make(map[string][]*websocket.Conn)
var upgrader = websocket.Upgrader{}

// Message which will be sent as JSON to Websocket clients
type Message struct {
	Headers  map[string]string `json:"headers"`
	Endpoint string            `json:"endpoint"`
	Data     interface{}       `json:"data"`
}

func handleHook(w http.ResponseWriter, r *http.Request, endpoint string) {
	msg := Message{}
	logEntry := log.WithField("endpoint", endpoint)

	// Transfer headers to response
	msg.Headers = make(map[string]string)
	for k, v := range r.Header {
		msg.Headers[k] = v[0]
	}

	// Set endpoint on response
	msg.Endpoint = endpoint

	// Read body of request
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)

	// If request is JSON, unmarshal and save to response. Otherwise just save as string.
	if r.Header.Get("Content-Type") == "application/json" {
		json.Unmarshal(buf.Bytes(), &msg.Data)
	} else {
		msg.Data = buf.Bytes()
	}

	// Get all clients listening to the current endpoint
	conns := clients[endpoint]

	if conns != nil {
		for i, conn := range conns {
			if conn.WriteJSON(msg) != nil {
				// Remove client and close connection if sending failed
				conns = append(conns[:i], conns[i+1:]...)
				conn.Close()
			}
		}
	}

	clients[endpoint] = conns

	twitchMessageType := r.Header.Get("Twitch-Eventsub-Message-Type")

	if twitchMessageType == "webhook_callback_verification" {

	//	fmt.Println(msg.Data)

		body := msg.Data




	challenge := body.(map[string]interface{})["challenge"]

	fmt.Println(challenge)
	str := fmt.Sprint(challenge)
	w.Write([]byte(str))

	}

	logEntry.WithField("clients", len(conns)).Infoln("Hook broadcasted")
}

func handleClient(w http.ResponseWriter, r *http.Request, endpoint string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	logEntry := log.WithField("endpoint", endpoint)

	if err != nil {
		logEntry.Println(err)
		// Send Upgrade required response if upgrade fails
		w.WriteHeader(426)
		return
	}

	// Add client to endpoint slice
	clients[endpoint] = append(clients[endpoint], conn)

	logEntry.WithField("clients", len(clients[endpoint])).Infoln("Client connected")
}

func handler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")

	/**
	 * Check prefix of URL path:
	 * 	/hook is used for webhooks and requests will be broadcasted to all listening clients.
	 * 	/socket is used for connect a new socket client
	 */
	if strings.HasPrefix(path, "/hook") {
		handleHook(w, r, strings.TrimPrefix(path, "/hook"))
	} else if strings.HasPrefix(path, "/socket") {
		handleClient(w, r, strings.TrimPrefix(path, "/socket"))
	} else {
		log.WithField("path", r.URL.Path).Warnln("404 Not found")
		w.WriteHeader(404)
	}

	//origin := r.Header.Get("Origin")
	twitchMessageId := r.Header.Get("Twitch-Eventsub-Message-Id")


	if twitchMessageId != "" {
		twitchTimeStamp := r.Header.Get("Twitch-Eventsub-Message-Timestamp")
		twitchSignature := r.Header.Get("Twitch-Eventsub-Message-Timestamp")

		// 2ccc3407-8f4b-40e6-9681-0059d87221932021-05-23T07:03:42.885277459Z6481a80f-d5da-4a1b-952b-ef16f29ef2b4
		//twitchBody := new(bytes.Buffer)
		//twitchBody.ReadFrom(r.Body)
		//twitchData := twitchBody.Bytes()

		//fmt.Println (twitchBody)
		/*twitchBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			return
		}*/

		/*twitchbody := new(bytes.Buffer)
		twitchbody.ReadFrom(r.Body)*/

		// twitchMessage := []byte(twitchMessageId)+[]byte(twitchTimeStamp)+twitchBody
		//twitchMessage := append([]byte(twitchMessageId), []byte(twitchTimeStamp)...)
		//twitchMessageX := append(twitchMessage, twitchBody...)

	    responseData, err := ioutil.ReadAll(r.Body)
    if err != nil {
        log.Fatal(err)
    }
		//twitchBody := newBody.Bytes()
		
		twitchMessageX := append ([]byte(twitchMessageId), []byte(twitchTimeStamp)...)
		twitchMessage :=  append (twitchMessageX, responseData...)

		signature := []byte(twitchSignature)
		secret := []byte("testhello123")
		valid := ValidMAC(twitchMessage, signature, secret)

		fmt.Println(valid)
		//if valid {fmt.Printf("Valid HMAC? %t\n")}
	}
	log.Print(r)

}

func ValidMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}

func main() {
	// Get command line options --address and --port
	address := flag.String("address", "", "Address to bind to.")
	port := flag.Int("port", 1234, "Port to bind to. Default: 1234")
	flag.Parse()
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	http.HandleFunc("/", handler)

	// Start HTTP server
	log.Infof("Sockethook is ready and listening at port %d ✅", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *address, *port), nil))
}
