package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

//cacheSize sets the number of shibes to cache
const cacheSize = 50

var (
	//Should the bot respond in these channels? Not included means false
	channels = map[string]bool{
		"428290425384599574": true,
	}
	shibeCache chan ([]byte)
)

func obtainShibe() []byte {
	resp, err := http.Get("http://shibe.online/api/shibes")
	if err != nil {
		log.Println("Err:", err.Error())
	}
	decoder := json.NewDecoder(resp.Body)
	shibeURLArray := make([]string, 1)
	decoder.Decode(&shibeURLArray)
	shibePicResp, err := http.Get(shibeURLArray[0])
	if err != nil {
		log.Println("Err:", err.Error())
	}
	shibePic, err := ioutil.ReadAll(shibePicResp.Body)
	if err != nil {
		log.Println("Err:", err.Error())
	}
	return shibePic
}

func shibeCacheWorker() {
	for {
		if len(shibeCache) != cacheSize {
			//Then we need more shibes!
			shibesRequired := cacheSize-len(shibeCache)
			log.Println("Insufficient Shibes Detected, accumulating. Required:",shibesRequired )
			for i := 0; i < shibesRequired; i++ {
				shibeCache <- obtainShibe()
				//time.Sleep(time.Millisecond * 50)
			}
			log.Println("Shibe cache rebuilt")
		}
	}
}
func pushCache() {
	shibeCache <- obtainShibe()
}

func main() {
	shibeCache = make(chan []byte, cacheSize)
	var token string
	flag.StringVar(&token, "t", "", "Bot token for the bot to use")
	flag.Parse()
	if token == "" {
		log.Fatalln("No token provided!")
		return
	}
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	discord.AddHandler(messageHandler)
	log.Println("Bot loading...")
	err = discord.Open()
	if err != nil {
		log.Fatalln("Error: Couldn't open connection.", err.Error())
	}
	//go shibeCacheWorker()
	//log.Println("Shibe cache worker started")
	for i := 0; i < cacheSize; i++ {
		pushCache()
	}
	log.Println("Shibe initialisation complete")
	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	discord.Close()
}
func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	//Ignore messages from the bot itself
	log.Println("Message:\t", m.Content)
	startTime := time.Now()
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Author.Bot { //Ignore other bots
		return
	}
	if m.Message.Content[0] != '^' {
		return
	}
	if m.Message.Content[0:6] != "^shibe" {
		return
	}
	//Find the guildmember attached to that user
	allowed, inMap := channels[m.ChannelID]
	if allowed && inMap { //If the bot should respond in that channel
		log.Println("Cache empty:\t",(len(shibeCache)==0))
		shibeReader := bytes.NewReader(<-shibeCache) //Pull shiba from the cache
		s.ChannelFileSend(m.ChannelID, "shibe.jpg", shibeReader)
		log.Println("Time to process:", time.Now().Sub(startTime))
		go pushCache()
	}
}
