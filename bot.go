package albumbot

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var callCommand string

// New()の中で上書きされる可能性がある

var table = "Albums"

var currentBot *albumBot = &albumBot{}

type albumBot struct {
	channelID     string
	selectedAlbum string
	urls          []string
	pageindex     int
}

func newBot(channelID string) *albumBot {
	return &albumBot{channelID: channelID}
}

//指定されたアルバムの画像URLを取得する
func (bot *albumBot) loadAlbum(albumname string) (err error) {
	bot.selectedAlbum = albumname
	bot.urls, err = GetAlbumUrls(table, albumname)
	if err != nil {
		return err
	}
	return nil
}

// 1ページに表示する画像の数
const imageNumber = 5

// 現在のページが画像を何枚目から何枚目まで表示するかを返す。1から始まる
func (bot *albumBot) imageOffset() (start int, end int) {
	start = bot.pageindex*imageNumber + 1
	end = start + imageNumber - 1
	if end > len(bot.urls) {
		end = len(bot.urls)
	}
	return
}

// 現在のページの画像を返す。
func (bot *albumBot) pageImages() string {
	start, end := bot.imageOffset()
	var s string
	for i := start; i <= end; i++ {
		s += bot.urls[i-1] + "\n"
	}
	return s
}

// 現在のページの画像をDiscordに送信する
func (bot *albumBot) sendPage(s *discordgo.Session) (messageID string) {
	_, err := s.ChannelMessageSend(bot.channelID, bot.pageImages())
	if err != nil {
		s.ChannelMessageSend(bot.channelID, "Error: "+err.Error())
	}
	start, end := currentBot.imageOffset()
	sent, err := s.ChannelMessageSend(bot.channelID, fmt.Sprint(start, "枚目~", end, "枚目"))
	if err != nil {
		s.ChannelMessageSend(bot.channelID, "Error: "+err.Error())
	}
	return sent.ID
}

func (bot *albumBot) hasNextPage() bool {
	maxPage := len(bot.urls) / imageNumber
	if len(bot.urls)%imageNumber == 0 {
		maxPage--
	}
	return bot.pageindex < maxPage
}

func (bot *albumBot) hasPrevPage() bool {
	return bot.pageindex > 0
}

func (bot *albumBot) goToNextPage(s *discordgo.Session) (messageID string) {
	if bot.pageindex == len(bot.urls)/imageNumber {
		sent, err := s.ChannelMessageSend(bot.channelID, "次のページはありません")
		if err != nil {
			s.ChannelMessageSend(bot.channelID, "Error: "+err.Error())
		}
		return sent.ID
	}
	bot.pageindex++
	return bot.sendPage(s)
}

func (bot *albumBot) goToPrevPage(s *discordgo.Session) (messageID string) {
	if bot.pageindex == 0 {
		sent, err := s.ChannelMessageSend(bot.channelID, "前のページはありません")
		if err != nil {
			s.ChannelMessageSend(bot.channelID, "Error: "+err.Error())
		}
		return sent.ID
	}
	bot.pageindex--
	return bot.sendPage(s)
}

func New() {
	table = os.Getenv("TABLE_NAME")

	discordToken := "Bot " + os.Getenv("DISCORD_TOKEN")
	var ok bool
	callCommand, ok = os.LookupEnv("CALL_COMMAND")
	if !ok {
		callCommand = "!album"
	}
	session, err := discordgo.New()
	if err != nil {
		fmt.Println("Error in create session")
		panic(err)
	}
	session.Token = discordToken
	session.AddHandler(onMessageCreate)
	session.AddHandler(onReactionAdd)

	if err = session.Open(); err != nil {
		panic(err)
	}
	defer session.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGTERM)

	fmt.Println("booted!!!")

	<-sc
}

func contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func isUrlImage(url string) bool {
	exts := []string{"png", "jpg", "jpeg", "gif"}
	parts := strings.Split(url, ".")
	return contains(exts, parts[len(parts)-1])
}

//getter関数を定義
func getNumOptions() []string {
	arr := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	return arr
}

//数字から数字スタンプ文字列を返す
func getNumEmoji(i int) string {
	if i < 1 {
		return "❓"
	}
	// 対応する絵文字がない場合はその値をそのまま返す
	if i > 10 {
		return strconv.Itoa(i)
	}
	arr := getNumOptions()
	return arr[i-1]
}

//数字スタンプ文字列から数値とbool値を返す
func getNumFromNumEmoji(s string) (int, bool) {
	arr := getNumOptions()
	for i := range arr {
		if s == arr[i] {
			return i, true
		}
	}
	return 0, false
}

func albumAdd(s *discordgo.Session, m *discordgo.MessageCreate) error {
	contents := strings.Split(m.Content, " ")
	if len(contents) != 3 {
		return fmt.Errorf("→ " + callCommand + " add actual_albumname の形でファイルをアップロードしてね！")
	}
	title := contents[2]
	titles, err := GetAlbumTitles(table)
	if err != nil {
		return err
	}
	if !contains(titles, title) {
		return fmt.Errorf("%sというアルバムはなかったよ。"+callCommand+" createコマンドで作れるよ！", title)
	}
	if len(m.Attachments) == 0 {
		return fmt.Errorf("画像が一枚も添付されてないよ。")
	}
	invalidAttaches := []string{}
	for _, attach := range m.Attachments {
		if isUrlImage(attach.URL) {
			err := PostImage(table, title, attach.URL)
			if err != nil {
				return err
			}
			s.ChannelMessageSend(m.ChannelID, attach.URL+" を"+title+"アルバムに追加したよ！")
		} else {
			invalidAttaches = append(invalidAttaches, attach.Filename)
		}
	}

	if len(invalidAttaches) > 0 {
		return fmt.Errorf("以下のファイルは画像じゃないから無視したよ：\n%s", strings.Join(invalidAttaches, "\n"))
	}
	return nil
}

func checkclhelp() string {
	return callCommand + "\n・登録されているアルバムから見たいアルバムを選択する\n" +
		callCommand + " create albumtitle\n・アルバムを作成する\n" +
		callCommand + " add actual_albumname\n・アルバムに写真を追加する（以下のコマンドと同時に写真を添付）\n"

}

func commandSplit(str string) []string {
	commandArray := strings.Split(str, " ")
	return commandArray
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	command := commandSplit(m.Content)
	if len(command) == 0 || command[0] != callCommand {
		return
	}

	if len(command) == 1 {
		currentBot = newBot(m.ChannelID)

		titles, err := GetAlbumTitles(table)
		tmpstr := ""
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
		}

		if len(titles) > 10 {
			titles = titles[:10]
		}
		for i, v := range titles {
			tmpstr += getNumEmoji(i+1) + " " + v + "\n"
		}
		s.ChannelMessageSend(m.ChannelID, tmpstr)
		sent, err := s.ChannelMessageSend(m.ChannelID, "番号を選んでね！")
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
		}
		for i := range titles {
			s.MessageReactionAdd(m.ChannelID, sent.ID, getNumEmoji(i+1))
		}
		return
	}

	subCommand := command[1]
	switch subCommand {
	case "-h", "--help", "help":
		s.ChannelMessageSend(m.ChannelID, checkclhelp())
	case "create":
		if len(command) == 3 {
			err := CreateAlbum(table, command[2])
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, err.Error())
			}
			s.ChannelMessageSend(m.ChannelID, command[2]+"というアルバムを作成したよ！")
		} else {
			s.ChannelMessageSend(m.ChannelID, "→ "+callCommand+" create titlename の形で記入してね！")
		}
	case "add":
		err := albumAdd(s, m)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
		}
	}
}
func onReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	titles, err := GetAlbumTitles(table)
	if err != nil {
		s.ChannelMessageSend(r.ChannelID, err.Error())
	}
	message, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		s.ChannelMessageSend(r.ChannelID, err.Error())
	}
	botID := s.State.User.ID
	if r.UserID == botID {
		// bot自身がリアクションを押した場合何もしない
		return
	}
	if message.Author.ID != botID {
		// bot以外のメッセージにリアクションが押された場合何もしない
		return
	}
	if message.Content == "番号を選んでね！" {
		index, NumEmojiFlag := getNumFromNumEmoji(r.MessageReaction.Emoji.Name)
		if NumEmojiFlag {
			s.ChannelMessageDelete(r.ChannelID, r.MessageID)
			err := currentBot.loadAlbum(titles[index])
			if err != nil {
				s.ChannelMessageSend(r.ChannelID, err.Error())
				return
			}
			messageID := currentBot.sendPage(s)
			if currentBot.hasNextPage() {
				s.MessageReactionAdd(r.ChannelID, messageID, "➡️")
			}
		}
		// ユーザーが押した絵文字によって次か前のページに移動する
	} else {
		userReaction := r.MessageReaction.Emoji.Name
		if userReaction == "➡️" {
			s.ChannelMessageDelete(r.ChannelID, r.MessageID)
			id := currentBot.goToNextPage(s)
			if currentBot.hasNextPage() {
				s.MessageReactionAdd(r.ChannelID, id, "⬅")
				s.MessageReactionAdd(r.ChannelID, id, "➡️")
			} else {
				s.MessageReactionAdd(r.ChannelID, id, "⬅")
			}
		} else if userReaction == "⬅" {
			id := currentBot.goToPrevPage(s)
			if currentBot.hasPrevPage() {
				s.MessageReactionAdd(r.ChannelID, id, "⬅")
				s.MessageReactionAdd(r.ChannelID, id, "➡️")
			} else {
				s.MessageReactionAdd(r.ChannelID, id, "➡️")
			}
		}
	}
}
