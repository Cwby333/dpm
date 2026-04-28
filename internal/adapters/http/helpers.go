package http

import (
	"bytes"
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"sync"
	// "bytes"
)

var (
	ErrMissingSongID = errors.New("Missing songID in path parameter")
)

const (
	defaultCountWorkers = 5
)

type Request interface {
	GetBody() *multipart.Reader
}

type contentTypeMissingError struct {
	fieldname string
}

func (e contentTypeMissingError) Error() string {
	return fmt.Sprint("Missing Content-Type for %s", e.fieldname)
}

func helper400ContentType(fieldname string) error {
	return contentTypeMissingError{
		fieldname: fieldname,
	}
}

func checkContentTypeAndSet(part *multipart.Part, body *[]byte, song *models.Song, songData *models.SongData) error {
	contentType := ""

	switch part.FormName() {
	case "name":
		song.Title = string(*body)
	case "music":
		if contentType = part.Header.Get("Content-Type"); contentType == "" {
			slog.Info("missing Content-Type")

			err := helper400ContentType(part.FormName())
			return err
		}

		songData.DataSong.Data = *body
		songData.DataSong.ContentType = contentType
	default:
		slog.Info(fmt.Sprintf("unknown form name=%s", part.FormName()))
		return nil
	}

	return nil
}

// readPartWithCustomBuffer читает part с использованием кастомного буфера
func readPartWithCustomBuffer(part *multipart.Part) ([]byte, error) {
	// Создаем буфер с начальным размером 32KB
	buf := make([]byte, 32*1024)
	result := make([]byte, 0)

	for {
		n, err := part.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return result, nil
}

func startReadForm(ctx context.Context, request Request, op string) (*models.Song, *models.SongData, error) {
	ctxWorkers, cancel := context.WithCancelCause(ctx)
	defer cancel(errors.New("defer cancel"))

	var song = models.Song{}
	var songData = models.SongData{}
	chanMultipart := make(chan *multipart.Part)
	chanMultipartForWorkers := make(chan *multipart.Part)
	wg := &sync.WaitGroup{}
	doneChan := make(chan struct{})

	reader := request.GetBody()

	// Функция для полного чтения part без вызова Close
	readPartCompletely := func(part *multipart.Part) ([]byte, error) {
		// Используем bytes.Buffer для накопления данных
		var buf bytes.Buffer

		// Читаем частями по 32KB
		chunk := make([]byte, 32*1024)
		for {
			n, err := part.Read(chunk)
			if n > 0 {
				buf.Write(chunk[:n])
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
		}

		// Не вызываем part.Close() - это важно!
		return buf.Bytes(), nil
	}

	worker := func(
		ctx context.Context,
		cancel context.CancelCauseFunc,
		multipartChannel chan *multipart.Part,
		song *models.Song,
		songData *models.SongData) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			select {
			case <-ctx.Done():
				return
			case part, ok := <-multipartChannel:
				if !ok {
					return
				}

				// Читаем данные без вызова Close
				body, err := readPartCompletely(part)
				if err != nil {
					cancel(err)
					return
				}

				err = checkContentTypeAndSet(part, &body, song, songData)
				if err != nil {
					cancel(err)
					return
				}
			}
		}
	}

	go func() {
		defer close(chanMultipartForWorkers)

		for {
			select {
			case <-ctx.Done():
				return
			case part, ok := <-chanMultipart:
				if !ok {
					return
				}

				select {
				case <-ctx.Done():
					return
				case chanMultipartForWorkers <- part:
				}
			}
		}
	}()

	go func() {
		defer close(chanMultipart)

		for {
			part, err := reader.NextPart()
			if err != nil {
				if errors.Is(err, io.EOF) {
					slog.Info("parts EOF")
					break
				}

				slog.Info(op, slog.String("err", err.Error()))

				cancel(err)
				return
			}

			select {
			case <-ctx.Done():
				return
			case chanMultipart <- part:
			}
		}
	}()

	for range defaultCountWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctxWorkers, cancel, chanMultipartForWorkers, &song, &songData)
		}()
	}

	wg.Wait()
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-ctxWorkers.Done():
		err := ctxWorkers.Err()
		return nil, nil, err
	case <-doneChan:
		return &song, &songData, nil
	}
}
