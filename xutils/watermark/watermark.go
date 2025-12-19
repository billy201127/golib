//go:build cgo
// +build cgo

package watermark

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/disintegration/imaging"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/zeromicro/go-zero/core/logc"
	"golang.org/x/image/font/gofont/goregular"
)

type Config struct {
	ImageBody         []byte
	InputPath         string
	WatermarkText     string
	MaxWidth          int
	Quality           int
	TileSpacingFactor float64
	MinTileStep       int
	Alpha             int
}

var (
	fontCache     *truetype.Font
	fontCacheOnce sync.Once
	httpClient    = &http.Client{Timeout: 15 * time.Second}
	vipsInitOnce  sync.Once
	wmLRU         = newWatermarkLRU(128)
)

func AddFromBytes(ctx context.Context, body []byte, text string) (io.ReadCloser, error) {
	cfg := Config{
		ImageBody:         body,
		WatermarkText:     text,
		MaxWidth:          2000,
		Quality:           85,
		TileSpacingFactor: 1.4,
		MinTileStep:       140,
		Alpha:             50,
	}

	outputBytes, err := applyWatermark(cfg)
	if err != nil {
		logc.Errorf(ctx, "applyWatermark error: %v", err)
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(outputBytes)), nil
}

func Add(ctx context.Context, path string, text string) (io.ReadCloser, error) {
	cfg := Config{
		InputPath:         path,
		WatermarkText:     text,
		MaxWidth:          2000,
		Quality:           85,
		TileSpacingFactor: 1.4,
		MinTileStep:       140,
		Alpha:             50,
	}

	outputBytes, err := applyWatermark(cfg)
	if err != nil {
		logc.Errorf(ctx, "applyWatermark error: %v", err)
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(outputBytes)), nil
}

func applyWatermark(cfg Config) ([]byte, error) {
	initVIPS()

	baseRef, err := loadBaseImage(cfg)
	if err != nil {
		return nil, err
	}
	defer baseRef.Close()

	_ = baseRef.AutoRotate()

	if cfg.MaxWidth > 0 && baseRef.Width() > cfg.MaxWidth {
		scale := float64(cfg.MaxWidth) / float64(baseRef.Width())
		if err := baseRef.Resize(scale, vips.KernelAuto); err != nil {
			return nil, fmt.Errorf("resize error: %w", err)
		}
	}

	if err := ensureRGBA(baseRef); err != nil {
		return nil, fmt.Errorf("ensureRGBA error: %w", err)
	}

	fontSize := determineFontSize(baseRef, cfg)

	watermarkPNG, err := createTextWatermarkPNG(cfg.WatermarkText, cfg.Alpha, fontSize)
	if err != nil {
		return nil, fmt.Errorf("createTextWatermarkPNG error: %w", err)
	}

	wmRef, err := vips.NewImageFromBuffer(watermarkPNG)
	if err != nil {
		return nil, fmt.Errorf("newImageFromBuffer error: %w", err)
	}
	defer wmRef.Close()

	if err := ensureRGBA(wmRef); err != nil {
		return nil, fmt.Errorf("ensureRGBA error: %w", err)
	}

	if wmRef.Interpretation() != baseRef.Interpretation() {
		if err := wmRef.ToColorSpace(baseRef.Interpretation()); err != nil {
			return nil, fmt.Errorf("toColorSpace error: %w", err)
		}
	}

	if wmRef.BandFormat() != baseRef.BandFormat() {
		if err := wmRef.Cast(baseRef.BandFormat()); err != nil {
			return nil, fmt.Errorf("cast error: %w", err)
		}
	}

	compositeItems := buildCompositeGrid(baseRef, wmRef, cfg)
	if len(compositeItems) == 0 {
		return nil, fmt.Errorf("no composite items")
	}

	if err := baseRef.CompositeMulti(compositeItems); err != nil {
		return nil, fmt.Errorf("compositeMulti error: %w", err)
	}

	ep := vips.NewJpegExportParams()
	ep.Quality = cfg.Quality
	ep.StripMetadata = true

	outputBytes, _, err := baseRef.ExportJpeg(ep)
	if err != nil {
		return nil, fmt.Errorf("exportJpeg error: %w", err)
	}

	return outputBytes, nil
}

func loadBaseImage(cfg Config) (*vips.ImageRef, error) {
	if len(cfg.ImageBody) > 0 {
		return vips.NewImageFromBuffer(cfg.ImageBody)
	}

	if strings.HasPrefix(cfg.InputPath, "http://") || strings.HasPrefix(cfg.InputPath, "https://") {
		data, err := fetchRemote(cfg.InputPath)
		if err != nil {
			return nil, err
		}
		return vips.NewImageFromBuffer(data)
	}
	return vips.NewImageFromFile(cfg.InputPath)
}

func fetchRemote(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetchRemote error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchRemote error: status code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("readAll error: %w", err)
	}

	return data, nil
}

func determineFontSize(img *vips.ImageRef, cfg Config) float64 {
	diagonalLen := float64(img.Width() + img.Height())
	size := diagonalLen * 0.03
	if size < 24 {
		size = 24
	}
	if size > 120 {
		size = 120
	}
	return size
}

func buildCompositeGrid(baseRef, wmRef *vips.ImageRef, cfg Config) []*vips.ImageComposite {
	wmWidth := wmRef.Width()
	wmHeight := wmRef.Height()

	xStep := int(float64(wmWidth) * cfg.TileSpacingFactor)
	if xStep < cfg.MinTileStep {
		xStep = cfg.MinTileStep
	}
	yStep := int(float64(wmHeight) * cfg.TileSpacingFactor)
	if yStep < cfg.MinTileStep {
		yStep = cfg.MinTileStep
	}

	var items []*vips.ImageComposite
	row := 0
	for y := -wmHeight; y < baseRef.Height()+wmHeight; y += yStep {
		rowOffset := 0
		if row%2 != 0 {
			rowOffset = xStep / 2
		}
		for x := -wmWidth; x < baseRef.Width()+wmWidth; x += xStep {
			finalX := x + rowOffset
			finalY := y

			if finalX+wmWidth <= 0 || finalX >= baseRef.Width() {
				continue
			}
			if finalY+wmHeight <= 0 || finalY >= baseRef.Height() {
				continue
			}

			items = append(items, &vips.ImageComposite{
				Image:     wmRef,
				BlendMode: vips.BlendModeOver,
				X:         finalX,
				Y:         finalY,
			})
		}
		row++
	}

	return items
}

func ensureRGBA(img *vips.ImageRef) error {
	if img.Interpretation() != vips.InterpretationSRGB {
		if err := img.ToColorSpace(vips.InterpretationSRGB); err != nil {
			return fmt.Errorf("toColorSpace error: %w", err)
		}
	}

	if img.Bands() < 4 {
		if err := img.AddAlpha(); err != nil {
			return fmt.Errorf("addAlpha error: %w", err)
		}
	}

	return nil
}

func createTextWatermarkPNG(text string, alpha int, fontSize float64) ([]byte, error) {
	// 使用 LRU 缓存，key 包含文字、透明度和字号（保留一位小数）
	cacheKey := fmt.Sprintf("%s_%d_%.1f", text, alpha, fontSize)
	if data, ok := wmLRU.Get(cacheKey); ok {
		return data, nil
	}

	font, err := getFont()
	if err != nil {
		return nil, err
	}

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(font)
	c.SetFontSize(fontSize)

	tmpImg := image.NewRGBA(image.Rect(0, 0, 1000, 100))
	c.SetClip(tmpImg.Bounds())
	c.SetDst(tmpImg)
	c.SetSrc(image.NewUniform(color.Black))
	ptStart := freetype.Pt(0, int(c.PointToFixed(fontSize)>>6))
	textBounds, err := c.DrawString(text, ptStart)
	if err != nil {
		return nil, err
	}

	textWidth := int((textBounds.X - ptStart.X) >> 6)
	textHeight := int(fontSize * 1.2)

	padding := 10
	width := textWidth + padding*2
	height := textHeight + padding*2

	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(color.RGBA{255, 255, 255, uint8(alpha)})) // more obvious semi-transparent white

	pt := freetype.Pt(padding, padding+int(c.PointToFixed(fontSize)>>6))
	if _, err := c.DrawString(text, pt); err != nil {
		return nil, err
	}

	rotatedImg := imaging.Rotate(img, 30, color.Transparent)

	var pngBuf bytes.Buffer
	if err := imaging.Encode(&pngBuf, rotatedImg, imaging.PNG); err != nil {
		return nil, err
	}

	pngData := pngBuf.Bytes()
	wmLRU.Put(cacheKey, pngData)
	return pngData, nil
}

func getFont() (*truetype.Font, error) {
	var err error
	fontCacheOnce.Do(func() {
		fontCache, err = truetype.Parse(goregular.TTF)
	})
	return fontCache, err
}

func initVIPS() {
	vipsInitOnce.Do(func() {
		vips.LoggingSettings(nil, vips.LogLevelWarning)
		vips.Startup(nil)
	})
}
