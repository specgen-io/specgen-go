package format

import "fmt"

func Message(msgAndArgs ...interface{}) string {
	switch len(msgAndArgs) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%v", msgAndArgs[0])
	default:
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
}

func WithCustomMessage(source string, msgAndArgs ...interface{}) string {
	custom := Message(msgAndArgs...)
	switch {
	case custom == "":
		return source
	case source == "":
		return custom
	}
	return fmt.Sprintf("%s: %s", source, custom)
}
