package utils

import (
	"crypto/md5"
	"fmt"

	"github.com/google/uuid"
)

func BytesToGUIDLocation(image []byte, extension string) (string, uuid.UUID) {
	imageHash := md5.Sum(image)
	var genericBytes []byte = imageHash[:] // Disgusting :)
	guid, _ := uuid.FromBytes(genericBytes)
	location := fmt.Sprintf("/static/cover.%s.%s", guid, extension)
	return location, guid
}
