package live

import (
	"Q-Solver/pkg/audio"
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/screen"
	"context"
	"encoding/base64"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 重连相关常量
const (
	maxReconnectAttempts = 3                // 最大重连尝试次数
	reconnectDelay       = time.Second      // 重连间隔
	connectTimeout       = 3 * time.Second // 连接超时时间
)

// SessionState 会话状态类型
type SessionState int32

// 会话状态常量
const (
	StateNormal       SessionState = 0 // 正常运行
	StateReconnecting SessionState = 1 // 正在重连
	StateStopped      SessionState = 2 // 已停止
)

// LiveSessionManager 管理 Live API 会话的完整生命周期
type LiveSessionManager struct {
	// 依赖注入
	ctx           context.Context
	llmService    *llm.Service
	configManager *config.ConfigManager
	screenService *screen.Service
	emitEvent     func(string, ...any)

	// Live Session 状态 (使用 atomic.Pointer 实现无锁访问)
	session atomic.Pointer[llm.LiveSession]

	// 音频采集
	audioCapture *audio.LoopbackCapture
	mu           sync.Mutex

	// 问题导图处理器
	graph *Graph

	// 当前轮次的问题和回答（用于累积后推送给 Graph）
	currentQuestion strings.Builder
	currentAnswer   strings.Builder
	roundMu         sync.Mutex

	// 重连状态机
	state       atomic.Int32 // 当前状态 (SessionState)
	reconnectMu sync.Mutex   // 保证只有一个协程执行重连

	// 协程管理 - 使用 context 控制生命周期
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
	errorChan  chan error
	wg         sync.WaitGroup
}

// NewLiveSessionManager 创建 Live Session 管理器
func NewLiveSessionManager(
	ctx context.Context,
	llmService *llm.Service,
	configManager *config.ConfigManager,
	screenService *screen.Service,
	emitEvent func(string, ...any),
) *LiveSessionManager {
	return &LiveSessionManager{
		ctx:           ctx,
		llmService:    llmService,
		configManager: configManager,
		screenService: screenService,
		emitEvent:     emitEvent,
	}
}

// Start 启动 Live API 会话
func (m *LiveSessionManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg := m.configManager.Get()

	// 检查 Provider 是否支持 Live
	provider := m.llmService.GetProvider()
	liveProvider, ok := provider.(llm.LiveProvider)
	if !ok {
		return &liveError{"当前模型不支持 Live API"}
	}

	m.emitEvent("live:status", "connecting")

	// 连接 Live Session
	liveCfg := llm.GetLiveConfig(cfg)
	session, err := liveProvider.ConnectLive(m.ctx, liveCfg)
	if err != nil {
		logger.Println("[Start] liveApi连接服务器失败", err)
		m.emitEvent("live:status", "error")
		return err
	}

	m.emitEvent("live:status", "connected")

	// 保存 session (atomic)
	m.session.Store(&session)
	m.state.Store(int32(StateNormal))

	// 初始化音频采集
	m.audioCapture, err = audio.NewLoopbackCapture(nil)
	if err != nil {
		logger.Printf("[Start] 音频采集初始化失败: %v", err)
		session.Close()
		m.session.Store(nil)
		m.emitEvent("live:status", "error")
		return &liveError{"音频采集初始化失败: " + err.Error()}
	}

	if err := m.audioCapture.Start(); err != nil {
		logger.Printf("[Start] 音频采集启动失败: %v", err)
		m.audioCapture.Close()
		m.audioCapture = nil
		session.Close()
		m.session.Store(nil)
		m.emitEvent("live:status", "error")
		return &liveError{"音频采集启动失败: " + err.Error()}
	}

	// 初始化 context 和 errorChan
	m.cancelCtx, m.cancelFunc = context.WithCancel(m.ctx)
	m.errorChan = make(chan error, 4)

	// 初始化并启动问题导图处理器（每3轮触发一次总结，使用 cancelCtx 统一控制）
	m.graph = NewGraph(m.cancelCtx, m.configManager, m.llmService, m.emitEvent, 3)
	m.graph.Start()

	// 启动错误监听协程
	m.wg.Add(1)
	go m.errorWatcher()

	// 启动音频发送协程 (不再传入 session，而是动态获取)
	m.wg.Add(1)
	go m.audioSender(m.audioCapture.GetAudioChannel())

	// 启动接收协程
	m.wg.Add(1)
	go m.receiveLoop()

	return nil
}

// Stop 停止 Live API 会话（外部调用）
func (m *LiveSessionManager) Stop() {
	// 设置停止状态
	m.state.Store(int32(StateStopped))

	// 通过 context 取消通知所有协程退出
	m.mu.Lock()
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	m.mu.Unlock()

	// 等待协程结束
	m.wg.Wait()

	// 停止问题导图处理器
	if m.graph != nil {
		m.graph.Stop()
		m.graph = nil
	}

	// 清空当前轮次缓存
	m.roundMu.Lock()
	m.currentQuestion.Reset()
	m.currentAnswer.Reset()
	m.roundMu.Unlock()

	// 执行清理
	m.cleanup()

	m.emitEvent("live:status", "disconnected")
}

// cleanup 内部清理方法
func (m *LiveSessionManager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.Println("[cleanup] Live: cleanup 调用")

	// 停止音频采集
	if m.audioCapture != nil {
		logger.Println("[cleanup] Live: 停止音频采集")
		m.audioCapture.Close()
		m.audioCapture = nil
	}

	// 关闭会话
	if session := m.session.Load(); session != nil {
		logger.Println("[cleanup] Live: 关闭会话")
		(*session).Close()
		m.session.Store(nil)
	}
}

// errorWatcher 监听错误通道，统一处理所有不可恢复的错误
// 这是唯一的错误处理入口，所有运行时错误都通过 errorChan 发送到这里
func (m *LiveSessionManager) errorWatcher() {
	defer m.wg.Done()

	for {
		select {
		case err := <-m.errorChan:
			logger.Printf("[errorWatcher] Live: errorWatcher 收到错误: %v", err)

			// 检查是否已经停止，避免重复处理
			if m.state.Load() == int32(StateStopped) {
				logger.Println("[errorWatcher] Live: 已停止，忽略错误")
				continue
			}

			// 统一错误处理：设置停止状态 -> 取消 context -> 清理资源 -> 通知前端
			m.state.Store(int32(StateStopped))
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			m.cleanup()
			m.emitEvent("live:status", "error")
			m.emitEvent("live:error", err.Error())
			return

		case <-m.cancelCtx.Done():
			logger.Println("[errorWatcher] Live: errorWatcher 正常退出")
			return
		}
	}
}

// IsActive 检查 Live Session 是否活跃
func (m *LiveSessionManager) IsActive() bool {
	return m.session.Load() != nil && m.state.Load() != int32(StateStopped)
}

// audioSender 从音频 channel 读取数据并发送给 Live Session
func (m *LiveSessionManager) audioSender(audioChan <-chan []byte) {
	defer m.wg.Done()

	logger.Println("[audioSender] Live: 音频发送协程已启动")
	failCount := 0

	for {
		select {
		case <-m.cancelCtx.Done():
			logger.Println("[audioSender] Live: 音频发送协程收到停止信号")
			return

		case audioData, ok := <-audioChan:
			if !ok {
				logger.Println("[audioSender] Live: 音频通道已关闭")
				return
			}

			// 如果正在重连，等待重连完成
			for m.state.Load() == int32(StateReconnecting) {
				failCount = 0 // 重连时重置失败计数
				select {
				case <-m.cancelCtx.Done():
					return
				case <-time.After(10 * time.Millisecond):
					// 短暂等待后重试
				}
			}

			// 获取当前 session (无锁)
			session := m.session.Load()
			if session == nil {
				continue
			}

			if err := (*session).SendAudio(audioData); err != nil {
				failCount++
				logger.Printf("[audioSender] Live: 发送音频失败 (%d/3): %v", failCount, err)

				if failCount >= 3 {
					logger.Println("[audioSender] Live: 连续 3 次发送失败，触发重连")
					// 主动触发重连（如果还没有的话）
					go m.handleGoAway(*session)
					failCount = 0
				}
				// 继续循环等待重连完成或重试
				continue
			}

			// 发送成功，重置计数
			if failCount > 0 {
				failCount = 0
			}
		}
	}
}

// receiveLoop 接收 Live 消息的循环
func (m *LiveSessionManager) receiveLoop() {
	defer m.wg.Done()

	logger.Println("[receiveLoop] Live: 接收循环已启动")

	for {
		// 检查是否已取消
		select {
		case <-m.cancelCtx.Done():
			logger.Println("[receiveLoop] Live: 接收循环收到停止信号")
			return
		default:
		}

		session := m.session.Load()
		if session == nil {
			// 可能在重连中，等待
			select {
			case <-m.cancelCtx.Done():
				return
			case <-time.After(50 * time.Millisecond):
				continue
			}
		}

		msg, err := (*session).Receive()
		if err != nil {
			// 检查是否已取消
			select {
			case <-m.cancelCtx.Done():
				return
			default:
			}

			// 检查是否正在重连
			if m.state.Load() == int32(StateReconnecting) {
				// 正在重连，忽略这个错误
				continue
			}

			// 如果已经停止则直接退出，避免后续重连日志
			if m.state.Load() == int32(StateStopped) {
				return
			}

			logger.Printf("[receiveLoop] Live: 接收错误: %v", err)

			// 连接断开（可能没有 GoAway），尝试重连
			// 不直接报错，而是触发重连机制
			logger.Println("[receiveLoop] Live: 连接断开，尝试重连...")
			m.handleGoAway(*session)
			continue
		}
		if msg == nil {
			continue
		}

		switch msg.Type {
		case llm.LiveMsgGoAway:
			// 检查是否已取消
			select {
			case <-m.cancelCtx.Done():
				return
			default:
			}
			// 收到 goaway，触发重连
			logger.Println("[receiveLoop] Live: 收到 GoAway，开始重连")
			m.handleGoAway(*session)
			continue

		case llm.LiveInterrupted:
			logger.Println("[receiveLoop] 检测到打断")
			m.emitEvent("live:Interrupted", msg.Text)
			// 打断时，清空当前轮次缓存
			m.roundMu.Lock()
			m.currentQuestion.Reset()
			m.currentAnswer.Reset()
			m.roundMu.Unlock()

		case llm.LiveMsgTranscript:
			m.emitEvent("live:transcript", msg.Text)
			// 累积问题文本
			m.roundMu.Lock()
			m.currentQuestion.WriteString(msg.Text)
			m.roundMu.Unlock()

		case llm.LiveMsgInterviewerDone:
			logger.Println("[receiveLoop] Live: 面试官说话结束")
			m.emitEvent("live:interviewer-done")

		case llm.LiveMsgAIText:
			m.emitEvent("live:ai-text", msg.Text)
			// 累积回答文本
			m.roundMu.Lock()
			m.currentAnswer.WriteString(msg.Text)
			m.roundMu.Unlock()

		case llm.LiveMsgToolCall:
			logger.Printf("[receiveLoop] Live: 工具调用 %s (ID=%s)", msg.ToolName, msg.ToolID)
			if msg.ToolName == "get_screenshot" {
				m.handleScreenshot(*session, msg.ToolID)
			}

		case llm.LiveMsgDone:
			logger.Println("[receiveLoop] Live: 对话轮完成")
			m.emitEvent("live:done")
			// 一轮对话完成，推送给 Graph
			m.roundMu.Lock()
			question := m.currentQuestion.String()
			answer := m.currentAnswer.String()
			m.currentQuestion.Reset()
			m.currentAnswer.Reset()
			m.roundMu.Unlock()

			if m.graph != nil && question != "" && answer != "" {
				m.graph.Push(question, answer)
			}

		case llm.LiveMsgError:
			// 服务端返回的错误，发送到 errorChan 统一处理
			logger.Printf("[receiveLoop] Live: 服务端错误: %s", msg.Text)
			m.errorChan <- &liveError{msg.Text}
			return
		}
	}
}

// handleGoAway 处理 goaway 重连
func (m *LiveSessionManager) handleGoAway(oldSession llm.LiveSession) {
	// 使用 TryLock 确保只有一个协程执行重连
	if !m.reconnectMu.TryLock() {
		// 已有其他协程在重连，直接返回
		return
	}
	defer m.reconnectMu.Unlock()

	// 检查是否已取消
	select {
	case <-m.cancelCtx.Done():
		return
	default:
	}

	// 设置重连状态
	m.state.Store(int32(StateReconnecting))

	m.emitEvent("live:status", "reconnecting")

	// 获取恢复令牌
	var resumeToken string
	if rs, ok := oldSession.(llm.ResumableSession); ok && rs.IsResumable() {
		resumeToken = rs.GetResumeToken()
		logger.Printf("[handleGoAway] Live: 使用 session handle 恢复会话")
	}

	// 关闭旧会话
	oldSession.Close()

	// 获取 provider
	provider := m.llmService.GetProvider()
	liveProvider, ok := provider.(llm.LiveProvider)
	if !ok {
		m.errorChan <- &liveError{"Provider 不支持 Live API"}
		return
	}

	// 重连尝试
	var newSession llm.LiveSession
	var err error

	for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
		// 每次循环都检查是否已取消
		select {
		case <-m.cancelCtx.Done():
			logger.Println("Live: 重连被取消")
			return
		default:
		}

		logger.Printf("[handleGoAway] Live: 重连尝试 %d/%d", attempt, maxReconnectAttempts)

		cfg := m.configManager.Get()
		liveCfg := llm.GetLiveConfig(cfg)
		liveCfg.ResumeToken = resumeToken // 使用恢复令牌

		newSession, err = liveProvider.ConnectLive(m.ctx, liveCfg)
		if err == nil {
			break
		}

		logger.Printf("[handleGoAway] Live: 重连失败: %v", err)
		if attempt < maxReconnectAttempts {
			// 使用 select 等待，这样可以响应停止信号
			select {
			case <-m.cancelCtx.Done():
				logger.Println("[handleGoAway] Live: 重连被取消（收到停止信号）")
				return
			case <-time.After(reconnectDelay):
				// 继续下一次尝试
			}
		}
	}

	// 最后再检查一次是否已取消
	select {
	case <-m.cancelCtx.Done():
		logger.Println("[handleGoAway] Live: 重连完成前被取消")
		if newSession != nil {
			newSession.Close()
		}
		return
	default:
	}

	if err != nil {
		logger.Printf("[handleGoAway] Live: 重连失败，已达最大尝试次数")
		// 发送错误到 errorChan，由 errorWatcher 统一处理
		m.errorChan <- err
		return
	}

	// 替换 session
	m.session.Store(&newSession)
	m.state.Store(int32(StateNormal))

	logger.Println("[handleGoAway] Live: 重连成功")
	m.emitEvent("live:status", "connected")
}

// liveError 统一的 Live API 错误类型
type liveError struct {
	msg string
}

func (e *liveError) Error() string {
	return e.msg
}

// handleScreenshot 处理 Live API 的截图请求
func (m *LiveSessionManager) handleScreenshot(session llm.LiveSession, toolID string) {
	cfg := m.configManager.Get()

	preview, err := m.screenService.CapturePreview(
		cfg.CompressionQuality,
		cfg.Sharpening,
		cfg.Grayscale,
		cfg.NoCompression,
		cfg.ScreenshotMode,
	)
	if err != nil {
		logger.Printf("[handleScreenshot] Live 截图失败: %v", err)
		_ = session.SendToolResponse(toolID, "截图失败: "+err.Error())
		return
	}

	// 解析 data URL 格式: data:image/jpeg;base64,xxxxx
	base64Str := preview.Base64
	mimeType := "image/jpeg" // 默认

	if strings.HasPrefix(base64Str, "data:") {
		if idx := strings.Index(base64Str, ";base64,"); idx > 5 {
			mimeType = base64Str[5:idx]
			base64Str = base64Str[idx+8:]
		}
	}

	imageData, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		logger.Printf("[handleScreenshot] Live Base64解码失败: %v", err)
		_ = session.SendToolResponse(toolID, "图片解码失败")
		return
	}

	err = session.SendToolResponseWithImage(toolID, imageData, mimeType)
	if err != nil {
		logger.Printf("[handleScreenshot] Live 发送截图失败: %v", err)
	} else {
		logger.Printf("[handleScreenshot] Live: 已发送屏幕截图给模型 (%d bytes, %s)", len(imageData), mimeType)
	}
}
