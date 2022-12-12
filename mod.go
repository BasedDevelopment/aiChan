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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func mod(s *discordgo.Session, m *discordgo.MessageCreate, msg string) (proceed bool) {
	proceed = false

	// Token
	token := k.String("ai.mod.token")
	var bearer = "Bearer " + token

	// Request Struct
	request := map[string]interface{}{
		"input": msg,
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Error: "+err.Error(), m.Reference()); err != nil {
			log.Error().Err(err).Msg("Mod: Error sending discord message")
		}
		return
	}

	// Send request
	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/moderations", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Authorization", bearer)
	httpReq.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	defer resp.Body.Close()
	if err != nil {
		_, err := s.ChannelMessageSendReply(m.ChannelID, "Mod: http err", m.Reference())
		if err != nil {
			log.Error().Err(err).Msg("Mod: Error sending discord message")
		}
		return
	}

	// Decode response
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	user := m.Author.Username + "#" + m.Author.Discriminator

	// Handle response
	if result == nil {
		log.Error().Str("user", user).Str("prompt", msg).Msg("Mod: result is nil")
		_, err := s.ChannelMessageSendReply(m.ChannelID, "Mod: results is nil", m.Reference())
		if err != nil {
			log.Error().Err(err).Msg("Mod: Error sending discord message")
		}
		return
	}
	if result["results"] == nil {
		resultStr := fmt.Sprintf("%#v", result)
		log.Warn().Str("user", user).Str("prompt", msg).Str("resp", resultStr).Msg("Mod: results[results] is nil")
		_, err := s.ChannelMessageSendReply(m.ChannelID, "Mod: results[results] is nil", m.Reference())
		if err != nil {
			log.Error().Err(err).Msg("Mod: Error sending discord message")
		}
		return
	}

	// Parse response
	aiResp := result["results"].([]interface{})[0].(map[string]interface{})["flagged"]
	if aiResp == true {
		resultIf := result["results"].([]interface{})[0].(map[string]interface{})
		resultStr := fmt.Sprintf("%#v", resultIf)
		log.Warn().Str("prompt", msg).Str("user", user).Str("results", resultStr).Msg("Flagged")
		s.MessageReactionAdd(m.ChannelID, m.Reference().MessageID, "⚠️")
		_, err := s.ChannelMessageSendReply(m.ChannelID, "modAI: This message has been flagged as inappropriate. This incident will be reported.", m.Reference())
		if err != nil {
			log.Error().Err(err).Msg("Mod: Error sending discord message")
		}
		return
	} else {
		log.Info().Str("prompt", msg).Str("user", user).Msg("Not flagged")
		s.MessageReactionAdd(m.ChannelID, m.Reference().MessageID, "✅")
		proceed = true
		return
	}
}
