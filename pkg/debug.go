package crontools

var debugEnabled bool

func SetDebug(enabled bool) {
	debugEnabled = enabled
}

func DebugEnabled() bool {
	return debugEnabled
}
