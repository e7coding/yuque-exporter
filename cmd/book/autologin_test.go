package book

import (
	"fmt"
	"testing"
	"time"
)

func TestAutoLogin(t *testing.T) {
	page := AutoLogin()
	defer page.Browser().MustClose()

	fmt.Println("Current URL:", page.MustInfo().URL)
	time.Sleep(5 * time.Second) // 查看结果
}
