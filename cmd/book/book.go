package book

import (
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/spf13/cobra"
	"time"
)

const yuqueURL = "https://www.yuque.com/"

func NewBook() *cobra.Command {
	bookCmd := &cobra.Command{
		Use:     "book",
		Short:   "导出语雀知识库",
		Long:    "导出语雀知识库",
		Example: "yuque book",
	}
	bookCmd.AddCommand(NewPassBookExport())
	bookCmd.AddCommand(NewCookieBookExport())
	//bookCmd.AddCommand(NewMineBookExport())

	return bookCmd
}

func NewMineBookExport() *cobra.Command {
	exportCmd := &cobra.Command{
		Use:     "mine-export",
		Short:   "导出自己的知识库",
		Long:    "导出自己的知识库",
		Example: "yuque book mine-export",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	return exportCmd
}
func NewCookieBookExport() *cobra.Command {
	var (
		bookID    string
		namespace string
		output    string
		cookie    string
	)
	exportCmd := &cobra.Command{
		Use:     "cookie-export",
		Short:   "根据cookie导出知识库",
		Long:    "根据cookie导出知识库",
		Example: "yuque book cookie-export",
		Run: func(cmd *cobra.Command, args []string) {
			err := Export(bookID, namespace, cookie, output)
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	flags := exportCmd.Flags()
	flags.SortFlags = false
	flags.StringVar(&bookID, "bookid", "", "目标知识库ID")
	flags.StringVar(&cookie, "cookie", "", "目标知识库访问cookie")
	flags.StringVarP(&output, "output", "-o", "", "输出目录")
	flags.StringVar(&namespace, "namespace", "", "目标知识库空间, 指定全名（用户slug+知识库slug）, 例：fsdfs/gdsgs")

	_ = exportCmd.MarkFlagRequired("pass")
	_ = exportCmd.MarkFlagRequired("bookid")

	return exportCmd
}
func NewPassBookExport() *cobra.Command {
	var (
		bookID    string
		namespace string
		output    string
		pass      string
	)
	exportCmd := &cobra.Command{
		Use:     "pass-export",
		Short:   "导出带密码的知识库",
		Long:    "无需登录即可访问带密码的知识库，则适用。此只处理输入访问密码的情况，不兼容登录情况下",
		Example: "yuque book pass-export",
		Run: func(cmd *cobra.Command, args []string) {
			cookie := GetCookie(namespace, pass)
			err := Export(bookID, namespace, cookie, output)
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	flags := exportCmd.Flags()
	flags.SortFlags = false
	flags.StringVar(&pass, "pass", "", "目标知识库访问密码")
	flags.StringVar(&bookID, "bookid", "", "目标知识库ID")
	flags.StringVarP(&output, "output", "-o", "", "输出目录")
	flags.StringVar(&namespace, "namespace", "", "目标知识库空间,指定全名（用户slug+知识库slug）, 例：fsdfs/gdsgs")

	_ = exportCmd.MarkFlagRequired("pass")
	_ = exportCmd.MarkFlagRequired("book-id")

	return exportCmd
}

func GetCookie(namespace string, pass ...string) string {
	u := launcher.New().
		Headless(true).  // 无界面模式
		NoSandbox(true). // 禁用 sandbox
		MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	page := browser.MustPage(fmt.Sprintf("%s%s", yuqueURL, namespace))
	page.MustWaitLoad()

	if len(pass) > 0 {
		// 等待密码输入框出现
		page.Timeout(15*time.Second).
			MustWaitElementsMoreThan("#main-right-content > div.WebPureLayout-module_rightMainContentChildren_Lpml2 > div > div > div > div.index-module_form_tlTsV > input", 0)

		inputEle, err := page.Element("#main-right-content > div.WebPureLayout-module_rightMainContentChildren_Lpml2 > div > div > div > div.index-module_form_tlTsV > input")
		if err != nil {
			// 没有获取input就失败
			panic(err)
		}
		inputEle.MustInput(pass[0])

		err = page.Reload()
		if err != nil {
			return ""
		}
	}

	// 获取当前页面所在域的所有 Cookie
	cookies := page.MustCookies()
	return CookieMapToHeader(cookies)
}
