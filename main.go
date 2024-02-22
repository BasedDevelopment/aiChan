/*
   Copyright (C) 2022 Tianyu Zhu eric@ericz.me

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	configPath = "config.toml"
)

var (
	k      = koanf.New(".")
	parser = toml.Parser()
	rdb    = redis.NewClient(&redis.Options{
		Addr:     "172.17.0.2:6379",
		Password: "",
		DB:       0,
	})
	ctx = context.Background()
)

func init() {
	// Init logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// Load config
	if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}
	if k.String("discord.token") == "" {
		log.Fatal().Msg("Configuration: token is required")
	}
}

func main() {
	// Init bot
	dg, err := discordgo.New("Bot " + k.String("discord.token"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Discord session")
		return
	}

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
		return
	}

	dg.Identify.Intents |= discordgo.IntentsGuildMessages

	dg.AddHandler(newMsg)

	err = dg.Open()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open connection")
		return
	}

	log.Info().Msg("Bot up")

	// Handling SIGINT
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	dg.Close()
}
