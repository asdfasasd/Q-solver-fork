package imageutil

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/disintegration/imaging"
)

// CompressForOCR 接收原始图片，返回压缩后的 JPEG 字节流
func CompressForOCR(originalImg image.Image, quality int, sharpen float64, Grayscale bool) ([]byte, error) {
	// 1. 获取原始尺寸
	bounds := originalImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 2. 调整大小 (Resize)
	// 策略：如果长边超过 2000px，就等比例缩小到 2000px
	// 2000px 对于绝大多数 OCR 场景已经足够清晰，且能显著减少 Token 消耗
	maxDimension := 2000
	var processedImg image.Image = originalImg

	if width > maxDimension || height > maxDimension {
		if width > height {
			// 宽是长边
			processedImg = imaging.Resize(originalImg, maxDimension, 0, imaging.Lanczos)
		} else {
			// 高是长边
			processedImg = imaging.Resize(originalImg, 0, maxDimension, imaging.Lanczos)
		}
	}

	// 3. 灰度化 (Grayscale)
	// 这一步能去掉颜色干扰，并减小文件体积（对于某些编码格式）
	if Grayscale {
		processedImg = imaging.Grayscale(processedImg)

	}
	// 4. 锐化 (Sharpen) - 可选
	// 稍微锐化一点点有助于 OCR 识别文字边缘，但不要过度
	if sharpen > 0 {
		processedImg = imaging.Sharpen(processedImg, sharpen)
	}

	// 5. 编码为 JPEG 并输出到内存
	// JPEG 质量范围 1-100。如果传入 0 或更小，强制设为 1
	if quality < 1 {
		quality = 1
	}
	if quality > 90 {
		quality = 90
	}

	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, processedImg, &jpeg.Options{
		Quality: quality, // 质量 80 是体积和清晰度的最佳平衡点
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
