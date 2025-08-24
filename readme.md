
# content
1. [Go Web 开发提速：基于 Spring 式注释方案，用 gos 自动生成运行代码https://zhuanlan.zhihu.com/p/1937905040842004437](https://zhuanlan.zhihu.com/p/1937905040842004437)
2. [Go Web 开发提速(gos)：Servlet 注解与参数解析全指南 —— 从定义到落地](https://zhuanlan.zhihu.com/p/1937994788919019061)
3. [Go Web 开发提速 3（gos）：Filter 实战与变量注入 —— 通用逻辑复用与依赖解耦](https://zhuanlan.zhihu.com/p/1942992392115446822)
# 准备
1. 我需要设计一个扫描go语言代码，解析出其中关于servlet的定义部分，然后自动跟gin的对接代码，完成端口监听并根据url调用servlet的功能。
2. 当定义一个函数 如下格式时，自动生成/hello的servet代码；并将请求解析为schema.HelloRequest类型，返回schema.HelloResponse类型。
```go
type Hello struct {

}
// @goservlet url="/hello";
func (hello *Hello) SayHello(ctx context.Context, req *schema.HelloRequest) (res schema.HelloResponse, err basic.Error) {
```
# 设计
1. Project
2. Package
3. Struct
4. Interface
5. Function
6. Method
7. Field

## 代码
创建astinfo目录，用于存放设计代码

## Project
1. 代码存放存放在astinfo/project.go中
2. Project结构体用于存储项目的信息。
3. Project结构体包含以下字段：
    - Name: 项目名称
    - Module: 项目的module名称
    - Path: 项目在工程中根目录在工程中的绝对路径
    - Packages: 项目中包含的package map key为package的全路径；value为Package结构体

## Package
1. 代码存放存放在astinfo/package.go中
2. Package结构体用于存储package的信息。
3. Package结构体包含以下字段：
    - Name: package名称
    - Module : package的全路径
    - Structs: package中包含的struct map key为struct的名称；value为Struct结构体


## Struct
1. 代码存放存放在astinfo/struct.go中
2. Struct结构体用于存储struct的信息。
3. Struct结构体包含以下字段：
    - Name: struct名称
## functionManager
我们的代码函数分为
1. 初始化函数initiator，用于自动运行，生成一些对象共注入其他对象中；
2. creator函数，用于生成对象供注入，跟intiator的区别时，initiator是全局的，creator是每次注入时都生成；
3. servlet函数；
4. postAction函数；
所有的函数分别有package和struct对象进行管理；所以function有pkg管理，method由struct管理；
5. filter该有谁管理；


## 代码语法解析
```
// project.parse
for eachdir {
    // package.parse
    for eachfile {
        // gosource.parse
        switch type
            // struct
            // interface
            // variable
            // function
            // method
        
    }
}
```

## 对象管理说明
### Project:Package管理
1. package的产生有两种来源
2. 第一种是实实在在的解析这个包时生成； 
3. 第二种是在解析过程中需要使用某个package，但是该package还没有生成时，需要先生成对象，然后再使用。
4. 所以提供了FindPackage方法，用于查找package，不存在则生成。
5. 提供了GetPackage方法，用于获取package，不存在则返回nil。


## project的函数定义
1. 生成Porject::GetPackage(module) 返回*Package；
    - 该方法查找Project中的Packages map中是否存在指定module的package；
2. 生成Porject::FindPackage(module) 返回*Package；
    - 该方法查找Project中的Packages map中是否存在指定module的package；
    - 如果存在，则返回该package；
    - 如果不存在，则生成一个新的package，并将其添加到项目的Packages map中；
   - 并返回
3. 在Project结构体,生成Parse方法，用于解析项目中的代码。
    - 调用ParseMode方法；
    - 调用filePath的Walk方法，遍历项目中的所有文件；
    - 对于每个文件，当其是目录时，调用ParsePackage方法；
4. 在Project结构体,生成ParseMode方法
   - 解析项目的go.mod文件，获取项目的module名称，并保存到Project结构体的Module field中；
5. 在Project结构体,生成ParsePackage方法
   - 生成Package对象；
   - 对于当前目录，调用pacakge的Parse方法完成本pacakge的解析；

## Package的函数定义
1.  生成NewPackage(module) 返回*Package；


## 主函数定义
1. 创建main.go文件，用于接收-p参数，调用Project的Parse方法；
2. 函数能接受-p参数，表示当前工作目录，并传递给Project对象的path中；
3. 创建Project对象，接收Path值，调用起Parse方法；
4. 调用其GenCode方法；
    
## 规则说明
1. 所有的parse方法，都是先构建一个对象，然后再调用其Parse方法；
2. 每个对象中包含包含本设计的对象的关联关系，同时关联的语法树的入口对象；




