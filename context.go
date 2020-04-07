/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/4/3
   Description :
-------------------------------------------------
*/

package zirisroute

import (
    "github.com/kataras/iris/v12"
)

// 上下文
type Context interface {
    // 初始化, 创建上下文后会立即调用这个方法
    Init(irisctx iris.Context, reqArg *ReqArg)
    // 返回路径参数, 必须实现这个方法给使用者调用
    GetPathParams() string
    // 处理开始, 在所有中间件处理完毕后, 在进入处理函数之前调用这个方法
    Before(reqArg *ReqArg)
    // 请求处理完毕后会调用这个方法, 如果请求处理函数没有返回值会传入nil
    SetResult(a interface{})
}
