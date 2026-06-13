package tui

import tea "github.com/charmbracelet/bubbletea"

var arabicKeyboardAliases = map[string]string{
	"ض":  "q",
	"ص":  "w",
	"ث":  "e",
	"ق":  "r",
	"ف":  "t",
	"غ":  "y",
	"ع":  "u",
	"ه":  "i",
	"خ":  "o",
	"ح":  "p",
	"ش":  "a",
	"س":  "s",
	"ي":  "d",
	"ب":  "f",
	"ل":  "g",
	"ا":  "h",
	"ت":  "j",
	"ن":  "k",
	"م":  "l",
	"ئ":  "z",
	"ء":  "x",
	"ؤ":  "c",
	"ر":  "v",
	"لا": "b",
	"ى":  "n",
	"ة":  "m",
	"و":  ",",
	"ز":  ".",
	"ظ":  "/",
	"؟":  "?",
}

func commandKey(msg tea.KeyMsg) string {
	key := msg.String()
	if msg.Type != tea.KeyRunes {
		return key
	}
	if alias, ok := arabicKeyboardAliases[key]; ok {
		return alias
	}
	return key
}
