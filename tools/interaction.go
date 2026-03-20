package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/go-vgo/robotgo"
)

type InteractionTool struct{}

func (t *InteractionTool) Name() string {
	return "interaction"
}

func (t *InteractionTool) Description() string {
	return "模拟鼠标和键盘操作，包括鼠标点击、输入文本、按下回车键，以及组合操作（点击-输入-回车）"
}

func (t *InteractionTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"description": "操作类型: click, type, enter, click_type_enter",
				"enum":        []string{"click", "type", "enter", "click_type_enter"},
			},
			"x": map[string]interface{}{
				"type":        "number",
				"description": "鼠标点击的x坐标（仅click和click_type_enter操作需要）",
				"minimum":     0,
			},
			"y": map[string]interface{}{
				"type":        "number",
				"description": "鼠标点击的y坐标（仅click和click_type_enter操作需要）",
				"minimum":     0,
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "要输入的文本（仅type和click_type_enter操作需要）",
			},
		},
		"required": []string{"type"},
	}
}

func (t *InteractionTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	opType, ok := params["type"].(string)
	if !ok {
		return "", fmt.Errorf("操作类型参数缺失或无效")
	}

	switch opType {
	case "click":
		return t.executeClick(params)
	case "type":
		return t.executeType(params)
	case "enter":
		return t.executeEnter()
	case "click_type_enter":
		return t.executeClickTypeEnter(params)
	default:
		return "", fmt.Errorf("不支持的操作类型: %s", opType)
	}
}

func (t *InteractionTool) executeClick(params map[string]interface{}) (string, error) {
	x, okX := params["x"].(float64)
	y, okY := params["y"].(float64)
	if !okX || !okY {
		return "", fmt.Errorf("鼠标点击操作需要x和y坐标参数")
	}

	// 1. 移动鼠标到目标坐标
	robotgo.Move(int(x), int(y))
	// 短暂延时，确保鼠标移动完成（避免点击位置偏差）
	time.Sleep(100 * time.Millisecond)

	// 2. 模拟左键单击（参数1：按键类型(left/right/middle)，参数2：是否长按）
	robotgo.Click("left", false)
	return fmt.Sprintf("成功在坐标(%d, %d)执行鼠标左键单击", int(x), int(y)), nil
}

func (t *InteractionTool) executeType(params map[string]interface{}) (string, error) {
	text, ok := params["text"].(string)
	if !ok {
		return "", fmt.Errorf("输入文本操作需要text参数")
	}

	// 短暂延时，确保文本框已激活（比如点击后等待焦点获取）
	time.Sleep(2 * time.Second)
	// 直接输入文本（robotgo已封装字符编码/按键组合逻辑）
	robotgo.TypeStr(text)
	return fmt.Sprintf("成功输入文本: %s", text), nil
}

func (t *InteractionTool) executeEnter() (string, error) {
	// 短暂延时，确保操作稳定性
	time.Sleep(100 * time.Millisecond)
	// 使用 KeyTap 模拟按下回车键
	robotgo.KeyTap("enter")
	return "成功按下回车键", nil
}

func (t *InteractionTool) executeClickTypeEnter(params map[string]interface{}) (string, error) {
	// 1. 执行鼠标点击
	x, okX := params["x"].(float64)
	y, okY := params["y"].(float64)
	if !okX || !okY {
		return "", fmt.Errorf("click_type_enter操作需要x和y坐标参数")
	}

	// 移动鼠标到目标坐标
	robotgo.Move(int(x), int(y))
	// 短暂延时，确保鼠标移动完成
	time.Sleep(100 * time.Millisecond)

	// 模拟左键单击
	robotgo.Click("left", false)

	// 2. 执行文本输入
	text, ok := params["text"].(string)
	if !ok {
		return "", fmt.Errorf("click_type_enter操作需要text参数")
	}

	// 短暂延时，确保文本框已激活
	time.Sleep(2 * time.Second)
	// 输入文本
	robotgo.TypeStr(text)

	// 3. 执行回车键
	// 短暂延时，确保输入完成
	time.Sleep(100 * time.Millisecond)
	// 按下回车键
	robotgo.KeyTap("enter")

	return fmt.Sprintf("成功执行点击-输入-回车操作: 在坐标(%d, %d)点击，输入文本 '%s' 并按下回车", int(x), int(y), text), nil
}
