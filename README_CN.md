# go-toml

[TOML](https://github.com/mojombo/toml) 格式 Go 语言支持包.

本包支持 TOML 版本
[v0.1.0](https://github.com/mojombo/toml/blob/master/versions/toml-v0.1.0.md)

[![Build Status](https://travis-ci.org/pelletier/go-toml.png?branch=master)](https://travis-ci.org/pelletier/go-toml)

这里有一篇关于 [TOML, 新的简洁配置语言](http://hit9.org/post/toml.html) 中文简介.

## Import

    import "github.com/pelletier/go-toml"


## 使用

假设你有一个TOML文件 `example.toml` 看起来像这样:

```toml
# 注释以"#"开头, 这是多行注释, 可以分多行
# go-toml 把这两行注释绑定到紧随其后的 key, 也就是 title

title = "TOML Example" # 这是行尾注释, go-toml 把此注释绑定给 title

# 虽然只有一行, 这也属于多行注释, go-toml 把此注释绑定给 owner
[owner] # 这是行尾注释, go-toml 把这一行注释绑定到 owner

name = "om Preston-Werner" # 这是行尾注释, go-toml 把这一行注释绑定到 owner.name

# 下面列举 TOML 所支持的类型与格式要求
organization = "GitHub" # 字符串
bio = "GitHub Cofounder & CEO\nLikes tater tots and beer." # 字符串可以包含转义字符
dob = 1979-05-27T07:32:00Z # 日期, 必须使用 RFC3339 格式. 对 Go 来说这很简单.

[database]
server = "192.168.1.1"
ports = [ 8001, 8001, 8002 ] # 数组, 其元素类型也必须是TOML所支持的. Go 语言下类型是 slice
connection_max = 5000 # 整型, go-toml 使用 int64 类型
enabled = true # 布尔型

[servers]

  # 可以使用缩进, tabs 或者 spaces 都可以, 毫无问题.
  [servers.alpha]
  ip = "10.0.0.1" # IP 格式只能用字符串了
  dc = "eqdc10"

  [servers.beta]
  ip = "10.0.0.2"
  dc = "eqdc10"

[clients]
data = [ ["gamma", "delta"], [1, 2] ] # 又一个数组
donate = 49.90 # 浮点, go-toml 使用 float64 类型
```

读取 `servers.alpha` 部分好像这样:

```go
import (
    "fmt"
    "github.com/pelletier/go-toml"
)
func main() {
    conf, err := toml.LoadFile("good.toml")
    if err != nil {
        return
    }
    fmt.Printf("%#v\n", conf.Get("servers.alpha.ip"))
    fmt.Printf("%#v\n", conf.Get("servers.alpha.dc"))
    fmt.Printf("%#v\n", conf.GetComment("servers.alpha"))
    fmt.Printf("%#v\n", conf.GetComment("servers.alpha.ip"))
}
```
输出是这样的:
```
10.0.0.1
eqdc10
{Multiline:[] EndOfLine:# 可以使用缩进, Tabs 或者 spaces 都可以, 毫无问题.}
{Multiline:[] EndOfLine:# IP 格式只能用字符串了}
```

您应该注意到了注释的表现形式, go-toml 提供了注释支持.

## 文档

Go DOC 文档请访问
[godoc.org](http://godoc.org/github.com/pelletier/go-toml).

* 支持自字符串和文件名两种方式载入 TOML
* 如果不通上述方式, 采用自写代码生成 TomlTree 实例,记得要 (*TomlTree).Init()
* 支持注释
* 支持 String()
* 支持 SaveToFile()
* 删除某个key 通过 Set("key",nil)

## 贡献

请使用 GitHub 系统提出 issues 或者 pull 补丁到
[pelletier/go-toml](https://github.com/pelletier/go-toml). 欢迎任何反馈！


## License

Copyright (c) 2013 Thomas Pelletier

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
