package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"math/rand"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"golang.org/x/net/html"
)

const (
	ContentSelector        = ".t.z"
	ThreadTextAreaSelector = "#textarea"
)

// 多个回帖的内容
var ReplyContents = []string{
	"感谢楼主分享好片",
	"感谢分享！！",
	"谢谢分享！",
	"感谢分享感谢分享",
	"必需支持",
	"简直太爽了",
	"感谢分享啊",
	"封面还不错",
	"有点意思啊",
	"封面还不错，支持一波",
	"真不错啊",
	"不错不错",
	"这身材可以呀",
	"终于等到你",
	"謝謝辛苦分享",
	"赏心悦目",
	"快乐无限~~",
	"這怎麼受的了啊",
	"谁也挡不住！",
	"分享支持。",
	"这谁顶得住啊",
	"这是要精J人亡啊!",
	"饰演很赞",
	"這系列真有戲",
	"感谢大佬分享v",
	"看着不错",
	"感谢老板分享",
	"可以看看",
	"谢谢分享！！！",
	"真是骚气十足",
	"给我看硬了！",
	"这个眼神谁顶得住。",
	"妙不可言",
	"看硬了，确实不错。",
	"这个我是真的喜欢",
	"如何做到像楼主一样呢",
	"分享一下技巧楼主",
	"身材真不错啊",
	"真是极品啊",
	"这个眼神谁顶得住。",
	"妙不可言",
}

// env变量
var (
	BaseURL        string
	LoginSection   string
	ReplySection   string
	CheckInSection string
	MyBotToken     string
	ChatID         int64
	EnableHeadless bool
	WaitingTime    int
)

// Browser 结构体封装了 chromedp 的执行上下文，用于后续多步操作
type Browser struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// init 用于初始化环境变量
func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Fatalf("加载 .env 文件失败: %v", err)
	}

	// 初始化配置变量
	BaseURL = os.Getenv("BASE_URL")
	LoginSection = os.Getenv("LOGIN_SECTION")
	ReplySection = os.Getenv("REPLY_SECTION")
	CheckInSection = os.Getenv("CHECK_IN_SECTION")
	MyBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")

	// 转换 TELEGRAM_CHAT_ID 为 int64
	if chatIDStr := os.Getenv("TELEGRAM_CHAT_ID"); chatIDStr != "" {
		if id, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
			ChatID = id
		}
	}

	// 转化 ENABLE_HEADLESS 为 bool
	if enableHeadlessStr := os.Getenv("ENABLE_HEADLESS"); enableHeadlessStr != "" {
		if enable, err := strconv.ParseBool(enableHeadlessStr); err == nil {
			EnableHeadless = enable
		}
	}

	// 转化 WAITING_TIME 为 int
	if waitingTimeStr := os.Getenv("WAITING_TIME"); waitingTimeStr != "" {
		if waitingTime, err := strconv.Atoi(waitingTimeStr); err == nil {
			WaitingTime = waitingTime
		}
	}
}

// NewBrowser 创建新的浏览器实例，并启动浏览器，确保上下文可用
func NewBrowser() (*Browser, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoDefaultBrowserCheck,
		// 非无头模式便于调试, 本地测试改成false，启动图形界面
		chromedp.Flag("headless", EnableHeadless),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.NoFirstRun,
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	// 创建分配器上下文
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	// 创建 Chrome 上下文
	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	// 启动浏览器（空任务），确保 ctx 正常启动
	if err := chromedp.Run(ctx); err != nil {
		cancelCtx()
		cancelAlloc()
		return nil, err
	}
	// 合并取消函数
	combinedCancel := func() {
		cancelCtx()
		cancelAlloc()
	}
	return &Browser{
		ctx:    ctx,
		cancel: combinedCancel,
	}, nil
}

// Close 关闭浏览器实例
func (b *Browser) Close() {
	b.cancel()
}

// Execute 用于执行一组 chromedp.Action，并设置一个超时
func (b *Browser) Execute(actions ...chromedp.Action) error {
	ctx, cancel := context.WithTimeout(b.ctx, 60*time.Second)
	defer cancel()
	return chromedp.Run(ctx, actions...)
}

// NavigateTo 导航到指定页面
func (b *Browser) NavigateTo(url string) error {
	return b.Execute(chromedp.Navigate(url))
}

// WaitForElement 等待页面中指定的元素可见
func (b *Browser) WaitForElement(selector string) error {
	return b.Execute(chromedp.WaitVisible(selector))
}

// GetHTML 获取指定 js 路径对应的HTML内容
func (b *Browser) GetHTML(sel string) (string, error) {
	var html string
	err := b.Execute(chromedp.OuterHTML(sel, &html, chromedp.ByQuery))
	return html, err
}

// Click 模拟点击操作
func (b *Browser) Click(selector string) error {
	return b.Execute(chromedp.Click(selector, chromedp.ByQuery))
}

// Input 模拟输入文本
func (b *Browser) Input(selector, text string) error {
	return b.Execute(
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.SendKeys(selector, text, chromedp.ByQuery),
	)
}

// GetFirstPost 从页面 HTML 中提取“广告连接”后第一个符合条件的帖子数据
func GetFirstPost(htmlContent string) (title string, href string, err error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", "", err
	}

	// 定位 table#ajaxtable 下的第二个 tbody
	tbody := doc.Find("table#ajaxtable tbody").Eq(1)
	if tbody.Length() == 0 {
		return "", "", errors.New("未找到第二个 tbody")
	}

	var target *goquery.Selection
	found := false
	// 遍历 tbody 的所有子节点，查找注释节点包含“广告连接”
	tbody.Contents().EachWithBreak(func(i int, s *goquery.Selection) bool {
		for _, node := range s.Nodes {
			if node.Type == html.CommentNode && strings.Contains(node.Data, "广告连接") {
				// 从该注释节点向后查找第一个符合条件的 tr
				target = s.NextFiltered("tr.tr3.t_one").First()
				if target.Length() > 0 {
					found = true
					return false // 找到后退出遍历
				}
			}
		}
		return true
	})
	if !found || target == nil || target.Length() == 0 {
		return "", "", errors.New("未找到广告连接后的帖子")
	}

	// 根据帖子的结构，假设帖子的标题链接在 target 内的 a.subject 中
	a := target.Find("a.subject").First()
	if a.Length() == 0 {
		return "", "", errors.New("未找到帖子的链接元素")
	}
	title = strings.TrimSpace(a.Text())
	href, exists := a.Attr("href")
	if !exists {
		return "", "", errors.New("帖子链接中没有 href 属性")
	}
	return title, href, nil
}

// 检查登陆状态是否有效，若无效则执行登陆并加载cookie
func (b *Browser) CheckLoginStatus() error {
    // 等待 header 元素加载
    if err := b.WaitForElement("div.header_up_sign"); err != nil {
        return err
    }
    // 获取 header 的 HTML 内容（如果页面中有多个 div.header_up_sign，这里取第一个）
    headerHTML, err := b.GetHTML("div.header_up_sign")
    if err != nil {
        return err
    }
    
    // 检查 cookies 文件是否存在且未过期（不超过7天）
    needLogin := false
    cookiesExpired := false
    
    // 如果 header 包含"登录"且不包含"退出"，认为未登录
    if strings.Contains(headerHTML, "登录") && !strings.Contains(headerHTML, "退出") {
        needLogin = true
    }
    
    // 检查 cookies 文件是否存在
    fileInfo, err := os.Stat("./cookies")
    if err != nil {
        // cookies 文件不存在
        needLogin = true
    } else {
        // 检查 cookies 文件的修改时间，如果超过7天则视为过期
        if time.Since(fileInfo.ModTime()).Hours() > 24*7 {
            log.Printf("cookies 已过期（超过7天），需要重新登录")
            cookiesExpired = true
            needLogin = true
        }
    }
    
    // 如果 cookies 过期，删除文件
    if cookiesExpired {
        err := os.Remove("./cookies")
        if err != nil {
            log.Printf("删除过期 cookies 文件失败: %v", err)
        } else {
            log.Printf("已删除过期 cookies 文件")
        }
    }
    
    if needLogin {
        // 执行登录操作
        if err := b.Login(); err != nil {
            return err
        }
        // 登录成功后，保存 cookies 到文件
        cookiesFile := b.SaveCookies()
        log.Printf("登录成功，cookies 已保存到 %s", cookiesFile)
    } else if fileInfo != nil && fileInfo.Size() > 0 {
        // cookies 文件存在且不为空，执行 setCookies 操作
        if err := b.SetCookies(); err != nil {
            return err
        }
        log.Printf("使用已有的 cookies 登录成功")
    } else {
        log.Printf("检测到已登录状态")
    }
    return nil
}

// 填写登录表单中：用户名、密码、安全问题（选择“我的中学校名”，value="4"）、答案
func (b *Browser) Login() error {
	// 直接导航到首页（index.html），因为登录表单在首页中
	if err := b.NavigateTo(BaseURL + LoginSection); err != nil {
		return err
	}

	// 等待登录表单区域加载
	if err := b.WaitForElement(".cc.p10.regItem"); err != nil {
		return err
	}

	err := b.Execute(
		chromedp.SendKeys(`//*[@id="main"]/form/div/table/tbody/tr/td/div/dl[1]/dd/input`, os.Getenv("FORUM_USERNAME")),
		chromedp.SendKeys(`//*[@id="main"]/form/div/table/tbody/tr/td/div/dl[2]/dd/input`, os.Getenv("FORUM_PASSWORD")),
		chromedp.SetValue(
			`//*[@id="main"]/form/div/table/tbody/tr/td/div/dl[3]/dd/select`,
			os.Getenv("SECURITY_QUESTION"),
			chromedp.BySearch,
		),
		chromedp.SendKeys(`//*[@id="main"]/form/div/table/tbody/tr/td/div/dl[4]/dd/input`, os.Getenv("SECURITY_ANSWER")),
		chromedp.Click(`//*[@id="main"]/form/div/table/tbody/tr/td/div/dl[7]/dd/input`),
		// 登录后等待页面切换，等待 header 中出现“退出”
		chromedp.WaitVisible(`div.header_up_sign`, chromedp.ByQuery),
		// 小等待确保登录后的 cookie 已经同步
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		log.Printf("登陆操作出错：%v", err)
		return err
	}

	return nil
}

// saveCookies 登陆后保存cookies到
func (b *Browser) SaveCookies() string {
	// 使用写入模式打开，并清空原文件内容
	file, err := os.OpenFile("./cookies", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Printf("打开cookies文件失败: %v", err)
		return ""
	}
	defer file.Close()

	err = b.Execute(
		// 登录后等待页面切换，等待 header 中出现“退出”
		chromedp.WaitVisible(`div.header_up_sign`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return err
			}

			j, err := json.Marshal(cookies)
			if err != nil {
				return err
			}

			// 写入 JSON 数据到文件
			_, err = file.Write(j)
			return err
		}),
	)
	if err != nil {
		log.Fatal("cookies保存失败: ", err)
	}

	return file.Name()
}

// setCookies 读取Cookies文件并自动登录
func (b *Browser) SetCookies() error {
	var text string
	return b.Execute(
		chromedp.ActionFunc(func(ctx context.Context) error {
			file, err := os.Open("./cookies")
			if err != nil {
				return err
			}

			defer file.Close()

			// 读取文件数据
			jsonBlob, err := io.ReadAll(file)
			if err != nil {
				return err
			}

			var cookies []*network.CookieParam
			// Json解码
			err = json.Unmarshal(jsonBlob, &cookies)
			if err != nil {
				return err
			}
			err = network.SetCookies(cookies).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
		chromedp.Reload(),
		chromedp.Title(&text),
	)
}

// replyPost 回帖
func (b *Browser) ReplyPost() error {
	// 等待回帖区域加载
	if err := b.WaitForElement(ThreadTextAreaSelector); err != nil {
		return err
	}
	// 随机选择回帖内容
	replyContent := ReplyContents[time.Now().Unix()%int64(len(ReplyContents))]
	// 输入回帖内容
	if err := b.Input(ThreadTextAreaSelector, replyContent); err != nil {
		return err
	}
	// 点击回帖按钮
	if err := b.Click(".btn.fpbtn"); err != nil {
		return err
	}
	// 等待3秒，刷新页面
	time.Sleep(3 * time.Second)
	return nil
}

// 到签到页面签到
func (b *Browser) CheckIn() (string, error) {
	// 直接导航到签到页面
	if err := b.NavigateTo(BaseURL + CheckInSection); err != nil {
		return "", err
	}
	// 等待签到按钮加载
	if err := b.WaitForElement("#submit_bbb"); err != nil {
		return "", err
	}
	// 随机选择一个表情
	expressions := []string{"kx", "ng", "ym", "wl", "nu", "ch", "fd", "yl", "shuai"}
	rand.Seed(time.Now().UnixNano())
	selected := expressions[rand.Intn(len(expressions))]
	// 获取签到结果文本
	var resultText string
	// 执行选择表情与点击签到按钮的操作
	err := b.Execute(
		// 点击选中的表情对应的 radio 按钮
		chromedp.Click(`input[name="qdxq"][value="`+selected+`"]`, chromedp.ByQuery),
		// 点击签到按钮（根据 index.html，其 id 为 submit_bbb）
		chromedp.Click(`#submit_bbb`, chromedp.ByQuery),
		// 等待签到结果文本加载
		chromedp.Text("span.f14", &resultText, chromedp.ByQuery),
	)
	if err != nil {
		log.Printf("签到操作出错：%v", err)
		return "", err
	}
	log.Printf("%s 签到结果：%s", time.Now().Format("2006-01-02"), resultText)
	return resultText, nil
}

// sendTelegramNotification 发送 Telegram 消息通知
func SendTelegramNotification(message string) error {
	bot, err := tgbotapi.NewBotAPI(MyBotToken)
	if err != nil {
		log.Printf("创建 Telegram Bot 实例失败: %v", err)
		return err
	}
	bot.Debug = false

	// 构建发送消息对象
	msg := tgbotapi.NewMessage(ChatID, message)
	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("发送 Telegram 消息通知失败: %v", err)
		return err
	}
	return nil
}

func main() {
	// 随机睡眠 0~120 秒
	rand.Seed(time.Now().UnixNano())
	delay := rand.Intn(WaitingTime)
	log.Printf("等待 %d 秒后开始执行", delay)
	time.Sleep(time.Duration(delay) * time.Second)

	// 创建浏览器实例
	browser, err := NewBrowser()
	if err != nil {
		log.Fatalf("无法创建浏览器实例: %v", err)
	}
	defer browser.Close()

	// 访问论坛回帖页面
	replyURL := BaseURL + ReplySection
	if err = browser.NavigateTo(replyURL); err != nil {
		log.Printf("导航回帖页失败: %v", err)
		return
	}
	// 检查登陆状态
	if err = browser.CheckLoginStatus(); err != nil {
		log.Printf("检查登陆状态出错：%v", err)
		return
	}

	// 访问论坛回帖页面并提取帖子数据
	if err = browser.NavigateTo(replyURL); err != nil {
		log.Printf("导航回帖页失败: %v", err)
		return
	}
	if err = browser.WaitForElement(ContentSelector); err != nil {
		log.Printf("等待元素失败: %v", err)
		return
	}
	htmlContent, err := browser.GetHTML("body")
	if err != nil {
		log.Printf("获取HTML失败: %v", err)
		return
	}
	title, href, err := GetFirstPost(htmlContent)
	if err != nil {
		log.Printf("提取数据失败: %v", err)
		return
	}
	log.Printf("找到帖子：%s, 链接：%s", title, href)
	fullURL := BaseURL + href
	if err = browser.NavigateTo(fullURL); err != nil {
		log.Printf("打开帖子失败: %v", err)
		return
	}

	// 回帖
	if err = browser.ReplyPost(); err != nil {
		log.Printf("回帖失败: %v", err)
		return
	}

	// 签到
	checkInResult, err := browser.CheckIn()
	if err != nil {
		log.Printf("签到失败: %v", err)
		return
	}

	// 打印今天的日期，以及签到成功的信息
	successMsg := fmt.Sprintf("%s 签到结果：%s", time.Now().Format("2006-01-02"), checkInResult)
	// 发送 Telegram 通知
	if err := SendTelegramNotification(successMsg); err != nil {
		log.Printf("发送 Telegram 通知失败: %v", err)
	}
}
