package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
)


type AlertmanagerMenu struct {
	Channel chan tgbotapi.Update
	States map[string] func(*ActionParams)(error, ActionReply)
}


func newAlertmanagerMenu(c chan *tgbotapi.Update)(Menuer){
	m := new(AlertmanagerMenu)
	return m
}

func (m *AlertmanagerMenu) DefaultAction() string {
	return ""
}

func handleCurrentMenu(params *ActionParams){
	msg := tgbotapi.NewMessage(params.update.Message.Chat.ID, "default text")
	switch params.update.Message.Text {
	case "list alerts":
		msg.Text = "Here will be the list of alerts :-) Later..."
	case "list silences":
		msg.Text = "Here will be the list of silences :-) Later..."
	case "back":
		msg.Text = "ok, back"
		state := findUserState(params.update.Message.Chat.ID)
		state.Action = ""
		if state.PrevMenu != Menuer(nil){
			state.CurMenu = state.PrevMenu
		}
		globalChan <- params.update

	}
	params.bot.Send(msg)
}


func (m *AlertmanagerMenu) RunAction(action string,params *ActionParams) (ActionReply, error){
	switch action {
	case "handlealertmanager":
		handleCurrentMenu(params)
	default:
		m.Draw(params)
	}
	return ActionReply{}, nil
}

func (m *AlertmanagerMenu) Draw(params *ActionParams){
	menuKeys := []tgbotapi.KeyboardButton{}
	menuKeys = append(menuKeys, tgbotapi.NewKeyboardButton("list silences"))
	menuKeys = append(menuKeys, tgbotapi.NewKeyboardButton("list alerts"))
	menuKeys = append(menuKeys, tgbotapi.NewKeyboardButton("back"))
	msg := tgbotapi.NewMessage(params.update.Message.Chat.ID, "AM menu")
	var keyboard = tgbotapi.NewReplyKeyboard(menuKeys)
	keyboard.OneTimeKeyboard = true
	msg.ReplyMarkup = keyboard
	state := findUserState(params.update.Message.Chat.ID)
	state.Action = "handlealertmanager"
	params.bot.Send(msg)
}