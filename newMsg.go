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
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func newMsg(s *discordgo.Session, m *discordgo.MessageCreate) {
	msgChan := k.Strings("discord.msgChan")
	bannedWords := k.Strings("discord.bannedWords")
	bannedUsers := k.Strings("discord.bannedUsers")

	var rightChan bool = false
	for _, channel := range msgChan {
		if m.ChannelID == channel {
			rightChan = true
			break
		}
	}
	if rightChan != true {
		return
	}
	if m.Author.ID == s.State.User.ID {
		return
	}
	for _, user := range bannedUsers {
		if m.Author.ID == user {
			return
		}
	}
	isPrefix, _ := regexp.MatchString(`^ai`, m.Content)
	if isPrefix == false {
		return
	}
	if len(m.Content) <= 4 {
		return
	}
	msg := m.Content[3:]
	if msg[0] == ' ' {
		msg = msg[1:]
	}
	user := m.Author.Username + "#" + m.Author.Discriminator
	for _, word := range bannedWords {
		if strings.Contains(msg, word) {
			s.MessageReactionAdd(m.ChannelID, m.Reference().MessageID, "⚠️")
			log.Warn().
				Str("user", user).
				Str("msg", msg).
				Str("word", word).
				Msg("Banned word detected")
			if _, err := s.ChannelMessageSendReply(m.ChannelID, "List: This message has been flagged as inappropriate. This incident will be reported.", m.Reference()); err != nil {
				log.Error().Err(err).Msg("Chat: Error sending discord message")
			}
			return
		}
	}

	isChat, _ := regexp.MatchString(`^ai!`, m.Content)
	isDraw, _ := regexp.MatchString(`^ai?`, m.Content)

	switch {
	case isChat:
		log.Info().
			Str("msg", msg).
			Str("user", user).
			Msg("AI Chat Request")
		if proceed := mod(s, m, msg); proceed == true {
			chat(s, m, msg)
		}
	case isDraw:
		log.Info().
			Str("msg", msg).
			Str("user", user).
			Msg("AI Draw Request")
		if proceed := mod(s, m, msg); proceed == true {
			draw(s, m, msg)
		}
	}
}
