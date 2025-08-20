package pkg

import (
	"errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gomodlag/internal/storage"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"unicode"
)

func Validator(login string, password string) bool {
	if len([]rune(login)) < 8 || len([]rune(password)) < 8 {
		return false
	}
	var hasDigitInLogin bool
	onlyLatin := true
	for _, v := range login {
		if unicode.IsNumber(v) {
			hasDigitInLogin = true
		}
		if !unicode.Is(unicode.Latin, v) {
			onlyLatin = false
		}
	}
	if !hasDigitInLogin || !onlyLatin {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, v := range password {
		switch {
		case unicode.IsUpper(v):
			hasUpper = true
		case unicode.IsLower(v):
			hasLower = true
		case unicode.IsDigit(v):
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	if !(hasUpper && hasLower && hasDigit && hasSpecial) {
		return false
	}

	return true
}

func CreateHash(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

//	func CompareHash(hash string, password string) bool {
//		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
//		if err != nil {
//			return false
//		}
//		return true
//	}
func GenerateDockId() uuid.UUID {
	return uuid.New()
}
func GenerateToken() string {
	return uuid.New().String()
}

var allowedMIMEs = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"video/mp4":  true,
}

func SaveFile(fileHeader *multipart.FileHeader, destDir string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		return "", err
	}

	mimeType := http.DetectContentType(buf)
	if !allowedMIMEs[mimeType] {
		return "", errors.New("unsupported file type")
	}

	id := uuid.New().String()
	ext := filepath.Ext(fileHeader.Filename)
	filename := id + ext
	destPath := filepath.Join(destDir, filename)

	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}
	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}

	return filename, nil
}
func GetFile(path string) ([]byte, error) {
	FilePath, err := os.Open(path)

	if err != nil {
		return nil, storage.Internal // 500
	}

	contents, err := io.ReadAll(FilePath)
	if err != nil {
		return nil, storage.Internal /// 500
	}
	return contents, nil
}
