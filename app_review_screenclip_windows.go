//go:build windows

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/bits"
	"time"
	"unsafe"

	"github.com/run-bigpig/jcp/internal/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	cfBitmap = 2
	cfDIB    = 8
	cfDIBV5  = 17
)

var (
	user32                         = windows.NewLazySystemDLL("user32.dll")
	shell32                        = windows.NewLazySystemDLL("shell32.dll")
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procGetClipboardSequenceNumber = user32.NewProc("GetClipboardSequenceNumber")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procShellExecuteW              = shell32.NewProc("ShellExecuteW")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
	procGlobalSize                 = kernel32.NewProc("GlobalSize")
)

func (a *App) CaptureReviewScreenClip() models.ReviewScreenCaptureResult {
	before := getClipboardSequenceNumber()
	if a.ctx != nil {
		runtime.WindowMinimise(a.ctx)
		time.Sleep(250 * time.Millisecond)
		defer func() {
			runtime.WindowUnminimise(a.ctx)
			runtime.WindowShow(a.ctx)
		}()
	}

	if err := shellExecuteURI("ms-screenclip:"); err != nil {
		return models.ReviewScreenCaptureResult{Error: err.Error()}
	}

	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(250 * time.Millisecond)
		if getClipboardSequenceNumber() == before || !clipboardHasImage() {
			continue
		}
		time.Sleep(200 * time.Millisecond)
		data, err := exportClipboardImagePNG()
		if err != nil {
			return models.ReviewScreenCaptureResult{Error: err.Error()}
		}
		return models.ReviewScreenCaptureResult{
			DataBase64: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
		}
	}
	return models.ReviewScreenCaptureResult{Error: "截图已取消或超时"}
}

func getClipboardSequenceNumber() uint32 {
	ret, _, _ := procGetClipboardSequenceNumber.Call()
	return uint32(ret)
}

func clipboardHasImage() bool {
	for _, format := range []uintptr{cfDIBV5, cfDIB, cfBitmap} {
		ret, _, _ := procIsClipboardFormatAvailable.Call(format)
		if ret != 0 {
			return true
		}
	}
	return false
}

func shellExecuteURI(uri string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(uri)
	if err != nil {
		return err
	}
	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(target)),
		0,
		0,
		uintptr(windows.SW_SHOWNORMAL),
	)
	if ret <= 32 {
		if callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("启动系统截图失败: %w", callErr)
		}
		return fmt.Errorf("启动系统截图失败，ShellExecute 返回 %d", ret)
	}
	return nil
}

func exportClipboardImagePNG() ([]byte, error) {
	data, err := readClipboardDIB()
	if err != nil {
		return nil, err
	}
	img, err := dibToImage(data)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("编码截图失败: %w", err)
	}
	return buf.Bytes(), nil
}

func readClipboardDIB() ([]byte, error) {
	ret, _, callErr := procOpenClipboard.Call(0)
	if ret == 0 {
		return nil, fmt.Errorf("打开剪贴板失败: %w", callErr)
	}
	defer procCloseClipboard.Call()

	for _, format := range []uintptr{cfDIBV5, cfDIB} {
		handle, _, _ := procGetClipboardData.Call(format)
		if handle == 0 {
			continue
		}
		size, _, _ := procGlobalSize.Call(handle)
		if size == 0 {
			continue
		}
		ptr, _, _ := procGlobalLock.Call(handle)
		if ptr == 0 {
			continue
		}
		data := make([]byte, int(size))
		copy(data, unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(size)))
		procGlobalUnlock.Call(handle)
		return data, nil
	}
	return nil, fmt.Errorf("剪贴板中没有可读取的截图位图")
}

func dibToImage(data []byte) (image.Image, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("剪贴板截图数据不完整")
	}
	headerSize := int(binary.LittleEndian.Uint32(data[0:4]))
	if headerSize < 40 || headerSize > len(data) {
		return nil, fmt.Errorf("剪贴板截图头信息无效")
	}

	width := int(int32(binary.LittleEndian.Uint32(data[4:8])))
	rawHeight := int(int32(binary.LittleEndian.Uint32(data[8:12])))
	bitCount := int(binary.LittleEndian.Uint16(data[14:16]))
	compression := binary.LittleEndian.Uint32(data[16:20])
	clrUsed := uint32(0)
	if headerSize >= 40 {
		clrUsed = binary.LittleEndian.Uint32(data[32:36])
	}
	if width <= 0 || rawHeight == 0 {
		return nil, fmt.Errorf("剪贴板截图尺寸无效")
	}
	height := rawHeight
	topDown := false
	if height < 0 {
		height = -height
		topDown = true
	}

	pixelOffset, masks, err := dibPixelOffsetAndMasks(data, headerSize, bitCount, compression, clrUsed)
	if err != nil {
		return nil, err
	}
	if pixelOffset >= len(data) {
		return nil, fmt.Errorf("剪贴板截图像素数据缺失")
	}

	stride := ((width*bitCount + 31) / 32) * 4
	required := pixelOffset + stride*height
	if required > len(data) {
		return nil, fmt.Errorf("剪贴板截图像素数据不完整")
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	switch bitCount {
	case 32:
		hasAlpha := false
		for y := 0; y < height; y++ {
			srcY := y
			if !topDown {
				srcY = height - 1 - y
			}
			row := data[pixelOffset+srcY*stride : pixelOffset+srcY*stride+width*4]
			for x := 0; x < width; x++ {
				raw := binary.LittleEndian.Uint32(row[x*4 : x*4+4])
				r, g, b, a := maskRGBA(raw, masks)
				if a != 0 {
					hasAlpha = true
				}
				img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
			}
		}
		if !hasAlpha {
			for i := 3; i < len(img.Pix); i += 4 {
				img.Pix[i] = 255
			}
		}
	case 24:
		for y := 0; y < height; y++ {
			srcY := y
			if !topDown {
				srcY = height - 1 - y
			}
			row := data[pixelOffset+srcY*stride : pixelOffset+srcY*stride+width*3]
			for x := 0; x < width; x++ {
				b := row[x*3]
				g := row[x*3+1]
				r := row[x*3+2]
				img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
			}
		}
	default:
		return nil, fmt.Errorf("暂不支持 %d 位截图格式", bitCount)
	}
	return img, nil
}

type dibMasks struct {
	red   uint32
	green uint32
	blue  uint32
	alpha uint32
}

func dibPixelOffsetAndMasks(data []byte, headerSize int, bitCount int, compression uint32, clrUsed uint32) (int, dibMasks, error) {
	const (
		biRGB       = 0
		biBitFields = 3
	)
	masks := dibMasks{red: 0x00ff0000, green: 0x0000ff00, blue: 0x000000ff, alpha: 0xff000000}

	if bitCount != 24 && bitCount != 32 {
		return 0, masks, fmt.Errorf("暂不支持 %d 位截图格式", bitCount)
	}
	if compression != biRGB && compression != biBitFields {
		return 0, masks, fmt.Errorf("暂不支持压缩截图格式 %d", compression)
	}

	offset := headerSize
	if compression == biBitFields {
		if headerSize >= 52 {
			masks.red = binary.LittleEndian.Uint32(data[40:44])
			masks.green = binary.LittleEndian.Uint32(data[44:48])
			masks.blue = binary.LittleEndian.Uint32(data[48:52])
			if headerSize >= 56 {
				masks.alpha = binary.LittleEndian.Uint32(data[52:56])
			} else {
				masks.alpha = 0
			}
		} else {
			if len(data) < offset+12 {
				return 0, masks, fmt.Errorf("剪贴板截图颜色掩码缺失")
			}
			masks.red = binary.LittleEndian.Uint32(data[offset : offset+4])
			masks.green = binary.LittleEndian.Uint32(data[offset+4 : offset+8])
			masks.blue = binary.LittleEndian.Uint32(data[offset+8 : offset+12])
			masks.alpha = 0
			offset += 12
		}
	}

	if bitCount <= 8 {
		colors := int(clrUsed)
		if colors == 0 {
			colors = 1 << bitCount
		}
		offset += colors * 4
	}
	return offset, masks, nil
}

func maskRGBA(raw uint32, masks dibMasks) (uint8, uint8, uint8, uint8) {
	return maskToByte(raw, masks.red), maskToByte(raw, masks.green), maskToByte(raw, masks.blue), maskToByte(raw, masks.alpha)
}

func maskToByte(raw uint32, mask uint32) uint8 {
	if mask == 0 {
		return 255
	}
	shift := bits.TrailingZeros32(mask)
	width := 32 - bits.LeadingZeros32(mask>>shift)
	value := (raw & mask) >> shift
	if width >= 8 {
		return uint8(value >> (width - 8))
	}
	maxValue := uint32((1 << width) - 1)
	if maxValue == 0 {
		return 0
	}
	return uint8((value * 255) / maxValue)
}
