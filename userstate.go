package main

import "fmt"

type UserState struct {
	ID int64
	CurMenu Menuer
	PrevMenu Menuer
	Action string
	UpChannel chan int
}


func userStateChanged() {
	//read from the channel in a loop
	for v := range globalChan {
		fmt.Printf("!!! User state changed\n")
		state := findUserState(v.Message.Chat.ID)
		m := state.CurMenu
		m.RunAction(state.Action, &ActionParams{bot: bot, update: v})
	}
}

func findUserState(id int64) *UserState {
	for _, state := range UserStates {
		if state.ID == id{
			return &state
		}
	}
	return &UserState{}
}

func initUserStates() []UserState{
	var us = []UserState{}
	for _, chatId := range cfg.AuthorizedChatIds{
		us = append(us, UserState{
			ID: int64(chatId),
			CurMenu: newMenu(globalChan),
		})
	}
	return us

}

func setUserState(us *UserState){
	for i, state := range UserStates {
		if state.ID == us.ID{
			UserStates[i] = *us
		}
	}
}
