package utility

import (
	"chrononewsapi/internal/entity"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

func CreateFileName(file *multipart.FileHeader) string {
	return uuid.New().String() + filepath.Ext(file.Filename)
}

func ExtractFileIDsFromContent(content string) ([]int32, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	var fileIDs []int32
	seenIDs := make(map[int32]bool)

	doc.Find("img").Each(func(_ int, sel *goquery.Selection) {
		if dataID, exists := sel.Attr("data-id"); exists {
			id, err := strconv.ParseUint(dataID, 10, 32)
			if err == nil && !seenIDs[int32(id)] {
				fileIDs = append(fileIDs, int32(id))
				seenIDs[int32(id)] = true
			}
		}
	})

	return fileIDs, nil
}

func StripImageSrcFromContent(content string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", err
	}

	doc.Find("img").Each(func(_ int, sel *goquery.Selection) {
		sel.RemoveAttr("src")
	})

	return doc.Html()
}

func RebuildContentWithImageSrc(content string, fileMap map[int32]*entity.File) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", err
	}

	doc.Find("img").Each(func(_ int, sel *goquery.Selection) {
		if dataID, exists := sel.Attr("data-id"); exists {
			if id, err := strconv.ParseInt(dataID, 10, 32); err == nil {
				if file, ok := fileMap[int32(id)]; ok {
					sel.SetAttr("src", file.Name)
				} else {
					sel.SetAttr("src", "")
				}
			}
		}
	})

	return doc.Html()
}
