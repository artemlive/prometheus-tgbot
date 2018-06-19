package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/labstack/gommon/log"
)

type ActionReply struct{

}

type ActionParams struct {
	bot *tgbotapi.BotAPI
	update *tgbotapi.Update
}

type Menuer interface {
	DefaultAction() string
	RunAction(string, *ActionParams) (ActionReply, error)
}

type Menu struct {
	Channel *chan tgbotapi.Update
	States map[string] func(*ActionParams)(error, ActionReply)
}


func newMenu(c chan *tgbotapi.Update)(Menuer){
	m := new(Menu)
	return m
}

func (m *Menu) DefaultAction() string {
	return ""
}

func (m *Menu) alertmanagerMenu(params *ActionParams) {
	log.Printf("We are in alertmanager menu handler")
	state := findUserState(params.update.Message.Chat.ID)
	state.CurMenu = newAlertmanagerMenu(globalChan)
	state.Action = ""
	globalChan <- params.update
}

func (m *Menu) handleMain(params *ActionParams){
	switch params.update.Message.Text {
	case "alertmanager":
		m.alertmanagerMenu(params)
	}
}

func (m *Menu) RunAction(action string, params *ActionParams) (ActionReply, error){
	switch action {
	case "handlemain":
		m.handleMain(params)
	default:
		m.Draw(params)
	}
	return ActionReply{}, nil
}

func (m *Menu) Draw(params *ActionParams){
	menuKeys := []tgbotapi.KeyboardButton{}
	menuKeys = append(menuKeys, tgbotapi.NewKeyboardButton("alertmanager"))
	menuKeys = append(menuKeys, tgbotapi.NewKeyboardButton("close"))
	msg := tgbotapi.NewMessage(params.update.Message.Chat.ID, "MainMenu")
	var keyboard = tgbotapi.NewReplyKeyboard(menuKeys)
	keyboard.OneTimeKeyboard = true
	msg.ReplyMarkup = keyboard
	state := findUserState(params.update.Message.Chat.ID)
	state.PrevMenu = state.CurMenu
	state.Action = "handlemain"
	params.bot.Send(msg)
}