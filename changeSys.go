package main

import "github.com/bwmarrin/discordgo"

func changeSys(s *discordgo.Session, m *discordgo.MessageCreate, prompt string) {
	basePrompt = prompt
	s.ChannelMessageSendReply(m.ChannelID, "System changed to "+prompt, m.Reference())
	return
}
