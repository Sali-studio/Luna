package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type weatherResponse struct {
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Name string `json:"name"`
}

type WeatherCommand struct {
	APIKey string
}

func (c *WeatherCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "weather",
		Description: "指定した都市の現在の天気を表示します",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "city", Description: "都市名 (例: Tokyo)", Required: true},
		},
	}
}

func (c *WeatherCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if c.APIKey == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ 天気機能は現在利用できません。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	city := i.ApplicationCommandData().Options[0].StringValue()
	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric&lang=ja", city, c.APIKey)

	resp, err := http.Get(url)
	if err != nil { /* ... エラー処理 ... */
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK { /* ... 都市が見つからない場合などのエラーレスポンス ... */
		return
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var data weatherResponse
	json.Unmarshal(body, &data)

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s の天気", data.Name),
		Color: 0x42b0f4,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "天気", Value: data.Weather[0].Description, Inline: true},
			{Name: "気温", Value: fmt.Sprintf("%.1f °C", data.Main.Temp), Inline: true},
			{Name: "湿度", Value: fmt.Sprintf("%d %%", data.Main.Humidity), Inline: true},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: fmt.Sprintf("http://openweathermap.org/img/wn/%s@2x.png", data.Weather[0].Icon)},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func (c *WeatherCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *WeatherCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *WeatherCommand) GetComponentIDs() []string                                            { return []string{} }
