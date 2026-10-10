# SetupAssistant

一键装机助手。在 Windows 上按 `app.json` 选择模块，下载安装包到 `./temp`，再静默安装。

## 运行

```bash
go run .
```

程序启动时如果还不是管理员，会提权一次。点「是」后在新窗口继续；点「否」则退出。

## 使用

菜单来自当前目录的 `app.json`。分类下面是软件，有版本号时再进入版本列表。

| 按键 | 作用 |
| --- | --- |
| ↑↓ | 移动 |
| 空格 | 勾选，或在已选列表里取消 |
| 回车 | 确认并进入下一级；选完后回到主菜单 |
| Tab | 查看已选模块 |
| Esc | 返回上一级。主菜单上不退出 |
| F5 | 开始安装。列表为空时提示「安装列表为空」 |

F5 之后读取本机固定磁盘分区，勾选一个作为安装盘。实际目录是「盘符 + 版本里的 `dir`」。未写 `dir` 时用软件名称。例如选 `D:\`，Notepad++ 安装到 `D:\Editor\Notepad++`。

没有 `url` 的项目会跳过下载。没有 `package` 的项目会跳过安装。

## 下载和安装

安装包保存到 `./temp`，文件名用 `package`。下载是流式的，先写入 `.part`，完成后改成正式文件。大小与服务器一致的已有文件会跳过。进度条分两行：上面是大小、百分比、速度和用时，下面是青到绿的色块。

有安装包的项目用完整路径静默安装：

```text
安装包 /S /D=目标目录
```

安装过程中显示「目标目录 安装中」并转圈，结束后显示「目标目录 安装完成」。安装包留在 `./temp`，不自动打开软件。

`temp/` 和 `*.part` 已写入 `.gitignore`。

## 依赖

| 库 | 模块 | 用途 |
| --- | --- | --- |
| bubbletea | `github.com/charmbracelet/bubbletea` v1.3.6 | 终端界面 |
| huh | `github.com/charmbracelet/huh` v1.0.0 | 取消选择 |
| lipgloss | `github.com/charmbracelet/lipgloss` v1.1.0 | 界面样式 |
| pb | `github.com/cheggaaa/pb/v3` v3.2.1 | 下载进度条 |
| resty | `github.com/go-resty/resty/v2` v2.17.2 | 下载 |
