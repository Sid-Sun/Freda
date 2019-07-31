package main

import (
	"database/sql"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq"
	"log"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	bot, err := tgbotapi.NewBotAPI("XXXXXXXXXXXXXXXXXXXXXX")
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
		if update.Message.From.ID == MESSAGEID  && update.Message.Chat.UserName == CHAT_USERNAME { //777000 is the ID of Telegram's replicating service for channel to discussion group.
		 //REPLEACE CHAT_USERNAME above with the discussion Chat's public ID (or equivalent) to make sure it only replcates message from that group
			fmt.Printf("message: %s\n", update.Message.Text)
			successful, addError := addToDatabase(update.Message.Text)
			if successful {
				msg := tgbotapi.NewMessage(YOUR_CHAT_ID, "") //Replace YOUR_CHAT_ID to get a message if bot is successful in adding it to the DB
				msg.Text = "My Lord, I have added message successfully to database, I hope I am serving you well."
				_, _ = bot.Send(msg)
			} else { // Get a message is something goes south
				msg := tgbotapi.NewMessage(YOUR_CHAT_ID, "Something failed, sending details. If you don't get the details in a message immediately after this one, It might be something very bad.")
				_, _ = bot.Send(msg) // addError.Error MAY lead to nil pointer derefernce which will cause a panic, I am not sure if that will ever happen in out case.
				msg = tgbotapi.NewMessage(YOUR_CHAT_ID, "")
				msg.Text = "My Lord, I have failed in adding the message database, the error I encountered is:" + addError.Error() + "I am sorry to have disappointed you."
				_, _ = bot.Send(msg)
				// Do replace YOUR_CHAT_ID with your chat ID (or a group or anything, if you want that)
			}
		} else {
			if update.Message.Chat.IsPrivate() {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text) // Just send back what they say, just for fun
				msg.ReplyToMessageID = update.Message.MessageID
				_, _ = bot.Send(msg)
			}
		}
}

func addToDatabase(message string) (bool, error) {
	const (
		host     = "localhost"
		port     = 5432
		user     = "GOOD_ADMIN" // Replace with required user
		password = "MOST_SECURE_PASSWORD" // Replace with required passoword (if you use this pass, i'll kill you)
		dbname   = "VERT_EFFICIENT_DB" // Replace with required DB name
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
