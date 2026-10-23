package main

import (
	"errors"
	"io"
	"net/url"
)

func parseWebURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("protocolo inválido")
	}
	if u.User != nil {
		return nil, errors.New("URL com credenciais não é aceita")
	}
	return u, nil
}

func ioCopyLimited(dst io.Writer, src io.Reader, max int64) (int64, error) {
	n, err := io.Copy(dst, io.LimitReader(src, max+1))
	if err != nil {
		return n, err
	}
	if n > max {
		return n, errors.New("arquivo maior que o esperado")
	}
	return n, nil
}
