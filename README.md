# 凡语言

凡语言是一门以中文关键字为核心的通用脚本语言，使用 Go 实现。它借鉴 Go 的设计哲学，强调简单、直接、组合优于继承，同时保留脚本语言的轻量和易用。

## 特性

- 中文关键字与符号运算符并存
- 变量、常量与可选类型标注
- 算术、比较、逻辑运算和三元表达式
- `如果`、`当`、`重复`、`遍历` 等控制结构
- 一等函数、闭包、递归、多返回值
- 显式类型参数（类型在前、名字在后）、默认参数与可变参数
- 模型（字段用 `属性` 声明）、嵌入组合、方法扩展与鸭子类型
- `定义 本地("模块.函数") 函数` 本地绑定，未注册原生实现时静默回退函数体
- Go 风格错误值、`检查` 与 `尝试…捕获`
- 真实模型标签，支持 Go 与凡语言处理器
- 基于文件的模块导入（标准库按 `库/`、`测试/` 分目录）与公开/私有可见性控制
- 单二进制 CLI，支持脚本运行与交互式 REPL

## 快速示例

```凡
变量 名字 = "世界"
打印("你好，", 名字)

定义 函数 求和(整数 甲，整数 乙)
    返回 甲 加 乙
结束

打印(求和(2, 3))

函数 多值求和(整数 开头，整数... 其余)
    变量 合计 = 开头
    遍历 其余 中的 数
        合计 += 数
    结束
    返回 合计
结束

打印(多值求和(1, 2, 3, 4))
```

## 安装

### Linux / macOS

```sh
curl -fsSL https://raw.githubusercontent.com/china-lang-fan/fan-go/main/scripts/install.sh | bash
```

### Windows

```powershell
irm https://raw.githubusercontent.com/china-lang-fan/fan-go/main/scripts/install.ps1 | iex
```

也可以从 [GitHub Releases](https://github.com/china-lang-fan/fan-go/releases) 下载对应平台的压缩包。

安装后验证：

```sh
fan version
```

## 使用

运行脚本：

```sh
fan run 脚本.凡
# 等价中文别名：fan 运行 脚本.凡
```

运行测试（递归发现 `*_测试.凡`，识别 `测试` 前缀函数与 `@测试` 标签）：

```sh
fan test 路径
# 等价中文别名：fan 测试 路径
```

进入 REPL：

```sh
fan repl
# 等价中文别名：fan 交互
```

查看帮助：

```sh
fan help
# 等价中文别名：fan 帮助
```

标准库（fan-sdk）与本仓库分离，按 `库/`、`测试/` 分目录组织。模块搜索顺序为：
脚本所在目录 → 环境变量 `FAN_SDK` → 可执行文件旁 → 系统安装位置。

完整示例见 Rust 实现仓库 [`fan-rust/examples`](https://github.com/china-lang-fan/fan-rust)，完整文档见 <https://china-lang-fan.github.io/>。

## 从源码构建

需要 Go 1.26 或更高版本。

```sh
go build ./cmd/fan
go test ./...
```

## 编辑器支持

- [VS Code 插件](https://github.com/china-lang-fan/fan-code-plugin)：语法高亮、代码片段、注释切换、运行脚本和 REPL
- JetBrains 插件：见 [`fan-jetbrains-plugin`](../fan-jetbrains-plugin)

## 许可证

MIT
