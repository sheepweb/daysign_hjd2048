# daysign_hjd2048

本项目是一个在2048核基地自动回帖与签到程序，通过 [chromedp](https://github.com/chromedp/chromedp) 实现网页自动化操作，同时使用 [Telegram Bot API](https://core.telegram.org/bots/api) 在签到成功后发送通知。

## 特性

- 使用 chromedp 实现网页自动化操作，使用 headless 模式，模拟人工操作
- 自动登录、保存并加载 cookies
- 随机等待时间，避免被检测(定时任务的时间 + 随机等待时间10分钟)
- 自动回帖与签到操作
- 签到成功后发送 Telegram 通知，暂时只支持单个 chatID
- 内置 Makefile 支持跨平台构建
- 使用 GitHub Action 自动构建发布

## 前置条件

- chrome/chromium
- Go 1.23.5+

## 使用说明

### 1. 本地构建与测试

**修改 `.env.example` 文件为 `.env`，并填入你的配置信息**

```bash
go build -o app main.go
./app
```

### 2. 服务器部署

1. 安装chrome/chromium

```bash
# Ubuntu
sudo apt-get update
sudo apt-get install -y chromium-browser
```


2. 克隆本项目到服务器

```bash
git clone https://github.com/Mr-jello/daysign_hjd2048.git
cd daysign_hjd2048
```

3. 复制环境变量模板文件：

```bash
cp .env.example .env
```
- 修改 .env 文件，填入你的配置信息

4. 执行 Makefile 构建

```bash
make build-all
```
- 选择适合服务器架构，将生成的二进制文件 `daysign2048_xx` 修改为 `daysign2048`上传到服务器
- 例如：/root目录下

5. 使用 Crontab 定时任务

```bash
# 确保二进制文件有执行权限
chmod +x /root/daysign2048

# 编辑定时任务
crontab -e

# 添加以下内容，每天凌晨 0 点 15 分执行
15 0 * * * cd /root && ./daysign2048 >> /root/daysign2048.log 2>&1
```

## 未来计划
- [x] 使用环境变量配置信息用户信息, 系统配置信息
- [x] 通知信息更加详细
- [ ] 支持更多通知方式，如钉钉、企业微信，邮箱等
- [ ] 自定义安全问题与答案
- [ ] 程序内置定时任务，无需使用 Crontab