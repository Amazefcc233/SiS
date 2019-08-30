package main

import (
	"fmt"
	"github.com/Tnze/CoolQ-Golang-SDK/cqp"
	"github.com/miaoscraft/SiS/data"
	"github.com/miaoscraft/SiS/syntax"
	"github.com/miaoscraft/SiS/whitelist"
	"runtime/debug"
)

//go:generate cqcfg -c .
// cqp: 名称: SiS
// cqp: 版本: 1.0.0:0
// cqp: 作者: Tnze
// cqp: 简介: Minecraft服务器综合管理器
func main() { /*空*/ }

func init() {
	cqp.AppID = "cn.miaoscraft.sis"
	cqp.Start = onStart
	cqp.Exit = onExit

	cqp.GroupMsg = onGroupMsg
	cqp.GroupMemberDecrease = onGroupMemberDecrease
}

// 酷Q启动事件
func onStart() int32 {
	defer panicer()

	// 连接数据源
	err := data.Init()
	if err != nil {
		cqp.AddLog(cqp.Error, "Init", fmt.Sprintf("初始化数据源失败: %v", err))
	}

	// 将登录账号载入命令解析器（用于识别@）
	syntax.CmdPrefix = fmt.Sprintf("[CQ:at,qq=%d]", cqp.GetLoginQQ())

	return 0
}

// 酷Q关闭事件
func onExit() int32 {
	err := data.Close()
	if err != nil {
		cqp.AddLog(cqp.Error, "Init", fmt.Sprintf("释放数据源失败: %v", err))
	}
	return 0
}

// 群消息事件
func onGroupMsg(subType, msgID int32, fromGroup, fromQQ int64, fromAnonymous, msg string, font int32) int32 {
	defer panicer()

	if fromQQ == 80000000 { // 忽略匿名
		return Ignore
	}

	ret := func(resp string) {
		cqp.SendGroupMsg(fromGroup, resp)
	}

	switch fromGroup {
	case data.Config.AdminID:
		// 当前版本，管理群和游戏群收到的命令不做区分
		fallthrough

	case data.Config.GroupID:
		if syntax.GroupMsg(fromQQ, msg, ret) {
			return Intercept
		}
	}
	return Ignore
}

// 群成员减少事件
func onGroupMemberDecrease(subType, sendTime int32, fromGroup, fromQQ, beingOperateQQ int64) int32 {
	defer panicer()

	retValue := Ignore
	ret := func(resp string) {
		cqp.SendGroupMsg(fromGroup, resp)
		retValue = Intercept
	}

	if fromGroup == data.Config.GroupID {
		whitelist.RemoveWhitelist(beingOperateQQ, ret)
	}
	return retValue
}

const (
	Ignore    int32 = 0 //忽略消息
	Intercept       = 1 //拦截消息
)

// 用于捕获所有panic，转换为酷Q的Fatal日志
func panicer() {
	if v := recover(); v != nil {
		// 在这里调用debug.Stack()获取调用栈
		cqp.AddLog(cqp.Fatal, "Main", fmt.Sprintf("%v\n%s", v, debug.Stack()))
	}
}
