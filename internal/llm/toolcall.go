package llm

import (
	"context"
	"fmt"

	"worldsim/internal/config"
)

// ToolLoopOptions 控制工具调用循环的行为。
type ToolLoopOptions struct {
	MaxRounds int // 最大工具调用轮数（默认 DefaultMaxToolRounds）
}

const DefaultMaxToolRounds = 4

// CallAPITools 执行一次带工具调用（function calling）的 chat 请求。
// 采用 OpenAI 兼容约定：请求带 tools → 模型返回 tool_calls → 执行工具，
// 把 assistant 消息（含 tool_calls）与 role:"tool" 结果回传 → 再请求，直到模型不再调用工具或达最大轮数。
// 底层统一走 chatOnceSync（非流式），保证 tool_calls 完整可解析。
// 若耗尽轮数模型仍想调工具，则做一次不带工具、强制依结果作答的收尾调用，保证总能返回正文。
func CallAPITools(ctx context.Context, apiCfg *config.APIConfig, system string, messages []Message, reg *ToolRegistry, opts *ToolLoopOptions) (string, error) {
	maxRounds := DefaultMaxToolRounds
	if opts != nil && opts.MaxRounds > 0 {
		maxRounds = opts.MaxRounds
	}
	msgs := make([]Message, 0, len(messages)+4)
	if system != "" {
		msgs = append(msgs, Message{Role: "system", Content: system})
	}
	msgs = append(msgs, messages...)

	exhausted := false
	for round := 0; round < maxRounds; round++ {
		chatResp, err := chatOnceSync(ctx, apiCfg, msgs, reg.Schemas())
		if err != nil {
			return "", err
		}
		if len(chatResp.Choices) == 0 {
			return "", fmt.Errorf("接口未响应有效 Choices 文本")
		}
		msg := chatResp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			// 模型未请求工具 → 正文即最终答案
			return msg.Content, nil
		}
		// 回传 assistant 消息（含 tool_calls），符合 OpenAI 约定
		msgs = append(msgs, Message{Role: "assistant", Content: msg.Content, ReasoningContent: msg.ReasoningContent, ToolCalls: msg.ToolCalls})
		// 依次执行工具，把结果作为 role:"tool" 消息回传（错误也回传，让模型自我修正）
		for _, tc := range msg.ToolCalls {
			result, execErr := reg.Execute(ctx, tc)
			if execErr != nil {
				result = "工具执行失败: " + execErr.Error()
			}
			msgs = append(msgs, Message{Role: "tool", ToolCallID: tc.ID, Content: result})
		}
		if round == maxRounds-1 {
			exhausted = true
		}
	}

	// 耗尽轮数仍想调工具：做一次不带工具的收尾调用，强制基于已收集的工具结果作答。
	if exhausted {
		forced := append(msgs, Message{Role: "user", Content: "请仅根据上面所有工具调用的返回结果，直接给出最终回答。不要再调用任何工具。"})
		chatResp, err := chatOnceSync(ctx, apiCfg, forced, nil)
		if err != nil {
			return "", err
		}
		if len(chatResp.Choices) > 0 {
			return chatResp.Choices[0].Message.Content, nil
		}
	}
	return "", fmt.Errorf("工具调用达到最大轮数 %d，未生成最终内容", maxRounds)
}
