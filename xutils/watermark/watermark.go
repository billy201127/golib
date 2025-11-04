package watermark

import (
	"bytes"
	"context"
	"io"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/zeromicro/go-zero/core/logc"
	"golang.org/x/image/font/gofont/goregular"
)

func AddWatermark(ctx context.Context, imagePath string, watermarkText string) io.ReadCloser {
	const fontSize = 48

	// 加载原图
	im, err := gg.LoadImage(imagePath)
	if err != nil {
		logc.Errorf(ctx, "AddWatermark load image failed, err: %v", err)
		return nil
	}

	w := im.Bounds().Dx()
	h := im.Bounds().Dy()

	dc := gg.NewContextForImage(im)

	// 使用内置字体而不是系统字体文件
	// 这样可以避免依赖特定的系统字体路径
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		logc.Errorf(ctx, "AddWatermark parse font failed, err: %v", err)
		return nil
	}
	dc.SetFontFace(truetype.NewFace(font, &truetype.Options{Size: fontSize}))
	dc.SetRGBA(1, 1, 1, 0.15) // 白色 + 半透明

	// 旋转整个画布（比如斜45°）
	dc.RotateAbout(gg.Radians(-30), float64(w)/2, float64(h)/2)

	// 计算间距（文字间隔）
	textWidth, textHeight := dc.MeasureString(watermarkText)
	xStep := textWidth * 2
	yStep := textHeight * 3

	// 平铺整个画布
	for x := -w; x < 2*w; x += int(xStep) {
		for y := -h; y < 2*h; y += int(yStep) {
			dc.DrawStringAnchored(watermarkText, float64(x), float64(y), 0.5, 0.5)
		}
	}

	// 保存输出
	// dc.SavePNG("output.png")

	var output bytes.Buffer
	dc.EncodePNG(&output)
	return io.NopCloser(&output)
}
