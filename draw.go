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
	"io"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func draw(s *discordgo.Session, m *discordgo.MessageCreate, msg string) {
	// Token
	token := k.String("ai.draw.token")
	var bearer = "Bearer " + token

	// send API req
	type req struct {
		Prompt string `json:"prompt"`
		n      int    `json:"n"`
		size   string `json:"size"`
		user   string
	}
	request := req{
		Prompt: msg,
		n:      1,
		size:   "1024x1024",
		user:   m.Author.ID,
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal request")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Error", m.MessageReference); err != nil {
			log.Error().Err(err).Msg("Draw: Error sending discord message")
		}
		return
	}
	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/images/generations", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Error", m.MessageReference); err != nil {
			log.Error().Err(err).Msg("Draw: Error sending discord message")
		}
		return
	}
	httpReq.Header.Set("Authorization", bearer)
	httpReq.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	defer resp.Body.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to send request")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Error http", m.Reference()); err != nil {
			log.Error().Err(err).Msg("Draw: Error sending discord message")
		}
		return
	}

	// Decode response
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	user := m.Author.Username + "#" + m.Author.Discriminator

	// Handle respons
	if result == nil {
		respRead, _ := io.ReadAll(resp.Body)
		respStr := string(respRead)
		log.Error().Str("user", user).Str("resp", respStr).Str("prompt", msg).Msg("Draw: result is nil")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Draw: result is nil", m.Reference()); err != nil {
			log.Error().Err(err).Msg("Draw: Error sending discord message")
		}
		return
	}
	if result["data"] == nil {
		if result["error"] != nil {
			errMsg := result["error"].(map[string]interface{})["message"].(string)
			log.Warn().Str("user", user).Str("prompt", msg).Str("error", errMsg).Msg("Draw: error")
			if _, err := s.ChannelMessageSendReply(m.ChannelID, "Draw: "+errMsg, m.Reference()); err != nil {
				log.Error().Err(err).Msg("Draw: Error sending discord message")
			}
			return
		}
		resultStr := fmt.Sprintf("%#v", result)
		log.Warn().Str("user", user).Str("prompt", msg).Str("resp", resultStr).Msg("Draw: data is nil")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Draw: data is nil (likely OpenAI rejecting request due to inappropriate prompt)", m.Reference()); err != nil {
			log.Error().Err(err).Msg("Draw: Error sending discord message")
		}
		return
	}

	// Send
	aiRespStr := result["data"].([]interface{})[0].(map[string]interface{})["url"].(string)
	log.Info().Str("user", user).Str("prompt", msg).Str("resp", aiRespStr).Msg("Draw: success")
	if _, err := s.ChannelMessageSendReply(m.ChannelID, aiRespStr, m.Reference()); err != nil {
		log.Error().Err(err).Msg("Draw: Error sending discord message")
	}
}
