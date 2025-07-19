package commands

import (
	"fmt"
	"luna/interfaces"
	"luna/player"

	"github.com/bwmarrin/discordgo"
)

// PlayCommand は音楽を再生するコマンドです。
type PlayCommand struct {
	Player interfaces.MusicPlayer
	Log    interfaces.Logger
}

func (c *PlayCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "play",
		Description: "指定されたURLの音楽を再生します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "url",
				Description: "再生したい音楽のURL (YouTubeなど)",
				Required:    true,
			},
		},
	}
}

func (c *PlayCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	url := options[0].StringValue()

	// ボットがボイスチャンネルに接続しているか確認
	gp := c.Player.GetGuildPlayer(i.GuildID)
	if gp == nil || gp.(*player.GuildPlayer).VoiceConnection == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ボットがボイスチャンネルに接続していません。`/join` コマンドで接続してください。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// yt-dlpからタイトルと作者を取得
	streamURL, title, author, err := gp.(*player.Player).getAudioStreamURL(url) // Playerのメソッドを呼び出す
	if err != nil {
		c.Log.Error("Failed to get song info from yt-dlp", "error", err, "url", url)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("曲情報の取得に失敗しました: %s", err.Error()),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 再生キューに追加
	err = c.Player.Play(i.GuildID, streamURL, title, author)
	if err != nil {
		c.Log.Error("Failed to play music", "error", err, "guildID", i.GuildID, "url", url)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("音楽の再生に失敗しました: %s", err.Error()),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 成功メッセージをEmbedで送信
	embed := &discordgo.MessageEmbed{
		Title:       "🎵 再生キューに追加しました！",
		Description: fmt.Sprintf("**[%s](%s)** - %s を再生キューに追加しました。", title, url, author),
		Color:       0x3498db, // Blue
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *PlayCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *PlayCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *PlayCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *PlayCommand) GetCategory() string                                                  { return "音楽" }