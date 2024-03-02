/*
   Copyright (C) 2022-2024 Ezri Zhu me@ezrizhu.com

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
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func newMsg(s *discordgo.Session, m *discordgo.MessageCreate) {
	bannedUsers := k.Strings("discord.bannedUsers")
	admins := k.Strings("discord.admins")

	if m.Author.ID == s.State.User.ID {
		return
	}
	if len(m.Content) <= 4 {
		return
	}
	if (m.Content[:2] != "ai") && (m.Content[:2] != "Ai") {
		return
	}
	for _, user := range bannedUsers {
		if m.Author.ID == user {
			s.ChannelMessageSendReply(m.ChannelID, "You are banned from using this bot.", m.Reference())
			return
		}
	}

	msg := m.Content[3:]
	if msg[0] == ' ' {
		msg = msg[1:]
	}

	user := m.Author.Username

	userIsAdmin := false
	for _, user := range admins {
		if m.Author.ID == user {
			userIsAdmin = true
		}
	}

	if m.Content[2] == '!' {
		log.Info().
			Str("msg", msg).
			Str("user", user).
			Msg("AI Chat Request")
		if proceed := mod(s, m, msg); proceed == true {
			chat(s, m, msg, userIsAdmin)
		}
	}

	if m.Content[2] == '?' {
		log.Info().
			Str("msg", msg).
			Str("user", user).
			Msg("AI Draw Request")
		if proceed := mod(s, m, msg); proceed == true {
			draw(s, m, msg)
		}
	}

	if m.Content[2] == '#' {
		if userIsAdmin {
			log.Info().
				Str("msg", msg).
				Str("user", user).
				Msg("AI Chat Request")
			chat(s, m, msg, userIsAdmin)
		}
		// Not admin
		if !userIsAdmin {
			if _, err := s.ChannelMessageSendReply(m.ChannelID, "You are not allowed to use this command.", m.Reference()); err != nil {
				log.Error().Err(err).Msg("Chat: Error sending discord message")
			}
		}
	}

	if m.Content[2] == '.' {
		if userIsAdmin {
			log.Info().
				Str("msg", msg).
				Str("user", user).
				Msg("AI Admin Request")
			changeSys(s, m, msg)
		}
		// Not admin
		if !userIsAdmin {
			if _, err := s.ChannelMessageSendReply(m.ChannelID, "You are not allowed to use this command.", m.Reference()); err != nil {
				log.Error().Err(err).Msg("Chat: Error sending discord message")
			}
		}
	}
}
