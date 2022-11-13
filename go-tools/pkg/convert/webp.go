package convert

import (
	"fmt"
	"image/png"
	"os"

	"golang.org/x/image/webp"
)

func ConvertWebp2Png(src, dst string) {
	f0, err := os.Open(src)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f0.Close()
	img0, err := webp.Decode(f0)
	if err != nil {
		fmt.Println(err)
		fmt.Println(err)
		return
	}
	pngFile, err := os.Create(dst)
	if err != nil {
		fmt.Println(err)
	}
	err = png.Encode(pngFile, img0)
	if err != nil {
		fmt.Println(err)
	}
}
