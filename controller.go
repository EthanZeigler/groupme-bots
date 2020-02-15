package main

import (
	"fmt"
	"github.com/ethanzeigler/groupme/botserver"
	"github.com/ethanzeigler/groupme/gmbots/meme"
	"github.com/sirupsen/logrus"
	"os"
)

type GlobalConfig struct {
}

type GroupConfigEntry struct {
	GroupID   string `json:"group_id"`
	BotID     string `json:"bot_id"`
	BotUserID string `json:"bot_user_id"`
	IsAlpha   bool   `json:"is_alpha"`
}

type MemeMachineConfig struct {
	GroupEntries []GroupConfigEntry `json:"groups"`
}

type Config struct {
	Global GlobalConfig `json:"global"`
}

func main() {
	memeChannel := meme.MakeMemeChannel()
	srv := botserver.NewInstance()
	srv.RegisterChannel(&memeChannel)
	srv.Log.Level = logrus.DebugLevel
	err := srv.ConfigureFromFile("config.json")
	if err != nil {
		// Something is very wrong. Die.
		os.Exit(1)
	}
	if len(os.Args) > 1 {
		if os.Args[1] == "--debug" {
			_ = srv.StartDebug(os.Stdin)
		} else {
			fmt.Println("Invalid argument.")
		}
	} else {
		_ = srv.Start()
	}
}
