package pdf

import (
	"bytes"
	"path/filepath"

	"github.com/jung-kurt/gofpdf"
)

func (d *Document) Image(path string) (*ImageInfo, error) {
	return d.RegisterImage(path)
}

func (d *Document) RegisterImage(path string) (*ImageInfo, error) {
	if _, ok := d.images[path]; ok {
		return d.images[path], nil
	}

	path, err := NormalizePath(path)
	if err != nil {
		return nil, err
	}

	if d.images == nil {
		d.images = make(map[string]*ImageInfo)
	}

	if _, ok := d.images[path]; ok {
		return d.images[path], nil
	}

	info := d.pdf.RegisterImage(path, "") // empty image-type => autodetect
	info.SetDpi(72)

	img := &ImageInfo{
		doc:  d,
		info: info,
		dpi:  72,
		path: path,
	}
	d.images[path] = img
	return img, nil
}

func (d *Document) RegisterImageFromBytes(name, imageType string, data []byte) (*ImageInfo, error) {
	if d.images == nil {
		d.images = make(map[string]*ImageInfo)
	}

	info := d.pdf.RegisterImageOptionsReader(name, gofpdf.ImageOptions{ImageType: imageType}, bytes.NewReader(data))
	info.SetDpi(72)

	img := &ImageInfo{
		doc:  d,
		info: info,
		dpi:  72,
		path: name,
	}

	d.images[name] = img
	return img, nil
}

func NormalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

func (d *Document) MustImage(path string) *ImageInfo {
	img, err := d.RegisterImage(path)
	if err != nil {
		panic(err)
	}
	return img
}

type ImageInfo struct {
	doc  *Document
	info *gofpdf.ImageInfoType
	Name string
	path string
	dpi  float64 // the DPI we last applied via SetDpi (starts at gofpdf default)
}

// Current physical size in the document's unit (mm, pt, …)

func (img *ImageInfo) Width() Unit  { return Unit(img.info.Width()) }
func (img *ImageInfo) Height() Unit { return Unit(img.info.Height()) }

// SetWidth sets the logical width (keeping the aspect ratio).
func (img *ImageInfo) SetWidth(width Unit) {
	if width <= 0 {
		return // ignore nonsensical input – or panic/log, your choice
	}
	oldWidth := img.info.Width()
	if oldWidth == 0 {
		return // should never happen if info is valid
	}

	// newDPI = oldDPI * oldWidth / newWidth     (pixel count stays constant)
	newDpi := img.dpi * oldWidth / width.Pt()
	img.info.SetDpi(newDpi)
	img.dpi = newDpi
}

// SetHeight sets the logical height (keeping the aspect ratio).
func (img *ImageInfo) SetHeight(height Unit) {
	if height <= 0 {
		return
	}
	oldHeight := img.info.Height()
	if oldHeight == 0 {
		return
	}

	newDpi := img.dpi * oldHeight / float64(height)
	img.info.SetDpi(newDpi)
	img.dpi = newDpi
}

// Direct access if the caller already knows the desired DPI.
func (img *ImageInfo) SetDpi(dpi float64) {
	if dpi <= 0 {
		return
	}
	img.info.SetDpi(dpi)
	img.dpi = dpi
}
