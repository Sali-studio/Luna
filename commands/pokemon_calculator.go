package commands

import (
	"fmt"
	"math"

	"github.com/bwmarrin/discordgo"
)

// タイプ相性のデータを保持するマップ
var typeChart = map[string]map[string]float64{
	"ノーマル":  {"いわ": 0.5, "ゴースト": 0, "はがね": 0.5},
	"ほのお":   {"ほのお": 0.5, "みず": 0.5, "くさ": 2, "こおり": 2, "むし": 2, "いわ": 0.5, "ドラゴン": 0.5, "はがね": 2},
	"みず":    {"ほのお": 2, "みず": 0.5, "くさ": 0.5, "じめん": 2, "いわ": 2, "ドラゴン": 0.5},
	"でんき":   {"みず": 2, "でんき": 0.5, "くさ": 0.5, "じめん": 0, "ひこう": 2, "ドラゴン": 0.5},
	"くさ":    {"ほのお": 0.5, "みず": 2, "くさ": 0.5, "どく": 0.5, "じめん": 2, "ひこう": 0.5, "むし": 0.5, "いわ": 2, "ドラゴン": 0.5, "はがね": 0.5},
	"こおり":   {"ほのお": 0.5, "みず": 0.5, "くさ": 2, "こおり": 0.5, "じめん": 2, "ひこう": 2, "ドラゴン": 2, "はがね": 0.5},
	"かくとう":  {"ノーマル": 2, "こおり": 2, "どく": 0.5, "ひこう": 0.5, "エスパー": 0.5, "むし": 0.5, "いわ": 2, "ゴースト": 0, "あく": 2, "はがね": 2, "フェアリー": 0.5},
	"どく":    {"くさ": 2, "どく": 0.5, "じめん": 0.5, "いわ": 0.5, "ゴースト": 0.5, "はがね": 0, "フェアリー": 2},
	"じめん":   {"ほのお": 2, "でんき": 2, "くさ": 0.5, "どく": 2, "ひこう": 0, "むし": 0.5, "いわ": 2, "はがね": 2},
	"ひこう":   {"でんき": 0.5, "くさ": 2, "かくとう": 2, "むし": 2, "いわ": 0.5, "はがね": 0.5},
	"エスパー":  {"かくとう": 2, "どく": 2, "エスパー": 0.5, "あく": 0, "はがね": 0.5},
	"むし":    {"ほのお": 0.5, "くさ": 2, "かくとう": 0.5, "どく": 0.5, "ひこう": 0.5, "エスパー": 2, "ゴースト": 0.5, "あく": 2, "はがね": 0.5, "フェアリー": 0.5},
	"いわ":    {"ほのお": 2, "こおり": 2, "かくとう": 0.5, "じめん": 0.5, "ひこう": 2, "むし": 2, "はがね": 0.5},
	"ゴースト":  {"ノーマル": 0, "エスパー": 2, "ゴースト": 2, "あく": 0.5},
	"ドラゴン":  {"ドラゴン": 2, "はがね": 0.5, "フェアリー": 0},
	"あく":    {"かくとう": 0.5, "エスパー": 2, "ゴースト": 2, "あく": 0.5, "フェアリー": 0.5},
	"はがね":   {"ほのお": 0.5, "みず": 0.5, "でんき": 0.5, "こおり": 2, "いわ": 2, "はがね": 0.5, "フェアリー": 2},
	"フェアリー": {"かくとう": 2, "どく": 0.5, "ドラゴン": 2, "あく": 2, "はがね": 0.5},
}

type PokemonCalculatorCommand struct{}

func (c *PokemonCalculatorCommand) GetCommandDef() *discordgo.ApplicationCommand {
	float64Ptr := func(f float64) *float64 { return &f }

	typeChoices := []*discordgo.ApplicationCommandOptionChoice{}
	for typeName := range typeChart {
		typeChoices = append(typeChoices, &discordgo.ApplicationCommandOptionChoice{Name: typeName, Value: typeName})
	}

	return &discordgo.ApplicationCommand{
		Name:        "calc-pokemon",
		Description: "ポケモンの各種数値を計算します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "stats",
				Description: "ポケモンのステータス実数値を計算します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "base_stat", Description: "種族値", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "iv", Description: "個体値 (0-31)", Required: true, MinValue: float64Ptr(0), MaxValue: 31},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "ev", Description: "努力値 (0-252)", Required: true, MinValue: float64Ptr(0), MaxValue: 252},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "レベル (1-100)", Required: true, MinValue: float64Ptr(1), MaxValue: 100},
					{Type: discordgo.ApplicationCommandOptionString, Name: "stat_name", Description: "ステータス名 (HP or その他)", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "HP", Value: "hp"}, {Name: "その他(攻撃/防御/特攻/特防/素早さ)", Value: "other"}}},
					{Type: discordgo.ApplicationCommandOptionNumber, Name: "nature_correction", Description: "性格補正 (1.1, 1.0, 0.9)", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "上昇(1.1)", Value: 1.1}, {Name: "無補正(1.0)", Value: 1.0}, {Name: "下降(0.9)", Value: 0.9}}},
				},
			},
			{
				Name:        "damage",
				Description: "ダメージを計算します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "power", Description: "技の威力", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "attack_stat", Description: "攻撃側の能力実数値", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "defense_stat", Description: "防御側の能力実数値", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "攻撃側のレベル", Required: true, MinValue: float64Ptr(1), MaxValue: 100},
				},
			},
			{
				Name:        "type",
				Description: "タイプ相性を計算します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "attack_type", Description: "攻撃側の技タイプ", Required: true, Choices: typeChoices},
					{Type: discordgo.ApplicationCommandOptionString, Name: "defense_type1", Description: "防御側のタイプ1", Required: true, Choices: typeChoices},
					{Type: discordgo.ApplicationCommandOptionString, Name: "defense_type2", Description: "防御側のタイプ2 (任意)", Required: false, Choices: typeChoices},
				},
			},
		},
	}
}

func (c *PokemonCalculatorCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subCommand := i.ApplicationCommandData().Options[0]
	options := subCommand.Options
	switch subCommand.Name {
	case "stats":
		c.handleStatsCalc(s, i, options)
	case "damage":
		c.handleDamageCalc(s, i, options)
	case "type":
		c.handleTypeCalc(s, i, options)
	}
}

func (c *PokemonCalculatorCommand) handleStatsCalc(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var baseStat, iv, ev, level int64
	var statName string
	var natureCorrection float64

	for _, opt := range options {
		switch opt.Name {
		case "base_stat":
			baseStat = opt.IntValue()
		case "iv":
			iv = opt.IntValue()
		case "ev":
			ev = opt.IntValue()
		case "level":
			level = opt.IntValue()
		case "stat_name":
			statName = opt.StringValue()
		case "nature_correction":
			natureCorrection = opt.FloatValue()
		}
	}

	var result float64
	if statName == "hp" {
		result = math.Floor((float64(baseStat*2+iv)+math.Floor(float64(ev)/4))*float64(level)/100) + 10 + float64(level)
	} else {
		result = math.Floor((math.Floor((float64(baseStat*2+iv)+math.Floor(float64(ev)/4))*float64(level)/100) + 5) * natureCorrection)
	}

	embed := &discordgo.MessageEmbed{
		Title: "ポケモン ステータス実数値 計算結果",
		Color: 0xFF0000,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "入力値", Value: fmt.Sprintf("種族値: %d, 個体値: %d, 努力値: %d, Lv: %d, 性格補正: %.1f", baseStat, iv, ev, level, natureCorrection)},
			{Name: "計算結果", Value: fmt.Sprintf("実数値: **%d**", int(result))},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
}

func (c *PokemonCalculatorCommand) handleDamageCalc(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var power, attackStat, defenseStat, level int64
	for _, opt := range options {
		switch opt.Name {
		case "power":
			power = opt.IntValue()
		case "attack_stat":
			attackStat = opt.IntValue()
		case "defense_stat":
			defenseStat = opt.IntValue()
		case "level":
			level = opt.IntValue()
		}
	}

	// ダメージ計算式 (A÷B) × C × D ÷ E + 2
	damageBase := math.Floor(float64(level)*2/5 + 2)
	damageValue := math.Floor(math.Floor(damageBase*float64(power)*float64(attackStat)/float64(defenseStat))/50) + 2

	// 乱数幅 (0.85 ~ 1.00) を考慮
	minDamage := math.Floor(damageValue * 0.85)
	maxDamage := math.Floor(damageValue * 1.00)

	embed := &discordgo.MessageEmbed{
		Title: "ポケモン ダメージ計算結果",
		Color: 0xFFA500,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "入力値", Value: fmt.Sprintf("攻撃側Lv: %d, 技威力: %d, 攻撃側能力: %d, 防御側能力: %d", level, power, attackStat, defenseStat)},
			{Name: "計算結果 (乱数幅考慮)", Value: fmt.Sprintf("ダメージ: **%d 〜 %d**", int(minDamage), int(maxDamage))},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
}

func (c *PokemonCalculatorCommand) handleTypeCalc(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var attackType, defenseType1, defenseType2 string
	for _, opt := range options {
		switch opt.Name {
		case "attack_type":
			attackType = opt.StringValue()
		case "defense_type1":
			defenseType1 = opt.StringValue()
		case "defense_type2":
			defenseType2 = opt.StringValue()
		}
	}

	magnification1, ok1 := typeChart[attackType][defenseType1]
	if !ok1 {
		magnification1 = 1.0
	}

	magnification2 := 1.0
	if defenseType2 != "" {
		m2, ok2 := typeChart[attackType][defenseType2]
		if !ok2 {
			m2 = 1.0
		}
		magnification2 = m2
	}

	totalMagnification := magnification1 * magnification2

	resultText := ""
	switch totalMagnification {
	case 4:
		resultText = "こうかは ばつぐんだ！ (4倍)"
	case 2:
		resultText = "こうかは ばつぐんだ！ (2倍)"
	case 1:
		resultText = "等倍 (1倍)"
	case 0.5:
		resultText = "こうかは いまひとつの ようだ (0.5倍)"
	case 0.25:
		resultText = "こうかは いまひとつの ようだ (0.25倍)"
	case 0:
		resultText = "こうかが ない みたいだ… (0倍)"
	default:
		resultText = fmt.Sprintf("計算結果: %.2f倍", totalMagnification)
	}

	embed := &discordgo.MessageEmbed{
		Title: "ポケモン タイプ相性 計算結果",
		Color: 0x008080,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "攻撃タイプ", Value: attackType, Inline: true},
			{Name: "防御タイプ", Value: fmt.Sprintf("%s / %s", defenseType1, defenseType2), Inline: true},
			{Name: "結果", Value: fmt.Sprintf("**%s**", resultText)},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
}

func (c *PokemonCalculatorCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
}
func (c *PokemonCalculatorCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
}
func (c *PokemonCalculatorCommand) GetComponentIDs() []string { return []string{} }
