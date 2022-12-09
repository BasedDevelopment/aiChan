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

func chat(s *discordgo.Session, m *discordgo.MessageCreate, msg string) {
	// Token
	token := k.String("ai.chat.token")
	var bearer = "Bearer " + token

	// send API req
	type req struct {
		Model             string  `json:"model"`
		Prompt            string  `json:"prompt"`
		Temperature       float64 `json:"temperature"`
		Max_tokens        int     `json:"max_tokens"`
		Top_p             float64 `json:"top_p"`
		Frequency_penalty float64 `json:"frequency_penalty"`
		Presence_penalty  float64 `json:"presence_penalty"`
		user              string  `json:"user"`
	}
	request := req{
		Model:             "text-davinci-003",
		Prompt:            msg,
		Temperature:       0.9,
		Max_tokens:        350,
		Top_p:             1,
		Frequency_penalty: 0,
		Presence_penalty:  0.6,
		user:              m.Author.ID,
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		fmt.Println(err)
		s.ChannelMessageSendReply(m.ChannelID, "Chat: http err", m.MessageReference)
		return
	}
	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println(err)
		s.ChannelMessageSendReply(m.ChannelID, "Error", m.MessageReference)
		return
	}
	httpReq.Header.Set("Authorization", bearer)
	httpReq.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	defer resp.Body.Close()
	if err != nil {
		log.Error().Err(err).Msg("Error sending request")
		s.ChannelMessageSendReply(m.ChannelID, "Error http", m.Reference())
		return
	}

	// read response
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	user := m.Author.Username + "#" + m.Author.Discriminator

	// Handle response
	if result == nil {
		respRead, _ := io.ReadAll(resp.Body)
		respStr := string(respRead)
		log.Warn().Str("user", user).Str("resp", respStr).Str("prompt", msg).Msg("Chat: result is nil")
		s.ChannelMessageSendReply(m.ChannelID, "Chat: results is nil", m.Reference())
		return
	}
	if result["choices"] == nil {
		resultStr := fmt.Sprintf("%#v", result)
		log.Warn().Str("user", user).Str("prompt", msg).Str("resp", resultStr).Msg("Chat: choices is nil")
		s.ChannelMessageSendReply(m.ChannelID, "Chat: choices is nil", m.Reference())
		return
	}

	// Send response
	aiRespStr := result["choices"].([]interface{})[0].(map[string]interface{})["text"].(string)
	log.Info().Str("user", user).Str("prompt", msg).Str("resp", aiRespStr).Msg("Chat: Success")
	s.ChannelMessageSendReply(m.ChannelID, aiRespStr, m.Reference())
}
