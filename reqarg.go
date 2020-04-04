/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/4/3
   Description :
-------------------------------------------------
*/

package zirisroute

// 请求参数
type ReqArg struct {
    // 请求方法
    reqMethod string
    // 执行的控制器名
    controlName string
    // 控制器执行的方法
    controlMethod string
    // 路径参数
    pathParams string
    // 是否停止
    stop bool
}

// 停止请求
func (m *ReqArg) Stop() {
    m.stop = true
}

// 返回是否调用了Stop()
func (m *ReqArg) IsStop() bool { return m.stop }

// 返回请求方法
func (m *ReqArg) ReqMethod() string { return m.reqMethod }

// 返回控制器名
func (m *ReqArg) ControlName() string { return m.controlName }

// 返回控制器方法
func (m *ReqArg) ControlMethod() string { return m.controlMethod }

// 返回路径参数
func (m *ReqArg) PathParams() string { return m.pathParams }

// 设置控制器执行的方法
func (m *ReqArg) SetControlMethod(method string) {
    m.controlMethod = method
}

// 设置路径参数
func (m *ReqArg) SetPathParams(params string) {
    m.pathParams = params
}
