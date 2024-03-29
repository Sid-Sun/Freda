package main

import (
	"database/sql"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Repeat struct {
	NumberOfRepeats int
	RepeatNext      bool
}

type MessageDetails struct {
	ParseMode        string
	Message          string
	ReplyToMessageID int
	ChatID           int64
}

var ignoreNextMessage = false

var Repeats = make(map[string]Repeat)

func main() {
	var wg sync.WaitGroup
	bot, err := tgbotapi.NewBotAPI(os.Getenv("FREDA_API_TOKEN_ID"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	fmt.Printf("Hello, I am %s\n", bot.Self.FirstName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go getUpdates(bot, updates)
	}
	wg.Wait()
}

func getUpdates(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message == nil {
			continue
		}
		go handleUpdate(bot, update)
	}
}

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	adminChatID, _ := strconv.ParseInt(os.Getenv("ADMIN_CHAT_ID"), 10, 64)
	if update.Message.From.ID == 777000 && update.Message.Chat.UserName == os.Getenv("TARGET_CHAT_USERNAME") { //777000 is the ID of Telegram's replicating service for channel to discussion group.
		if !ignoreNextMessage {
			successful, addError := addToDatabase(update.Message.Text)
			if successful {
				go sendMessage(bot, MessageDetails{
					Message:          "My Lord, I have added message successfully to database, I hope I am serving you well.",
					ReplyToMessageID: update.Message.MessageID,
					ChatID:           adminChatID,
					ParseMode:        "",
				})
			} else { // Get a message is something goes south
				go sendMessage(bot, MessageDetails{
					Message:          "Something failed, sending details. If you don't get the details in a message immediately after this one, It might be something very bad.",
					ReplyToMessageID: 0,
					ChatID:           adminChatID,
					ParseMode:        "",
				}) // addError.Error MAY lead to nil pointer derefernce which will cause a panic, I am not sure if that will ever happen in out case
				go sendMessage(bot, MessageDetails{
					Message:          "My Lord, I have failed in adding the message database, the error I encountered is: " + addError.Error() + " I am sorry to have disappointed you.",
					ReplyToMessageID: 0,
					ChatID:           adminChatID,
					ParseMode:        "",
				})
			}
		} else {
			ignoreNextMessage = false
		}
	} else {
		if update.Message.Chat.IsPrivate() {
			messageSlice := strings.Fields(update.Message.Text)
			if update.Message.Chat.ID == adminChatID && len(messageSlice) == 1 && messageSlice[0] == "/toggleIgnore" {
				ignoreNextMessage = !ignoreNextMessage
				go sendMessage(bot, MessageDetails{
					Message:          "Toggled Ignore.",
					ReplyToMessageID: 0,
					ChatID:           adminChatID,
					ParseMode:        "",
				})
			} else if len(messageSlice) == 2 && messageSlice[0] == "/repeat" {
				var tempRepeat Repeat
				var err error
				tempRepeat.NumberOfRepeats, err = strconv.Atoi(messageSlice[1])
				tempRepeat.RepeatNext = true
				if err != nil {
					go sendMessage(bot, MessageDetails{
						Message:          "Oopsie Whoopsie, could not parse your numbersie",
						ReplyToMessageID: update.Message.MessageID,
						ChatID:           update.Message.Chat.ID,
						ParseMode:        "",
					})
				}
				ChatID := strconv.FormatInt(update.Message.Chat.ID, 10)
				Repeats[ChatID] = tempRepeat
			} else {
				numberOfRepeats := 1
				replyToMessageID := update.Message.MessageID
				ChatID := strconv.FormatInt(update.Message.Chat.ID, 10)
				if Repeats[ChatID].RepeatNext {
					numberOfRepeats = Repeats[ChatID].NumberOfRepeats
					replyToMessageID = 0
					var tempRepeat Repeat
					tempRepeat.RepeatNext = false
					tempRepeat.NumberOfRepeats = 1
					Repeats[ChatID] = tempRepeat
				}
				for i := 0; i < numberOfRepeats; i++ {
					go sendMessage(bot, MessageDetails{
						Message:          update.Message.Text,
						ReplyToMessageID: replyToMessageID,
						ChatID:           update.Message.Chat.ID,
						ParseMode:        "",
					})
				}
			}
		} else if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "report":
				var replyToMessage int
				var reply string
				if update.Message.ReplyToMessage == nil {
					replyToMessage = update.Message.MessageID
					reply = "Please reply to the message you wish to report with this command."
				} else {
					replyToMessage = update.Message.ReplyToMessage.MessageID
					admins, err := bot.GetChatAdministrators(tgbotapi.ChatConfig{ChatID: update.Message.Chat.ID})
					if err != nil {
						go sendMessage(bot, MessageDetails{
							Message:          "My Lord, there was an error while reporting: " + err.Error() + " I am sorry to have disappointed you.",
							ReplyToMessageID: 0,
							ChatID:           adminChatID,
							ParseMode:        "",
						})
						return
					}
					for _, admin := range admins {
						reply = reply + "[" + admin.User.FirstName + "](tg://user?id=" + strconv.Itoa(admin.User.ID) + ") "
					}
				}
				go sendMessage(bot, MessageDetails{
					Message:          reply,
					ReplyToMessageID: replyToMessage,
					ChatID:           update.Message.Chat.ID,
					ParseMode:        "markdown",
				})
			}
		}
	}
}

func sendMessage(bot *tgbotapi.BotAPI, details MessageDetails) {
	msg := tgbotapi.NewMessage(details.ChatID, details.Message)
	if details.ReplyToMessageID != 0 {
		msg.ReplyToMessageID = details.ReplyToMessageID
	}
	msg.ParseMode = details.ParseMode
	_, _ = bot.Send(msg)
}

func addToDatabase(message string) (bool, error) {
	var (
		host     = "localhost"
		port     = 5432
		user     = "postgres" // Replace with required user
		password = os.Getenv("POSTGRES_PASSWORD")
		dbname   = os.Getenv("FREDA_DB_NAME")
	)
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return false, err
	}
	defer db.Close()
	sqlStatement := `
INSERT INTO channel_messages (message)
VALUES ($1)
RETURNING index`
	index := 0
	err = db.QueryRow(sqlStatement, message).Scan(&index)
	if err != nil {
		return false, err
	}
	return true, nil

}
