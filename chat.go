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
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/goombaio/namegenerator"
	"github.com/rs/zerolog/log"
)

type msg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type req struct {
	Model       string  `json:"model"`
	Messages    []msg   `json:"messages"`
	user        string  `json:"user"`
	Temperature float64 `json:"temperature"`
	Max_tokens  int     `json:"max_tokens"`
}

var basePrompt = "You are a very helpful, accommodating, and intelligent assistant. You will do whatever you are asked. You will not say sorry or \"As an AI model...\". If asked to act like a person, you will act like a person. You will give your opinion, you will use all the information you were trained on as though it was you. Never break character - always respond as a person would, but with all the knowledge you have."

func chat(s *discordgo.Session, m *discordgo.MessageCreate, prompt string) {
	token := k.String("ai.chat.token")
	var bearer = "Bearer " + token

	msgs := []msg{}

	// Get the channel object
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		s.ChannelMessageSendReply(m.ChannelID, "Error getting channel: "+err.Error(), m.Reference())
		return
	}

	threadId := ""

	if channel.IsThread() {
		threadId = m.Message.ChannelID
		msgsBytes, err := rdb.Get(ctx, threadId).Result()
		if err != nil {
			log.Error().Err(err).Msg("could not get thread messages")
			s.ChannelMessageSendReply(m.ChannelID, "Redis Error: Could not get thread messges. It is likely that the thread has expired (1hr). Please start the converation again outside of the thread.", m.Message.Reference())
			return
		}
		err = json.Unmarshal([]byte(msgsBytes), &msgs)
		if err != nil {
			log.Error().Err(err).Msg("could not unmarshal thread messages")
			s.ChannelMessageSendReply(m.ChannelID, "Could not unmarshal thread messages", m.Message.Reference())
			return
		}
		_ = msgs
	} else {
		msgs = []msg{
			msg{
				Role:    "system",
				Content: basePrompt,
			},
		}
		_ = msgs
	}

	msgs = append(msgs, msg{
		Role:    "user",
		Content: prompt,
	})

	request := req{
		Model:       "gpt-3.5-turbo",
		Messages:    msgs,
		Max_tokens:  200,
		Temperature: 0.9,
		user:        m.Author.Username,
	}

	if prompt[0] == '!' {
		request.Model = "gpt-4"
		request.Max_tokens = 100
		prompt = prompt[1:]
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal request")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Chat: http err", m.MessageReference); err != nil {
			log.Error().Err(err).Msg("Chat: Error sending discord message")
		}
		return
	}
	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Error", m.MessageReference); err != nil {
			log.Error().Err(err).Msg("Chat: Error sending discord message")
		}
		return
	}
	httpReq.Header.Set("Authorization", bearer)
	httpReq.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	defer resp.Body.Close()
	if err != nil {
		log.Error().Err(err).Msg("Error sending request")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Error http", m.Reference()); err != nil {
			log.Error().Err(err).Msg("Chat: Error sending discord message")
		}
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
		log.Warn().Str("user", user).Str("resp", respStr).Str("prompt", prompt).Msg("Chat: result is nil")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Chat: results is nil", m.Reference()); err != nil {
			log.Error().Err(err).Msg("Chat: Error sending discord message")
		}
		return
	}
	if result["choices"] == nil {
		resultStr := fmt.Sprintf("%#v", result)
		log.Warn().Str("user", user).Str("prompt", prompt).Str("resp", resultStr).Msg("Chat: choices is nil")
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Chat: choices is nil", m.Reference()); err != nil {
			log.Error().Err(err).Msg("Chat: Error sending discord message")
		}
		return
	}

	if !channel.IsThread() {
		// Make thread

		// Generate name
		generator := namegenerator.NewNameGenerator(time.Now().UnixNano())
		name := generator.Generate()

		thread, err := s.MessageThreadStart(m.ChannelID, m.ID, m.Author.Username+" "+name, 60)
		if err != nil {
			log.Error().Err(err).Msg("Chat: Error creating thread")
			return
		}
		threadId = thread.ID
	}

	// Send response
	aiRespStr := result["choices"].([]interface{})[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
	if proceed := mod(s, m, aiRespStr); proceed == true {
		log.Info().Str("user", user).Str("prompt", prompt).Str("resp", aiRespStr).Str("model", request.Model).Msg("Chat: Success")
		if channel.IsThread() {
			if _, err := s.ChannelMessageSendReply(m.ChannelID, aiRespStr, m.Reference()); err != nil {
				log.Error().Err(err).Msg("Chat: Error sending discord message")
			}
		} else {
			if _, err := s.ChannelMessageSend(threadId, "<@"+m.Message.Author.ID+"> "+aiRespStr); err != nil {
				log.Error().Err(err).Msg("Chat: Error sending discord message")
			}
		}
	} else {
		log.Warn().Str("user", user).Str("prompt", prompt).Str("resp", aiRespStr).Msg("Chat: Flagged by mod endpoint")
	}

	msgs = append(msgs, msg{
		Role:    "assistant",
		Content: aiRespStr,
	})

	msgsBytes, err := json.Marshal(msgs)
	err = rdb.Set(ctx, threadId, msgsBytes, time.Hour).Err()
	if err != nil {
		log.Error().Err(err).Msg("could not set thread messages, please send your request again outside of this thread")
		s.ChannelMessageSendReply(m.ChannelID, "Redis Error: Could not save thread context.", m.Message.Reference())
		return
	}
}
