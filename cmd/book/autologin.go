package book

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const cookieCacheFile = "./yuque.cookies"

func AutoLogin() *rod.Page {
	u := launcher.New().
		Headless(true).  // 无界面模式
		NoSandbox(true). // 禁用 sandbox
		MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	page := browser.MustPage("https://www.yuque.com/login")
	page.MustWaitLoad()

	// 尝试加载 Cookie
	httpCookies, err := loadCookies(cookieCacheFile)
	if err == nil && len(httpCookies) > 0 {
		fmt.Println("Login using cached httpCookies...")
		protoCookies := HttpCookiesToProto(httpCookies)

		// 必须先设置 Cookie，再访问域名
		err = page.SetCookies(protoCookies)
		if err != nil {
			log.Printf("Warning: failed to set cookies: %v", err)
		}
		page.MustNavigate("https://www.yuque.com/dashboard")
		page.MustWaitLoad()
		fmt.Println("Logged in via cookies.")
		return page
	}

	//var (
	//	phone     string
	//	checkCode string
	//)

	//// 自动发送验证码
	//page.MustElement(".code-send-button").MustClick()
	//
	//// 填写账号密码（使用 CSS 属性选择器）
	//page.MustElement(`input[data-testid="prefix-phone-input"]`).MustInput(phone)
	//page.MustElement(`input[data-testid="checkCodeInput"]`).MustInput(checkCode)

	// 处理滑块验证码
	if err = solveSliderCaptcha(page); err != nil {
		log.Fatalf("Captcha failed: %v", err)
	}

	// 勾选协议
	checkbox := page.MustElement(`input[data-testid="protocolCheckBox"]`)
	// 检查是否已勾选（可选）
	checked, _ := checkbox.Attribute("checked")
	if checked == nil {
		checkbox.MustClick()
	}

	// 点击登录
	page.MustElement(`button[data-testid="btnLogin"]`).MustClick()

	// 等待跳转
	page.MustWaitNavigation()
	page.MustWaitLoad()

	// 保存 Cookie
	newCookies := page.MustCookies()
	saveCookies(newCookies, cookieCacheFile)
	fmt.Println("Login successful. Cookies saved.")

	return page
}

// getElementRect 返回 x, y, width, height
func getElementRect(el *rod.Element) (x, y, width, height float64, err error) {
	res, err := el.Evaluate(&rod.EvalOptions{
		ThisObj: el.Object, // 将 this 绑定为当前元素
		JS: `(function() {
        const rect = this.getBoundingClientRect();
        return {
            x: rect.x,
            y: rect.y,
            width: rect.width,
            height: rect.height
        };
    })()`,
	})
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// 解析返回值
	obj := res.Value
	x = obj.Get("x").Num()
	y = obj.Get("y").Num()
	width = obj.Get("width").Num()
	height = obj.Get("height").Num()
	return x, y, width, height, nil
}

// solveSliderCaptcha 模拟拖动滑块
func solveSliderCaptcha(page *rod.Page) error {
	fmt.Println("Waiting for captcha slider...")

	// 等待滑块出现（id="nc_3_n1z"）
	//page.Timeout(15*time.Second).
	//	MustWaitElementsMoreThan(`//span[@id='nc_3_n1z']`, 0)
	//slider, err := page.ElementX(`//span[@id='nc_3_n1z']`)
	//if err != nil {
	//	return err
	//}
	slider, err := page.Timeout(10 * time.Second).ElementX(`//*[@id="nc_3_n1z"]`)
	if err != nil {
		// 如果没有滑块，退出
		return fmt.Errorf("no captcha slider found")
	}
	// 获取位置
	x, y, width, height, err := getElementRect(slider)
	if err != nil {
		return fmt.Errorf("get slider position: %w", err)
	}

	// 语雀滑块通常只需向右拖动 ~300px（经验值）
	// 更稳健做法：找轨道宽度，但为简化，我们固定拖动距离
	distance := 300 // 像素

	// 计算中心点
	startX := x + width/2
	startY := y + height/2

	// 移动到滑块中心
	page.Mouse.MustMoveTo(startX, startY)
	// 按下左键
	page.Mouse.MustDown(proto.InputMouseButtonLeft)

	// 模拟拖动（加一点随机性更真实）
	for i := 0; i <= distance; i += 5 {
		x = startX + float64(i)
		y = startY + float64(i%3-1) // 微小抖动，模拟人工
		page.Mouse.MustMoveTo(x, y)
		time.Sleep(time.Millisecond * 15)
	}

	page.Mouse.MustUp(proto.InputMouseButtonLeft)
	fmt.Println("Slider dragged.")

	// 等待验证结果（检查是否有错误提示）
	time.Sleep(2 * time.Second)

	// 可选：检查是否还有滑块（如果还在，说明失败）
	_, err = page.ElementX(`//span[@id='nc_3_n1z']`)
	if err == nil {
		return fmt.Errorf("slider still exists, likely failed")
	}

	return nil
}

// --- Cookie 工具函数 ---
func loadCookies(path string) ([]*http.Cookie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cookies []*http.Cookie
	return cookies, json.Unmarshal(data, &cookies)
}

func saveCookies(cookies []*proto.NetworkCookie, path string) {
	data, _ := json.MarshalIndent(cookies, "", "  ")
	_ = os.WriteFile(path, data, 0600)
}

// --- Main for test ---
func main() {
	page := AutoLogin()
	defer page.Browser().MustClose()

	fmt.Println("Current URL:", page.MustInfo().URL)
	time.Sleep(5 * time.Second) // 查看结果
}
