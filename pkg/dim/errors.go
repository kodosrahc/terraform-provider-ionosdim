package dim

import "fmt"

type Error struct {
	Func    string
	Code    int64
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s error (%d): %s", e.Func, e.Code, e.Message)
}
