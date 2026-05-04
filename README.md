# PanPlayer 115

`PanPlayer 115` 是一个用 `Go + Wails v3 + Vue 3 + Vuetify` 实现的桌面播放器原型：

- 启动后直接展示 115 网盘目录
- 用 115 官方二维码接口扫码登录
- 本地保存登录态，不依赖你自建服务端
- 选中视频后通过本地代理交给外部播放器播放

## 当前能力

- 115 二维码登录
- 115 Cookie 登录
- 登录态恢复
- 浏览目录
- 紧凑型文件管理界面
- 115 离线下载任务管理
- 搜索、类型筛选、排序、快捷目录访问
- 双击视频交给外部播放器
- 记住上次播放进度，再次打开时直接续播
- 为单个视频绑定外挂字幕，并在下次播放时自动带上
- 起播跳转，支持手动指定秒数或时间点
- 自定义多个播放器路径
- 切换默认播放器

## 运行前提

1. 安装 Go
2. 安装 Wails v3 CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.80`
3. 系统能运行 Wails 桌面应用
4. 安装以下任一播放器，或者在应用里手动指定路径：
   - `mpv`
   - `VLC`
   - `PotPlayer`
   - `MPC-HC`
   - `MPC-BE`

## 开发运行

```bash
wails3 dev
```

前端位于 `frontend/`，使用 `Vite` 构建；直接运行 `wails3 dev` / `wails3 build` 即可自动安装和编译前端依赖。

## 打包

```bash
wails3 build
```

当前仓库已按 Windows 本地开发做了精简。默认构建产物在：

- `bin/panplayer115.exe`
- 安装器：`build/windows/nsis/panplayer115-installer.exe`

## 凭证与设置

本地配置文件默认保存在：

- Windows: `%AppData%\\panplayer\\config.json`

保存内容包括：

- 默认播放器
- 各播放器路径
- 115 登录 cookie 凭证
- 上次浏览目录
- 每个视频的续播记录
- 每个视频的字幕路径

日志文件默认在：

- Windows: `%AppData%\\panplayer\\panplayer.log`
- `mpv` 日志: `%AppData%\\panplayer\\mpv.log`

`mpv` 的续播中间状态默认写在：

- Windows: `%AppData%\\panplayer\\mpv-watch-later`

这份配置只保存在本机，不会上传到任何服务端。

## 说明

- 当前已经做成播放器适配层，后面可以继续往更多播放器和平台扩展
- 现在开发链默认只保留 Windows 构建；`mpv / VLC` 仍然可以作为后续跨平台播放器目标
- 播放时会先经过本地回环代理，再由外部播放器拉流
- 续播记录目前以 `mpv` 适配最完整；其他播放器已经支持基础启动、字幕和部分起播跳转能力
