package utils

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/models"
	color_extractor "github.com/marekm4/color-extractor"
)

// TODO: Move into config
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

func ExtractImageContent(imageUrl string) ([]byte, string, models.SerializableColours, error) {
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

	image, _, err := image.Decode(&buf)
	if err != nil {
		return []byte{}, "", []string{}, err
	}
	fmt.Printf(image.Bounds().String())
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

func LoadCover(cfg config.Config, hash string, extension string) (string, error) {
	// TODO: Properly standardise on webp or something
	if strings.Contains(hash, "retroachievements") {
		extension = "png"
	}
	coverLocation := fmt.Sprintf("%s/%s.%s", cfg.Gunslinger.StorageDir, strings.ReplaceAll(hash, ":", "."), extension)
	slog.With(slog.String("cover_location", coverLocation)).Debug("Loading cover from disc")
	img, err := os.ReadFile(coverLocation)
	if err != nil {
		slog.With(slog.String("error", err.Error())).Error("Failed to load image")
		return "", err
	}
	return string(img), nil
}

func SaveCover(cfg config.Config, hash string, image []byte, extension string) error {
	coverLocation := fmt.Sprintf("%s/%s.%s", cfg.Gunslinger.StorageDir, strings.ReplaceAll(hash, ":", "."), extension)
	slog.With(slog.String("cover_location", coverLocation)).Debug("Saving cover to disc")
	return os.WriteFile(coverLocation, image, 0644)
}
