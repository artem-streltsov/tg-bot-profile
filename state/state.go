package state

import (
	"sync"
)

type State string

const (
	StateIdle             State = "idle"
	StateEditingName      State = "editing_name"
	StateEditingZodiac    State = "editing_zodiac"
	StateEditingBirthDate State = "editing_birthdate"
	StateEditingBirthTime State = "editing_birthtime"
	StateBuyingPremium    State = "buying_premium"
)

type UserState struct {
	State State
}

var userStates = make(map[int64]*UserState)
var mutex sync.RWMutex

func SetState(userID int64, state State) {
	mutex.Lock()
	defer mutex.Unlock()
	userStates[userID] = &UserState{State: state}
}

func GetState(userID int64) State {
	mutex.RLock()
	defer mutex.RUnlock()
	if userState, exists := userStates[userID]; exists {
		return userState.State
	}
	return StateIdle
}

func ClearState(userID int64) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(userStates, userID)
}
