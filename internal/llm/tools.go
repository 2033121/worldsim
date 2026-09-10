package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// ---------- OpenAI 兼容 function calling 数据结构 ----------

// Tool 是发给模型的工具声明（OpenAI tools 数组元素）。
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction 描述一个可调用函数及其参数 JSON Schema。
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// FunctionCall 是模型返回的工具调用中的函数部分。
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串，需二次 json.Unmarshal
}

// ToolCall 是模型返回的一条工具调用。
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// objSchema 便捷构造一个简单的 JSON 对象参数 Schema。
func objSchema(props map[string]any, required []string) json.RawMessage {
	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	b, _ := json.Marshal(schema)
	return b
}

// ---------- 工具注册表 ----------

// ToolExec 执行一个工具调用，返回结果文本（会作为 role:"tool" 消息回传给模型）。
type ToolExec func(ctx context.Context, args json.RawMessage) (string, error)

// RegisteredTool 绑定工具 Schema 与执行函数。
type RegisteredTool struct {
	Schema  Tool
	Execute ToolExec
}

// ToolRegistry 以名称注册工具，供工具调用循环按名分发执行。
type ToolRegistry struct {
	tools map[string]RegisteredTool
}

// NewToolRegistry 创建空注册表。
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: map[string]RegisteredTool{}}
}

// Register 注册一个工具。
func (r *ToolRegistry) Register(name string, schema Tool, fn ToolExec) {
	schema.Function.Name = name
	r.tools[name] = RegisteredTool{Schema: schema, Execute: fn}
}

// Has 判断某工具是否已注册。
func (r *ToolRegistry) Has(name string) bool {
	_, ok := r.tools[name]
	return ok
}

// Schemas 返回全部已注册工具的 Schema（按注册名排序，保证请求稳定）。
func (r *ToolRegistry) Schemas() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Schema)
	}
	return out
}

// Execute 执行一条工具调用，未知工具返回错误（错误也会作为 tool 消息回传，让模型自我修正）。
func (r *ToolRegistry) Execute(ctx context.Context, tc ToolCall) (string, error) {
	t, ok := r.tools[tc.Function.Name]
	if !ok {
		return "", fmt.Errorf("未知工具: %s", tc.Function.Name)
	}
	return t.Execute(ctx, json.RawMessage(tc.Function.Arguments))
}

// ---------- 内置工具 Schema ----------

// WebSearchArgs 是与 web_search 工具参数 Schema 对应的入参结构。
type WebSearchArgs struct {
	Query     string `json:"query"`
	MaxResult int    `json:"max_results,omitempty"`
	Language  string `json:"language,omitempty"` // 例如 zh-CN / en
}

// WebSearchToolSchema 返回 web_search 工具的 Schema 定义。
// 只声明形状，实际执行需在阶段2由 search 后端注册（见 RegisterWebSearchTool）。
func WebSearchToolSchema() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "web_search",
			Description: "搜索互联网获取实时信息。当模型需要查询当前事实、最新新闻、人物/事件背景、资料核实等时使用。返回按相关性排序的标题/链接/摘要。",
			Parameters: objSchema(map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "搜索关键词，尽量具体（如：'法国 2026 大选 最新结果'）",
				},
				"max_results": map[string]any{
					"type":        "integer",
					"description": "返回结果条数，默认5，最大10",
				},
				"language": map[string]any{
					"type":        "string",
					"description": "结果语言偏好，如 zh-CN / en，留空由引擎决定",
				},
			}, []string{"query"}),
		},
	}
}
