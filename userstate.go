package main

import "fmt"

type UserState struct {
	ID int64
	CurMenu Menuer
	PrevMenu Menuer
	Action string
	UpChannel chan int
}

func userStateChanger(menu *Menuer, state string){

}

func userStateChanged(){
	//read from the channel in a loop
	for v := range globalChan {
		fmt.Printf("!!! User state changed\n")
		state := findUserState(v.Message.Chat.ID)
		m := state.CurMenu
		m.RunAction(state.Action, &ActionParams{bot: bot, update: v})
	}
}

func findUserState(id int64) *UserState {
	for index, state := range UserStates{
		if state.ID == id{
			return &UserStates[index]
		}
	}
	return &UserState{}
}

func initUserStates(){
	for chatId := range cfg.AuthorizedChatIds{
		UserStates = append(UserStates, UserState{
			ID: int64(chatId),
			CurMenu: newMenu(globalChan),
		})
	}

}
