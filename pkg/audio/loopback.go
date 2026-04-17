package audio

import (
	"Q-Solver/pkg/common"
	"Q-Solver/pkg/logger"
	"errors"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
)

const (
	PacketDurationMs = 30                                                    // 每个数据包的时长（毫秒）
	SampleRate       = 16000                                                 // 采样率 16kHz
	BytesPerSample   = 2                                                     // S16 格式，每个样本 2 字节
	PacketSize       = SampleRate * BytesPerSample * PacketDurationMs / 1000 // 每包字节数
	ChannelCapacity  = 100                                                   // channel 容量（可存储的数据包数量）
	RingBufferSize   = PacketSize * 200                                      // 环形缓冲区大小（可存储 200 个数据包，约 8 秒）
)

var (
	ErrNoLoopbackDevice = errors.New("未找到可用的系统音频捕获设备，macOS 需要安装 BlackHole 或类似的虚拟音频驱动")
)

// LoopbackCapture 扬声器音频采集（使用环形缓冲区 + channel）
type LoopbackCapture struct {
	ctx     *malgo.AllocatedContext
	device  *malgo.Device
	mu      sync.Mutex
	running bool

	// 环形缓冲区：音频采集回调直接写入
	ringBuffer *common.RingBuffer

	// channel：消费者从此读取固定大小的音频包
	audioChan chan []byte

	// 停止信号
	stopChan chan struct{}
	wg       sync.WaitGroup

	// macOS 虚拟音频设备 ID（BlackHole 等）
	loopbackDeviceID *malgo.DeviceID
}

// NewLoopbackCapture 创建 Loopback 采集器
func NewLoopbackCapture(onData func([]byte)) (*LoopbackCapture, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	capture := &LoopbackCapture{
		ctx:        ctx,
		ringBuffer: common.NewRingBuffer(RingBufferSize),
		audioChan:  make(chan []byte, ChannelCapacity),
		stopChan:   make(chan struct{}),
	}

	// macOS 需要查找虚拟音频设备
	if runtime.GOOS == "darwin" {
		deviceID, err := capture.findLoopbackDevice()
		if err != nil {
			logger.Printf("警告: %v", err)
			// 不返回错误，允许创建但 Start 时会失败
		} else {
			capture.loopbackDeviceID = deviceID
		}
	}

	return capture, nil
}

// findLoopbackDevice 查找 macOS 上的虚拟音频设备（如 BlackHole）
func (c *LoopbackCapture) findLoopbackDevice() (*malgo.DeviceID, error) {
	// 获取所有捕获设备
	infos, err := c.ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, err
	}

	logger.Printf("可用的捕获设备列表 (%d 个):", len(infos))
	for i, info := range infos {
		logger.Printf("  [%d] %s", i, info.Name())
	}

	// 按优先级查找虚拟音频设备
	loopbackKeywords := []string{
		"blackhole",    // BlackHole (推荐)
		"soundflower",  // Soundflower
		"loopback",     // Loopback by Rogue Amoeba
		"virtual",      // 其他虚拟设备
		"multi-output", // 多输出设备
	}

	for _, keyword := range loopbackKeywords {
		for _, info := range infos {
			deviceName := strings.ToLower(info.Name())
			if strings.Contains(deviceName, keyword) {
				logger.Printf("找到虚拟音频设备: %s", info.Name())
				id := info.ID
				return &id, nil
			}
		}
	}

	return nil, ErrNoLoopbackDevice
}

// Start 开始采集扬声器输出
func (c *LoopbackCapture) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	var deviceConfig malgo.DeviceConfig

	if runtime.GOOS == "darwin" {
		// macOS: 使用虚拟音频设备作为捕获输入
		if c.loopbackDeviceID == nil {
			return ErrNoLoopbackDevice
		}
		deviceConfig = malgo.DefaultDeviceConfig(malgo.Capture)
		deviceConfig.Capture.DeviceID = c.loopbackDeviceID.Pointer()
		logger.Println("macOS: 使用虚拟音频设备进行系统音频捕获")
	} else {
		// Windows: 使用 WASAPI Loopback
		deviceConfig = malgo.DefaultDeviceConfig(malgo.Loopback)
		logger.Println("Windows: 使用 WASAPI Loopback 进行系统音频捕获")
	}

	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = SampleRate

	// 数据回调 - 直接写入环形缓冲区
	onRecv := func(_, pInput []byte, frameCount uint32) {
		if len(pInput) > 0 {
			_, _ = c.ringBuffer.Write(pInput)
		}
	}

	callbacks := malgo.DeviceCallbacks{Data: onRecv}

	device, err := malgo.InitDevice(c.ctx.Context, deviceConfig, callbacks)
	if err != nil {
		return err
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		return err
	}

	c.device = device
	c.running = true

	// 启动消费者协程：从环形缓冲区读取固定大小数据包并发送到 channel
	c.wg.Add(1)
	go c.packetizer()

	logger.Println("Loopback 采集已启动（环形缓冲区模式）")
	return nil
}

// packetizer 从环形缓冲区读取固定大小的数据包并发送到 channel
func (c *LoopbackCapture) packetizer() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Duration(PacketDurationMs) * time.Millisecond)
	defer ticker.Stop()

	packetBuffer := make([]byte, PacketSize)

	for {
		select {
		case <-c.stopChan:
			logger.Println("音频分包器已停止")
			return

		case <-ticker.C:
			// 尝试从环形缓冲区读取一个完整的数据包
			n, err := c.ringBuffer.Read(packetBuffer)
			if err == common.ErrNotEnoughData {
				// 缓冲区数据不足，跳过这次
				continue
			}
			if err != nil {
				logger.Printf("读取环形缓冲区失败: %v", err)
				continue
			}

			if n == PacketSize {
				// 创建副本发送到 channel
				packet := make([]byte, PacketSize)
				copy(packet, packetBuffer)

				select {
				case c.audioChan <- packet:
					// 成功发送
				default:
					// channel 满了，丢弃此包
				}
			}
		}
	}
}

// GetAudioChannel 获取音频数据 channel（供 Live API 消费）
func (c *LoopbackCapture) GetAudioChannel() <-chan []byte {
	return c.audioChan
}

// Stop 停止采集
func (c *LoopbackCapture) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	// 停止设备
	if c.device != nil {
		c.device.Stop()
		c.device.Uninit()
		c.device = nil
	}

	// 停止分包器协程
	close(c.stopChan)
	c.wg.Wait()

	// 重置环形缓冲区
	c.ringBuffer.Reset()

	// 清空 channel
	for len(c.audioChan) > 0 {
		<-c.audioChan
	}

	c.running = false
	logger.Println("Loopback 采集已停止")
}

// Close 释放资源
func (c *LoopbackCapture) Close() {
	c.Stop()

	// 关闭 channel
	close(c.audioChan)

	if c.ctx != nil {
		_ = c.ctx.Uninit()
		c.ctx.Free()
		c.ctx = nil
	}
}

// IsRunning 是否正在采集
func (c *LoopbackCapture) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// GetBufferStatus 获取缓冲区状态（用于监控）
func (c *LoopbackCapture) GetBufferStatus() (bufferSize int, channelSize int) {
	return c.ringBuffer.Len(), len(c.audioChan)
}

// HasLoopbackSupport 检查是否支持系统音频捕获
func (c *LoopbackCapture) HasLoopbackSupport() bool {
	if runtime.GOOS == "windows" {
		return true // Windows 原生支持
	}
	return c.loopbackDeviceID != nil // macOS 需要虚拟音频设备
}

// GetLoopbackDeviceName 获取当前使用的 loopback 设备名称
func (c *LoopbackCapture) GetLoopbackDeviceName() string {
	if runtime.GOOS == "windows" {
		return "WASAPI Loopback"
	}
	if c.loopbackDeviceID == nil {
		return ""
	}
	// 重新获取设备信息
	infos, err := c.ctx.Devices(malgo.Capture)
	if err != nil {
		return "Unknown"
	}
	for _, info := range infos {
		if info.ID == *c.loopbackDeviceID {
			return info.Name()
		}
	}
	return "Unknown"
}

// ListCaptureDevices 列出所有可用的捕获设备（用于调试）
func (c *LoopbackCapture) ListCaptureDevices() []string {
	infos, err := c.ctx.Devices(malgo.Capture)
	if err != nil {
		return nil
	}
	var names []string
	for _, info := range infos {
		names = append(names, info.Name())
	}
	return names
}
