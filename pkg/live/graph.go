package live

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/prompts"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ChatRound 一轮对话
type ChatRound struct {
	Question string
	Answer   string
}

// GraphNode 导图节点（发送给前端）
type GraphNode struct {
	ID       string `json:"id"`
	PID      string `json:"pid,omitempty"` // 父节点ID
	Title    string `json:"title"`         // 简短标题
	Question string `json:"question"`      // 完整问题
	Answer   string `json:"answer"`        // 完整回答
}

// Graph 问题导图处理器
type Graph struct {
	cancelCtx     context.Context
	configManager *config.ConfigManager
	llmService    *llm.Service
	emitEvent     func(string, ...any)

	// channel 接收对话，无需加锁
	roundChan chan ChatRound

	// 配置
	triggerRound int // 每多少轮触发一次总结

	// 历史存储（只在消费协程中访问，无需加锁）
	allRounds []ChatRound // 所有对话历史
	nodes     []GraphNode // 已生成的节点
}

// NewGraph 创建问题导图处理器
func NewGraph(
	cancelCtx context.Context,
	configManager *config.ConfigManager,
	llmService *llm.Service,
	emitEvent func(string, ...any),
	triggerRound int,
) *Graph {
	if triggerRound <= 0 {
		triggerRound = 3
	}
	return &Graph{
		cancelCtx:     cancelCtx,
		configManager: configManager,
		llmService:    llmService,
		emitEvent:     emitEvent,
		roundChan:     make(chan ChatRound, 100),
		triggerRound:  triggerRound,
		allRounds:     make([]ChatRound, 0),
		nodes:         make([]GraphNode, 0),
	}
}

// Start 启动消费协程
func (g *Graph) Start() {
	go g.consumeLoop()
	logger.Println("Graph: 已启动")
}

// Stop 停止处理（实际由 cancelCtx 控制）
func (g *Graph) Stop() {
	logger.Println("[Graph] Graph: Stop 调用")
}

// Push 推送一轮对话（问题+回答）
func (g *Graph) Push(question, answer string) {
	select {
	case g.roundChan <- ChatRound{Question: question, Answer: answer}:
		logger.Printf("Graph: 推送对话成功")
	default:
		logger.Println("Graph: channel 已满，消息被丢弃")
	}
}

// Clear 清空导图
func (g *Graph) Clear() {
	// 发送清空事件，实际清空在消费协程中处理
	g.emitEvent("graph:clear", nil)
}

// consumeLoop 消费循环
func (g *Graph) consumeLoop() {
	pendingRounds := make([]ChatRound, 0, g.triggerRound)

	for {
		select {
		case <-g.cancelCtx.Done():
			logger.Println("[Graph] Graph: 收到停止信号，退出消费循环")
			return

		case round := <-g.roundChan:
			// 存储到历史
			g.allRounds = append(g.allRounds, round)
			// 添加到待处理
			pendingRounds = append(pendingRounds, round)

			logger.Printf("Graph: 收到第 %d 轮对话，待处理: %d", len(g.allRounds), len(pendingRounds))

			// 检查是否达到触发轮数
			if len(pendingRounds) >= g.triggerRound {
				// 复制待处理的对话
				toProcess := make([]ChatRound, len(pendingRounds))
				copy(toProcess, pendingRounds)
				// 清空待处理
				pendingRounds = make([]ChatRound, 0, g.triggerRound)
				g.summarize(toProcess)
			}
		}
	}
}

// summarize 调用辅助模型总结对话
func (g *Graph) summarize(rounds []ChatRound) {
	cfg := g.configManager.Get()

	// 检查是否配置了辅助模型
	if cfg.AssistantModel == "" {
		logger.Println("Graph: 未配置辅助模型，跳过总结")
		return
	}

	logger.Printf("Graph: 开始总结 %d 轮对话", len(rounds))

	// 构建 prompt（包含已有节点信息）
	prompt := g.buildPrompt(rounds)
	logger.Println("生成导图的prompt: ", prompt)
	// 调用模型
	ctx, cancel := context.WithTimeout(g.cancelCtx, 60*time.Second)
	defer cancel()

	provider := g.llmService.GetProvider()
	response, err := provider.GenerateContent(ctx, cfg.AssistantModel, []llm.Message{
		llm.NewUserMessage(prompt),
	})
	if err != nil {
		logger.Printf("Graph: 总结失败: %v", err)
		return
	}
	logger.Println("导图总结回复 %s", response.Content)
	// 解析并添加节点
	newNodes := g.parseResponse(response.Content, rounds)
	for _, node := range newNodes {
		g.nodes = append(g.nodes, node)
		g.emitEvent("graph:add-node", node)
		logger.Printf("Graph: 添加节点: %s", node.Title)
	}
}

// buildPrompt 构建提示词
func (g *Graph) buildPrompt(rounds []ChatRound) string {
	var nodesSb strings.Builder
	var dialogSb strings.Builder

	// 构建已有节点信息
	if len(g.nodes) > 0 {
		for _, node := range g.nodes {
			nodesSb.WriteString(fmt.Sprintf("- NodeID: %s NodeTitle: %s NodeAnswer: %s \n", node.ID, node.Title, node.Answer))
		}
	} else {
		nodesSb.WriteString("（暂无已有节点）\n")
	}

	// 构建对话内容
	for i, round := range rounds {
		dialogSb.WriteString(fmt.Sprintf("第%d轮：\n问：%s\n答：%s\n\n", i+1, round.Question, round.Answer))
	}

	return fmt.Sprintf(prompts.GraphSummarizePromptTemplate, nodesSb.String(), dialogSb.String())
}

// parseResponse 解析模型响应
func (g *Graph) parseResponse(response string, rounds []ChatRound) []GraphNode {
	response = strings.TrimSpace(response)

	// 移除 markdown 代码块
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var results []struct {
		Title    string  `json:"title"`
		Question string  `json:"question"`
		Answer   string  `json:"answer"`
		PID      *string `json:"pid,omitempty"`
	}

	if err := json.Unmarshal([]byte(response), &results); err != nil {
		logger.Printf("Graph: 解析JSON失败: %v", err)
		// 解析失败，使用简单模式
		return g.createSimpleNodes(rounds)
	}

	// 获取最后一个节点ID作为默认父节点
	var lastNodeID string
	if len(g.nodes) > 0 {
		lastNodeID = g.nodes[len(g.nodes)-1].ID
	}

	nodes := make([]GraphNode, 0, len(results))
	for _, r := range results {
		nodeID := fmt.Sprintf("node-%d", time.Now().UnixNano())

		// 确定父节点：AI指定的 > 默认最后一个
		pid := lastNodeID
		if r.PID != nil && *r.PID != "" {
			// 验证父节点是否存在
			for _, n := range g.nodes {
				if n.ID == *r.PID {
					pid = *r.PID
					break
				}
			}
		}

		nodes = append(nodes, GraphNode{
			ID:       nodeID,
			PID:      pid,
			Title:    r.Title,
			Question: r.Question,
			Answer:   r.Answer,
		})
		lastNodeID = nodeID // 后续节点挂到这个节点下
	}

	return nodes
}

// createSimpleNodes 简单模式生成节点
func (g *Graph) createSimpleNodes(rounds []ChatRound) []GraphNode {
	var lastNodeID string
	if len(g.nodes) > 0 {
		lastNodeID = g.nodes[len(g.nodes)-1].ID
	}

	nodes := make([]GraphNode, 0, len(rounds))
	for _, round := range rounds {
		// 截取标题
		title := round.Question
		runes := []rune(title)
		if len(runes) > 15 {
			title = string(runes[:15]) + "..."
		}

		nodeID := fmt.Sprintf("node-%d", time.Now().UnixNano())
		nodes = append(nodes, GraphNode{
			ID:       nodeID,
			PID:      lastNodeID,
			Title:    title,
			Question: round.Question,
			Answer:   round.Answer,
		})
		lastNodeID = nodeID
	}

	return nodes
}

// GetAllRounds 获取所有对话历史
func (g *Graph) GetAllRounds() []ChatRound {
	return g.allRounds
}

// GetNodes 获取所有节点
func (g *Graph) GetNodes() []GraphNode {
	return g.nodes
}
