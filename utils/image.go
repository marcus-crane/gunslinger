package utils

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"

	"github.com/google/uuid"
	color_extractor "github.com/marekm4/color-extractor"
)

const (
	UserAgent = "Gunslinger/1.0 (gunslinger@utf9k.net)"
)

func BytesToGUIDLocation(image []byte, extension string) (string, uuid.UUID) {
	imageHash := md5.Sum(image)
	var genericBytes []byte = imageHash[:] // Disgusting :)
	guid, _ := uuid.FromBytes(genericBytes)
	location := fmt.Sprintf("/static/cover.%s.%s", guid, extension)
	return location, guid
}

func ExtractImageContent(imageUrl string) ([]byte, string, []string, error) {
	var client http.Client
	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		return []byte{}, "", []string{}, err
	}
	req.Header = http.Header{
		"User-Agent": []string{UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		return []byte{}, "", []string{}, err
	}
	defer res.Body.Close()

	var buf bytes.Buffer
	tee := io.TeeReader(res.Body, &buf)

	body, err := io.ReadAll(tee)
	if err != nil {
		return []byte{}, "", []string{}, err
	}

	mimeType := http.DetectContentType(body)

	extension := ""

	switch mimeType {
	case "image/jpeg":
		extension = "jpeg"
	case "image/png":
		extension = "png"
	}

	var domColours []string

	image, _, _ := image.Decode(&buf)
	colours := color_extractor.ExtractColors(image)
	for _, c := range colours {
		domColours = append(domColours, colorToHexString(c))
	}

	return body, extension, domColours, nil
}

func colorToHexString(c color.Color) string {
	r, g, b, a := c.RGBA()
	rgba := color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
	return fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)
}
