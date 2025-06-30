package commands

import (
	"fmt"
	"luna/logger"
	"math"

	"github.com/bwmarrin/discordgo"
)

// --- 計算用の共有マップ ---
var natureMultipliers = map[string]float64{
	"up": 1.1, "neutral": 1.0, "down": 0.9,
}
var rankMultipliers = map[int64]float64{
	6: 4.0, 5: 3.5, 4: 3.0, 3: 2.5, 2: 2.0, 1: 1.5,
	0: 1.0, -1: 0.66, -2: 0.5, -3: 0.4, -4: 0.33, -5: 0.28, -6: 0.25,
}

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "calc-pokemon",
		Description: "ポケモンの各種数値を計算します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "stats",
				Description: "ポケモンの実数値を計算します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "base_stat", Description: "計算したいステータスの種族値", Required: true},
					{
						Type: discordgo.ApplicationCommandOptionString, Name: "stat_type", Description: "ステータスの種類", Required: true,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: "HP", Value: "hp"}, {Name: "こうげき (Attack)", Value: "attack"}, {Name: "ぼうぎょ (Defense)", Value: "defense"},
							{Name: "とくこう (Sp. Atk)", Value: "sp_attack"}, {Name: "とくぼう (Sp. Def)", Value: "sp_defense"}, {Name: "すばやさ (Speed)", Value: "speed"},
						},
					},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "レベル (デフォルト: 50)", Required: false},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "iv", Description: "個体値 (0-31, デフォルト: 31)", Required: false},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "ev", Description: "努力値 (0-252, デフォルト: 0)", Required: false},
					{
						Type: discordgo.ApplicationCommandOptionString, Name: "nature", Description: "性格補正 (デフォルト: 補正なし)", Required: false,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: "⬆️ 上昇補正 (1.1倍)", Value: "up"}, {Name: "補正なし (1.0倍)", Value: "neutral"}, {Name: "⬇️ 下降補正 (0.9倍)", Value: "down"},
						},
					},
					{
						Type: discordgo.ApplicationCommandOptionInteger, Name: "rank", Description: "能力ランク (-6 ~ +6, デフォルト: 0)",
						MinValue: func(f float64) *float64 { return &f }(-6.0), MaxValue: 6, Required: false,
					},
					{
						Type: discordgo.ApplicationCommandOptionString, Name: "item", Description: "ステータスや威力に補正のある持ち物", Required: false,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: "こだわりハチマキ (Choice Band)", Value: "choice_band"}, {Name: "こだわりメガネ (Choice Specs)", Value: "choice_specs"},
							{Name: "こだわりスカーフ (Choice Scarf)", Value: "choice_scarf"}, {Name: "いのちのたま (Life Orb)", Value: "life_orb"},
							{Name: "たつじんのおび (Expert Belt)", Value: "expert_belt"}, {Name: "各種プレート (Plates)", Value: "plate"},
						},
					},
				},
			},
			{
				Name:        "damage",
				Description: "ダメージ計算を行います。(現在開発中)",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "type",
				Description: "タイプ相性を表示します。(現在開発中)",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.ApplicationCommandData().Options[0].Name {
		case "stats":
			handleStatsCalc(s, i)
		case "damage":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "🛠️ この機能は現在開発中です。", Flags: discordgo.MessageFlagsEphemeral},
			})
		case "type":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "🛠️ この機能は現在開発中です。", Flags: discordgo.MessageFlagsEphemeral},
			})
		}
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// handleStatsCalc は実数値計算の処理を行います
func handleStatsCalc(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Info.Println("calc-pokemon stats command received")

	options := i.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	baseStat := float64(optionMap["base_stat"].IntValue())
	statType := optionMap["stat_type"].StringValue()
	level := float64(50)
	if opt, ok := optionMap["level"]; ok {
		level = float64(opt.IntValue())
	}
	iv := float64(31)
	if opt, ok := optionMap["iv"]; ok {
		iv = float64(opt.IntValue())
	}
	ev := float64(0)
	if opt, ok := optionMap["ev"]; ok {
		ev = float64(opt.IntValue())
	}
	natureKey := "neutral"
	if opt, ok := optionMap["nature"]; ok {
		natureKey = opt.StringValue()
	}
	rank := int64(0)
	if opt, ok := optionMap["rank"]; ok {
		rank = opt.IntValue()
	}
	item := ""
	if opt, ok := optionMap["item"]; ok {
		item = opt.StringValue()
	}

	var result float64
	if statType == "hp" {
		result = math.Floor((baseStat*2+iv+math.Floor(ev/4))*level/100) + level + 10
	} else {
		base := math.Floor((baseStat*2+iv+math.Floor(ev/4))*level/100) + 5
		natureMultiplier := natureMultipliers[natureKey]
		result = math.Floor(base * natureMultiplier)
		if rank != 0 {
			result = math.Floor(result * rankMultipliers[rank])
		}
		itemMultiplier := 1.0
		switch item {
		case "choice_band":
			if statType == "attack" {
				itemMultiplier = 1.5
			}
		case "choice_specs":
			if statType == "sp_attack" {
				itemMultiplier = 1.5
			}
		case "choice_scarf":
			if statType == "speed" {
				itemMultiplier = 1.5
			}
		}
		result = math.Floor(result * itemMultiplier)
	}

	itemNote := ""
	switch item {
	case "life_orb":
		itemNote = "⚠️ いのちのたま: 技のダメージが1.3倍になります。"
	case "expert_belt":
		itemNote = "⚠️ たつじんのおび: 効果ばつぐんの技のダメージが1.2倍になります。"
	case "plate":
		itemNote = "⚠️ プレート: 一致するタイプの技のダメージが1.2倍になります。"
	}

	embed := &discordgo.MessageEmbed{
		Title: "📊 ポケモン実数値計算結果",
		Color: 0x3498DB,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "入力情報",
				Value: fmt.Sprintf("Lv`%.0f` / 種族値`%.0f` / 個体値`%.0f` / 努力値`%.0f`\n性格補正: `%s` / ランク: `%+d` / 持ち物: `%s`",
					level, baseStat, iv, ev, natureKey, rank, item),
			},
			{
				Name:  "計算結果",
				Value: fmt.Sprintf("▶️ **実数値: `%.0f`**", result),
			},
		},
	}

	if itemNote != "" {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: itemNote}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
