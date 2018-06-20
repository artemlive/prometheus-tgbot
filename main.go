package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"fmt"
	"time"
	"log"
	"github.com/gin-gonic/gin"
	"html/template"

	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"regexp"
)
var (
	cfg = Config{}
	bot *tgbotapi.BotAPI
	tmpH *template.Template
	globalChan = make(chan *tgbotapi.Update)
	UserStates = []UserState{}
)

func handleTextMessage(update *tgbotapi.Update){
	if update.Message == nil {
		return
	}
	state := findUserState(update.Message.Chat.ID)
	m := state.CurMenu
	m.RunAction(state.Action, &ActionParams{bot: bot, update: update})
}

func telegramBot(bot *tgbotapi.BotAPI) {
	go userStateChanged()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	introduce := func(update tgbotapi.Update) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Hi, your chat id is '%d'", update.Message.Chat.ID))
		bot.Send(msg)
	}

	time.Sleep(time.Millisecond * 500)
	for len(updates) != 0 {
		<-updates
	}

	for update := range updates {
		log.Printf("update tick")
		if update.Message != nil && update.Message.IsCommand() {
			if checkAuthorized(update.Message.Chat.ID) == false {
				continue
			}
		} else if update.InlineQuery != nil {
			log.Printf("inline query")
		}  else if update.Message != nil && update.Message.Text != "" {
			if checkAuthorized(update.Message.Chat.ID) == false {
				continue
			}
			if update.Message.Text == "introduce" {
				introduce(update)
				continue
			}
			if update.Message.Text == "callback" {
				var silenceKeyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("test1", "1"),
						tgbotapi.NewInlineKeyboardButtonData("test2", "2"),
						tgbotapi.NewInlineKeyboardButtonData("test3", "3"),
					),
				)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Test: ' '%d'", update.Message.Chat.ID))
				msg.ReplyMarkup = silenceKeyboard
				bot.Send(msg)
				continue
			}
			handleTextMessage(&update)
		} else if update.CallbackQuery != nil && update.Message == nil {
			if checkAuthorized(int64(update.CallbackQuery.From.ID)) == false {
				continue
			}
			log.Println("callback message received")
			re := regexp.MustCompile("([^,]*),([^,]*),(.*)")
			match := re.FindStringSubmatch(update.CallbackQuery.Data)
			if len(match) < 3 {
				log.Printf("Can't find any targets for callback")
				continue
			}
			target := match[1]
			silenceTime := match[2]
			alertname := match[3]

			userName := update.CallbackQuery.From.UserName
			switch target {
			case "am":
				silenceId, err := silentAlert(silenceTime, alertname, userName)
				if err != nil {
					msg := tgbotapi.NewMessage(int64(update.CallbackQuery.From.ID), fmt.Sprintf("Error: %s", err))
					bot.Send(msg)
				} else {
					callbackConfig := tgbotapi.CallbackConfig{
						CallbackQueryID: update.CallbackQuery.ID,
						Text:            fmt.Sprintf("Silence successfully set, id: %s", silenceId),
						ShowAlert:       true,
					}

					bot.AnswerCallbackQuery(callbackConfig)
					msg := tgbotapi.NewMessage(int64(update.CallbackQuery.From.ID), fmt.Sprintf("Silence successfully set, id: %s", silenceId))
					bot.Send(msg)
				}
			default:
				msg := tgbotapi.NewMessage(int64(update.CallbackQuery.From.ID), fmt.Sprintf("Unknown target: %s", target))
				bot.Send(msg)
			}
		}
	}
}

// Global
var config_path = flag.String("c", "config.yaml", "Path to a config file")
var listen_addr = flag.String("l", ":9088", "Listen address")
var template_path = flag.String("t", "", "Path to a template file")
var debug = flag.Bool("d", false, "Debug template")


func main() {

	flag.Parse()

	content, err := ioutil.ReadFile(*config_path)
	if err != nil {
		log.Fatalf("Problem reading configuration file: %v", err)
	}
	err = yaml.Unmarshal(content, &cfg)
	if err != nil {
		log.Fatalf("Error parsing configuration file: %v", err)
	}
	fmt.Printf("%+v", cfg)
	if *template_path != "" {
		cfg.TemplatePath = *template_path
	}
	UserStates = initUserStates()
	bot_tmp, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}
	bot = bot_tmp
	if cfg.TemplatePath != "" {

		tmpH = loadTemplate(cfg.TemplatePath)

		if cfg.TimeZone == "" {
			log.Fatalf("You must define time_zone of your bot")
			panic(-1)
		}

	} else {
		*debug = false
		tmpH = nil
	}
	if !(*debug) {
		gin.SetMode(gin.ReleaseMode)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	go telegramBot(bot)

	router := gin.Default()

	router.GET("/ping/:chatid", GET_Handling)
	router.POST("/alert/:chatid", POST_Handling)
	router.Run(*listen_addr)
}


func checkAuthorized(chatId int64) bool {
	// if authorized chats property exists and it's value exists
	if len(cfg.AuthorizedChatIds) > 0 {
		for _, iter := range cfg.AuthorizedChatIds {
			//if chatId found in authorized list, return true
			if iter == chatId {
				log.Printf("%d authorized", chatId)
				return true
			}
		}
	}else{
		// if no chat ids defined
		log.Printf("%d authorized, there is no authorized_chat_ids property, allow all", chatId)
		return true
	}
	// if not found any elem
	log.Printf("%d not authorized, breaking requests", chatId)
	return false
}



