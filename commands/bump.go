package commands

import (
	"fmt"
	"luna/logger"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// 共有変数
var (
	bumpChannelID = make(map[string]string)
	bumpRoleID    = make(map[string]string)
)

const (
	disboardBumpCommandID = "947088344167366698"
	dissokuUpCommandID    = "977373245519372299"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "bump-config",
		Description:              "BUMP/UPリマインダーの通知チャンネルとロールを設定します。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "通知を送信するチャンネル",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				Required:     true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "role",
				Description: "通知の際にメンションするロール",
				Required:    true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options
		channel := options[0].ChannelValue(s)
		role := options[1].RoleValue(s, i.GuildID)

		bumpChannelID[i.GuildID] = channel.ID
		bumpRoleID[i.GuildID] = role.ID

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ BUMP/UPリマインダーの通知先を <#%s> に、メンションロールを %s に設定しました。", channel.ID, role.Mention()),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// HandleMessageCreate はメッセージが作成されたときに呼び出されます
func HandleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	channelID, ok := bumpChannelID[m.GuildID]
	if !ok {
		return
	}
	roleID := bumpRoleID[m.GuildID]

	var duration time.Duration
	var serviceName string
	var commandMention string

	// DisboardのBot IDとメッセージ内容で判断
	if m.Author.ID == "302050872383242240" && strings.Contains(m.Content, "表示順をアップしたよ") {
		duration = 2 * time.Hour
		serviceName = "Disboard"
		commandMention = fmt.Sprintf("</bump:%s>", disboardBumpCommandID)
	} else if m.Author.ID == "605364421593235466" && strings.Contains(m.Content, "Up完了") {
		// DissokuのBot IDとメッセージ内容で判断
		duration = 1 * time.Hour
		serviceName = "Dissoku"
		commandMention = fmt.Sprintf("</up:%s>", dissokuUpCommandID)
	} else {
		return
	}

	logger.Info.Printf("Detected %s bump. Setting a reminder for %v.", serviceName, duration)

	go func() {
		time.Sleep(duration)

		mentionStr := fmt.Sprintf("<@&%s>", roleID)

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("🔔 %s の時間です！", serviceName),
			Description: fmt.Sprintf("下のコマンドをクリックして、サーバーを宣伝しましょう！\n▶️ %s", commandMention),
			Color:       0x57F287,
		}

		s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: mentionStr,
			Embeds:  []*discordgo.MessageEmbed{embed},
		})
	}()
}
