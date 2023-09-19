package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type envelope map[string]interface{}

func (app *application) readIDParam(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope,
	headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {

	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {

		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		switch {

		case errors.As(err, &syntaxError):
			return fmt.Errorf("O corpo está com JSON formatado incorretamente (no caractere %d)",
				syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("O corpo está com JSON formatado incorretamente")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("O corpo contém tipo JSON incorreto para o campo %q",
					unmarshalTypeError.Field)
			}
			return fmt.Errorf("O corpo contém tipo JSON incorreto (no caractere %d)",
				unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("O corpo não pode ficar vazio")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("O corpo contém uma chave desconhecida %s", fieldName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("O corpo não pode ter tamanho maior que %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}

	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("O corpo só pode conter um valor JSON")
	}

	return nil
}
