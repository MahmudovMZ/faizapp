package telegram

func (h *Handler) getBotState(chatID int64) string {
	h.lock.RLock()
	defer h.lock.RUnlock()

	return h.botState[chatID]
}

func (h *Handler) setBotState(chatID int64, state string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.botState[chatID] = state
}

func (h *Handler) deleteBotState(chatID int64) {
	h.lock.Lock()
	defer h.lock.Unlock()
	delete(h.botState, chatID)
}

func (h *Handler) getUserState(chatID int64) string {
	h.lock.RLock()
	defer h.lock.RUnlock()

	return h.userState[chatID]
}

func (h *Handler) setUserState(chatID int64, state string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.userState[chatID] = state
}

func (h *Handler) deleteUserState(chatID int64) {
	h.lock.Lock()
	defer h.lock.Unlock()

	delete(h.userState, chatID)
}

func (h *Handler) initUserData(chatID int64) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.userData[chatID] = make(map[string]string)
}

func (h *Handler) getUserDataValue(chatID int64, key string) string {
	h.lock.RLock()
	defer h.lock.RUnlock()

	return h.userData[chatID][key]
}

func (h *Handler) setUserDataValue(chatID int64, key, value string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.userData[chatID] == nil {
		h.userData[chatID] = make(map[string]string)
	}

	h.userData[chatID][key] = value
}

func (h *Handler) deleteUserData(chatID int64) {
	h.lock.Lock()
	defer h.lock.Unlock()

	delete(h.userData, chatID)
}
