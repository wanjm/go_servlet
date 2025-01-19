### 说明
本项目是用来自动产生servlet通用代码， 包括servlet注册，prpc的客户端代码，服务端注册代码，filter注册代码，已经swagger文件等；具体参见 https://infi.cn/panel/?id=66e0e2f19513a50ea9a7d8cf
## 开发说明
### 案例
已开发一个发送 application/json，并返回json的程序为例说明过程；
1. 向url /hello 发送请求
2. 请求为 {"name":"world"};
3. 返回结果为{"code":0,"msg":"","obj":{"greeting":"hello world"}}
### servlet 定义
1. 定义servlet服务类
```
// @goservlet servlet
type Hello struct {
}
```
2. servlet函数定义
```
// @goservlet url="/hello";
func (hello *Hello) SayHello(ctx context.Context, req *schema.HelloRequest) (res schema.HelloResponse, err basic.Error) {
	res.Greeting = "hello " + req.Name
	return
}
```
3. 参数说明

```
type HelloRequest struct {
	Name string `json:"name"`
}
type HelloResponse struct {
	Greeting string `json:"greeting"`
}
```
4. 运行go_servlet生成胶水代码，就可以完成该服务了

### 使用说明
1. 在运行go_gen目标工程目录； 会自动扫描文件，并在根目录生成gen目录
2. 使用-p指定目标工程目录，或者直接在工程列表中运行；
3. 使用-i目录，可以在一个空的工程目录，可能自动生成main文件；

###  

4. 完成全局变量的注入
5. 完成不存在的全局变量的注入方法：

# 2024-9-7
1. 完成initorator的初始化的函数调用，并复制给变量### 使用说明
1. 在运行go_gen目标工程目录； 会自动扫描文件，并在根目录生成gen目录
2. 使用-p指定目标工程目录，或者直接在工程列表中运行；
3. 使用-i目录，可以在一个空的工程目录，可能自动生成main文件；

###  

4. 完成全局变量的注入
5. 完成不存在的全局变量的注入方法：

