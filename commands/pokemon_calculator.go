package commands

import (
	"fmt"
	"luna/logger"
	"math"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// 計算用の共有マップ
var natureMultipliers = map[string]float64{
	"up": 1.1, "neutral": 1.0, "down": 0.9,
}
var rankMultipliers = map[int64]float64{
	6: 4.0, 5: 3.5, 4: 3.0, 3: 2.5, 2: 2.0, 1: 1.5,
	0: 1.0, -1: 0.66, -2: 0.5, -3: 0.4, -4: 0.33, -5: 0.28, -6: 0.25,
}
var typeChart = map[string]map[string]float64{
	"normal":   {"rock": 0.5, "ghost": 0, "steel": 0.5},
	"fire":     {"fire": 0.5, "water": 0.5, "grass": 2, "ice": 2, "bug": 2, "rock": 0.5, "dragon": 0.5, "steel": 2},
	"water":    {"fire": 2, "water": 0.5, "grass": 0.5, "ground": 2, "rock": 2, "dragon": 0.5},
	"electric": {"water": 2, "electric": 0.5, "grass": 0.5, "ground": 0, "flying": 2, "dragon": 0.5},
	"grass":    {"fire": 0.5, "water": 2, "electric": 1, "grass": 0.5, "poison": 0.5, "ground": 2, "flying": 0.5, "bug": 0.5, "rock": 2, "dragon": 0.5, "steel": 0.5},
	"ice":      {"fire": 0.5, "water": 0.5, "grass": 2, "ice": 0.5, "ground": 2, "flying": 2, "dragon": 2, "steel": 0.5},
	"fighting": {"normal": 2, "ice": 2, "poison": 0.5, "flying": 0.5, "psychic": 0.5, "bug": 0.5, "rock": 2, "ghost": 0, "dark": 2, "steel": 2},
	"poison":   {"grass": 2, "poison": 0.5, "ground": 0.5, "rock": 0.5, "ghost": 0.5, "steel": 0},
	"ground":   {"fire": 2, "electric": 2, "grass": 0.5, "poison": 2, "flying": 0, "bug": 0.5, "rock": 2, "steel": 2},
	"flying":   {"electric": 0.5, "grass": 2, "fighting": 2, "bug": 2, "rock": 0.5, "steel": 0.5},
	"psychic":  {"fighting": 2, "poison": 2, "psychic": 0.5, "dark": 0, "steel": 0.5},
	"bug":      {"fire": 0.5, "grass": 2, "fighting": 0.5, "poison": 0.5, "flying": 0.5, "psychic": 2, "ghost": 0.5, "dark": 2, "steel": 0.5},
	"rock":     {"fire": 2, "ice": 2, "fighting": 0.5, "ground": 0.5, "flying": 2, "bug": 2, "steel": 0.5},
	"ghost":    {"normal": 0, "psychic": 2, "ghost": 2, "dark": 0.5},
	"dragon":   {"dragon": 2, "steel": 0.5},
	"dark":     {"fighting": 0.5, "psychic": 2, "ghost": 2, "dark": 0.5},
	"steel":    {"fire": 0.5, "water": 0.5, "electric": 0.5, "ice": 2, "rock": 2, "dragon": 1, "steel": 0.5},
}
var typeNames = []string{"normal", "fire", "water", "electric", "grass", "ice", "fighting", "poison", "ground", "flying", "psychic", "bug", "rock", "ghost", "dragon", "dark", "steel"}

func init() {
	var typeChoices []*discordgo.ApplicationCommandOptionChoice
	for _, t := range typeNames {
		typeChoices = append(typeChoices, &discordgo.ApplicationCommandOptionChoice{Name: t, Value: t})
	}

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
			}, // ★★★ このカンマが重要です ★★★
			{
				Name:        "damage",
				Description: "ダメージ計算を行います。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "攻撃側ポケモンのレベル", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "power", Description: "使用する技の威力", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "attack_stat", Description: "攻撃側の こうげき / とくこう の実数値", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "defense_stat", Description: "防御側の ぼうぎょ / とくぼう の実数値", Required: true},
					{Type: discordgo.ApplicationCommandOptionNumber, Name: "multiplier", Description: "タイプ相性や天候などの補正倍率 (例: 2倍なら2.0, 半減なら0.5)", Required: false},
				},
			},
			{
				Name:        "type",
				Description: "タイプ相性を表示します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type: discordgo.ApplicationCommandOptionString, Name: "type1", Description: "ポケモンのタイプ1",
						Required: true, Choices: typeChoices,
					},
					{
						Type: discordgo.ApplicationCommandOptionString, Name: "type2", Description: "ポケモンのタイプ2 (任意)",
						Required: false, Choices: typeChoices,
					},
				},
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.ApplicationCommandData().Options[0].Name {
		case "stats":
			handleStatsCalc(s, i)
		case "damage":
			handleDamageCalc(s, i)
		case "type":
			handleTypeCalc(s, i)
		}
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// handleDamageCalc はダメージ計算の処理を行います
func handleDamageCalc(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Info.Println("calc-pokemon damage command received")

	options := i.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	level := float64(optionMap["level"].IntValue())
	power := float64(optionMap["power"].IntValue())
	attack := float64(optionMap["attack_stat"].IntValue())
	defense := float64(optionMap["defense_stat"].IntValue())
	multiplier := 1.0
	if opt, ok := optionMap["multiplier"]; ok {
		multiplier = opt.FloatValue()
	}

	// ダメージ計算式 (乱数幅を考慮し、最小と最大のダメージを計算)
	baseDamage := math.Floor(math.Floor(level*2/5+2) * power * attack / defense)
	minDamage := math.Floor(math.Floor(baseDamage/50+2) * 0.85 * multiplier)
	maxDamage := math.Floor(math.Floor(baseDamage/50+2) * 1.0 * multiplier)

	embed := &discordgo.MessageEmbed{
		Title: "⚔️ ポケモンダメージ計算結果",
		Color: 0xE74C3C, // 赤色
		Fields: []*discordgo.MessageEmbedField{
			{Name: "計算結果 (ダメージ範囲)", Value: fmt.Sprintf("▶️ **`%.0f` ~ `%.0f`**", minDamage, maxDamage)},
			{Name: "入力情報", Value: fmt.Sprintf("A(C): `%.0f` / B(D): `%.0f` / 技威力: `%.0f` / 補正: `x%.2f`", attack, defense, power, multiplier)},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

// handleTypeCalc はタイプ相性表示の処理を行います
func handleTypeCalc(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Info.Println("calc-pokemon type command received")

	options := i.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	type1 := optionMap["type1"].StringValue()
	type2 := ""
	if opt, ok := optionMap["type2"]; ok {
		type2 = opt.StringValue()
	}

	// 各タイプの攻撃を受けたときの倍率を計算
	resistances := make(map[string]float64)
	for _, t := range typeNames {
		multiplier := 1.0
		if m, ok := typeChart[t][type1]; ok {
			multiplier *= m
		}
		if type2 != "" {
			if m, ok := typeChart[t][type2]; ok {
				multiplier *= m
			}
		}
		resistances[t] = multiplier
	}

	// 倍率ごとに分類
	x4_weak := []string{}
	x2_weak := []string{}
	x0_5_resist := []string{}
	x0_25_resist := []string{}
	x0_immune := []string{}

	for t, m := range resistances {
		switch m {
		case 4:
			x4_weak = append(x4_weak, t)
		case 2:
			x2_weak = append(x2_weak, t)
		case 0.5:
			x0_5_resist = append(x0_5_resist, t)
		case 0.25:
			x0_25_resist = append(x0_25_resist, t)
		case 0:
			x0_immune = append(x0_immune, t)
		}
	}

	// 結果表示用のEmbedを作成
	title := strings.Title(type1)
	if type2 != "" {
		title += " / " + strings.Title(type2)
	}
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("タイプ相性: %s", title),
		Color: 0x95A5A6, // グレー
	}

	addFieldIfNotEmpty := func(name string, types []string) {
		if len(types) > 0 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: name, Value: strings.Join(types, ", ")})
		}
	}

	addFieldIfNotEmpty("x4 弱点", x4_weak)
	addFieldIfNotEmpty("x2 弱点", x2_weak)
	addFieldIfNotEmpty("x0.5 耐性", x0_5_resist)
	addFieldIfNotEmpty("x0.25 耐性", x0_25_resist)
	addFieldIfNotEmpty("x0 無効", x0_immune)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}
