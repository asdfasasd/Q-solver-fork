package llm

import (
	"Q-Solver/pkg/config"
	"context"
)

// ==================== Live API Types ====================

// LiveMessageType 实时消息类型
type LiveMessageType string

const (
	LiveMsgTranscript      LiveMessageType = "transcript"       // 面试官语音转录
	LiveMsgInterviewerDone LiveMessageType = "interviewer_done" // 面试官说话结束
	LiveMsgAIText          LiveMessageType = "ai_text"          // AI 文本回复
	LiveMsgToolCall        LiveMessageType = "tool_call"        // 工具调用请求
	LiveMsgDone            LiveMessageType = "done"             // 对话轮完成
	LiveMsgError           LiveMessageType = "error"            // 错误
	LiveInterrupted        LiveMessageType = "interrupted"      // 打断
	LiveMsgGoAway          LiveMessageType = "goaway"           // 服务器要求断开，需重连
)

// LiveMessage 实时消息
type LiveMessage struct {
	Type     LiveMessageType `json:"type"`
	Text     string          `json:"text,omitempty"`
	ToolName string          `json:"toolName,omitempty"` // 工具名称 (如 get_screenshot)
	ToolID   string          `json:"toolId,omitempty"`   // 工具调用 ID
}

// LiveConfig 实时会话配置
type LiveConfig struct {
	Model             string
	SystemInstruction string
	// 模型参数
	MaxTokens   int
	Temperature float64
	TopP        float64
	TopK        int
	// 会话恢复令牌（用于 goaway 后重连）
	ResumeToken string
}

// LiveSession 实时会话接口
type LiveSession interface {
	SendAudio(data []byte) error
	Receive() (*LiveMessage, error)
	SendToolResponse(toolID string, result string) error
	SendToolResponseWithImage(toolID string, imageData []byte, mimeType string) error
	Close() error
}

// ResumableSession 支持会话恢复的扩展接口（可选实现）
// 不同厂商可能有不同的会话恢复机制，实现此接口即可支持 goaway 后的无缝重连
type ResumableSession interface {
	LiveSession

	// GetResumeToken 获取用于恢复会话的令牌
	// Gemini: session handle
	// OpenAI: 可能是 conversation_id 或其他形式
	GetResumeToken() string

	// IsResumable 当前会话是否支持恢复
	IsResumable() bool
}

// LiveProvider 支持实时对话的 Provider 接口
type LiveProvider interface {
	// ConnectLive 建立新连接
	// 如果 cfg.ResumeToken 不为空，尝试恢复之前的会话
	ConnectLive(ctx context.Context, cfg *LiveConfig) (LiveSession, error)
}

// GetLiveConfig 从配置创建 LiveConfig
func GetLiveConfig(cfg config.Config) *LiveConfig {
	return &LiveConfig{
		Model:             cfg.Model,
		SystemInstruction: cfg.Prompt,
		MaxTokens:         cfg.MaxTokens,
		Temperature:       cfg.Temperature,
		TopP:              cfg.TopP,
		TopK:              cfg.TopK,
	}
}
