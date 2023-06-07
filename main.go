package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/mehanon/telebot"
	"github.com/mehanon/telebot/middleware"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

const DefaultConfigPath = "./cfg.json"

var ttgRegex = regexp.MustCompile("@.*?_20[0-9]{2}-[0-1][0-9]-[0-3][0-9]\\.mp4")
var tikmehRegex = regexp.MustCompile(".*?_20[0-9]{2}-[0-1][0-9]-[0-3][0-9]_[0-9]+\\.mp4")

type Config struct {
	Token     string  `json:"token"`
	AdminList []int64 `json:"admin-list"`
}

func GuessName(filename string) string {
	if ttgRegex.MatchString(filename) {
		return filename[1:strings.LastIndex(filename, "_")]
	} else if tikmehRegex.MatchString(filename) {
		noId := filename[0:strings.LastIndex(filename, "_")]
		return noId[0:strings.LastIndex(noId, "_")]
	} else {
		return ""
	}
}

func main() {
	configPath := flag.String("cfg", DefaultConfigPath,
		"path to the config file (may be useful to run multiple bots in parallel)")
	debug := flag.Bool("debug", false, "print debug log")
	flag.Parse()

	buffer, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalln(err)
	}
	var cfg Config
	err = json.Unmarshal(buffer, &cfg)
	if err != nil {
		log.Fatalln(err)
	}

	bot, err := telebot.NewBot(telebot.Settings{
		Token:   cfg.Token,
		Verbose: *debug,
		Poller:  &telebot.LongPoller{Timeout: 30 * time.Second},
		OnError: func(err error, ctx telebot.Context) {
			for _, admin := range cfg.AdminList {
				_, _ = ctx.Bot().Send(&telebot.Chat{ID: admin},
					fmt.Sprintf("Error :c\n\n	%s\n\nAt chat: '%s' [%d]", err.Error(), ctx.Chat().Title, ctx.Chat().ID))
			}
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	bot.HandleAlbum(func(cs []telebot.Context) error {
		names := make([]string, 0)

		for _, ctx := range cs {
			if ctx.Message() != nil && ctx.Message().Video != nil {
				name := GuessName(ctx.Message().Video.FileName)
				contains := false
				for _, known := range names {
					if known == name {
						contains = true
						break
					}
				}
				if !contains && name != "" {
					names = append(names, name)
				}
			}
		}

		if len(names) != 0 {
			return cs[0].Reply(strings.Join(names, "\n"))
		}

		return nil
	})

	admin := bot.Group()
	admin.Use(middleware.Whitelist(cfg.AdminList...))
	admin.Handle("/shutdown", func(ctx telebot.Context) error {
		if len(ctx.Args()) > 0 && strings.ToLower(ctx.Args()[0]) == "please" {
			_ = ctx.Reply("shutting down...")
			os.Exit(0)
		} else {
			return ctx.Reply("say 'please', be gentle")
		}
		return nil
	})

	bot.Start()
}
