//go:build !cgo
// +build !cgo

package watermark

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/zeromicro/go-zero/core/logc"
	"golang.org/x/image/font/gofont/goregular"
)

// smartDecode 解决 image.Decode 对 OSS URL 格式识别失败的问题
func smartDecode(r io.Reader, contentType string) (image.Image, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}
	buf := bytes.NewBuffer(data)

	// 1. 优先根据 Content-Type
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "jpeg") || strings.Contains(ct, "jpg") {
		img, err := jpeg.Decode(buf)
		return img, "jpeg", err
	}

	if strings.Contains(ct, "png") {
		img, err := png.Decode(buf)
		return img, "png", err
	}

	// 2. fallback 自动探测 PNG → JPEG
	img, err := png.Decode(bytes.NewBuffer(data))
	if err == nil {
		return img, "png", nil
	}

	img, err = jpeg.Decode(bytes.NewBuffer(data))
	if err == nil {
		return img, "jpeg", nil
	}

	return nil, "", err
}

func AddFromBytes(ctx context.Context, body []byte, text string) (io.ReadCloser, error) {
	smarIm, format, err := smartDecode(bytes.NewBuffer(body), "")
	if err != nil {
		logc.Errorf(ctx, "AddWatermark decode image failed, err: %v", err)
		return nil, err
	}
	return draw(ctx, smarIm, format, text)
}

func Add(ctx context.Context, uri string, watermarkText string) (io.ReadCloser, error) {
	const fontSize = 48

	var (
		im     image.Image
		format string
	)

	// ---------- 1. 加载图片 ----------
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		resp, err := http.Get(uri)
		if err != nil {
			logc.Errorf(ctx, "AddWatermark load http image failed, err: %v", err)
			return nil, err
		}
		defer resp.Body.Close()

		im, format, err = smartDecode(resp.Body, resp.Header.Get("Content-Type"))
		if err != nil {
			logc.Errorf(ctx, "AddWatermark decode http image failed, err: %v", err)
			return nil, err
		}

	} else {
		// 本地文件
		raw, err := gg.LoadImage(uri)
		if err != nil {
			logc.Errorf(ctx, "AddWatermark load local image failed, err: %v", err)
			return nil, err
		}
		im = raw

		if strings.HasSuffix(strings.ToLower(uri), ".png") {
			format = "png"
		} else {
			format = "jpeg"
		}
	}

	return draw(ctx, im, format, watermarkText)
}

func draw(ctx context.Context, im image.Image, format string, watermarkText string) (io.ReadCloser, error) {
	const fontSize = 48

	var (
		err error
	)

	// ---------- 2. 绘制水印 ----------
	w := im.Bounds().Dx()
	h := im.Bounds().Dy()
	dc := gg.NewContextForImage(im)

	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		logc.Errorf(ctx, "AddWatermark parse font failed, err: %v", err)
		return nil, err
	}

	dc.SetFontFace(truetype.NewFace(font, &truetype.Options{Size: fontSize}))
	dc.SetRGBA(1, 1, 1, 0.25)
	dc.RotateAbout(gg.Radians(-30), float64(w)/2, float64(h)/2)

	textWidth, textHeight := dc.MeasureString(watermarkText)
	xStep := textWidth * 2
	yStep := textHeight * 3

	for x := -w; x < 2*w; x += int(xStep) {
		for y := -h; y < 2*h; y += int(yStep) {
			dc.DrawStringAnchored(watermarkText, float64(x), float64(y), 0.5, 0.5)
		}
	}

	// ---------- 3. 保存 ----------
	var output bytes.Buffer

	switch format {
	case "png":
		err = png.Encode(&output, dc.Image())
	default:
		err = jpeg.Encode(&output, dc.Image(), &jpeg.Options{Quality: 95})
	}

	if err != nil {
		logc.Errorf(ctx, "AddWatermark encode image failed, err: %v", err)
		return nil, err
	}

	return io.NopCloser(&output), nil
}
