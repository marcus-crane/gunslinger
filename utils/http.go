package utils

import "net/http"

type UARoundtripper struct {
	RT http.RoundTripper
}

func (uart *UARoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", UserAgent)
	return uart.RT.RoundTrip(req)
}

func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: &UARoundtripper{},
	}
}
