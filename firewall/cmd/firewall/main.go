//go:build !solution

package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Rules []Rule `yaml:"rules"`
}

type Rule struct {
	Endpoint               string   `yaml:"endpoint"`
	ForbiddenUserAgents    []string `yaml:"forbidden_user_agents"`
	ForbiddenHeaders       []string `yaml:"forbidden_headers"`
	RequiredHeaders        []string `yaml:"required_headers"`
	MaxRequestLengthBytes  int64    `yaml:"max_request_length_bytes"`
	MaxResponseLengthBytes int64    `yaml:"max_response_length_bytes"`
	ForbiddenResponseCodes []int    `yaml:"forbidden_response_codes"`
	ForbiddenRequestRE     []string `yaml:"forbidden_request_re"`
	ForbiddenResponseRE    []string `yaml:"forbidden_response_re"`
}

var errForbidden = errors.New("forbidden")

func matches(s string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, s)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func hasForbiddenHeaders(header http.Header, forbidden []string) bool {
	for name, values := range header {
		for _, value := range values {
			if matches(name+": "+value, forbidden) {
				return true
			}
		}
	}
	return false
}

func hasRequiredHeaders(header http.Header, required []string) bool {
	for _, r := range required {
		if header.Get(r) == "" {
			return false
		}
	}
	return true
}

func checkRequest(req *http.Request, rules []Rule) error {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	req.Body.Close()

	// восстанавливаем тело запроса
	req.Body = io.NopCloser(strings.NewReader(string(body)))

	for _, rule := range rules {
		if rule.Endpoint != req.URL.Path {
			continue
		}

		if matches(req.UserAgent(), rule.ForbiddenUserAgents) {
			return errForbidden
		}

		if hasForbiddenHeaders(req.Header, rule.ForbiddenHeaders) {
			return errForbidden
		}

		if !hasRequiredHeaders(req.Header, rule.RequiredHeaders) {
			return errForbidden
		}

		if rule.MaxRequestLengthBytes > 0 && int64(len(body)) > rule.MaxRequestLengthBytes {
			return errForbidden
		}

		if matches(string(body), rule.ForbiddenRequestRE) {
			return errForbidden
		}
	}

	return nil
}

func isForbiddenResponseCode(statusCode int, forbidden []int) bool {
	for _, code := range forbidden {
		if statusCode == code {
			return true
		}
	}
	return false
}

func checkResponse(resp *http.Response, rules []Rule) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	// восстанавливаем тело ответа
	resp.Body = io.NopCloser(strings.NewReader(string(body)))

	for _, rule := range rules {
		if rule.Endpoint != resp.Request.URL.Path { // TODO: resp.Request может быть nil
			continue
		}

		if rule.MaxResponseLengthBytes > 0 && int64(len(body)) > rule.MaxResponseLengthBytes {
			return errForbidden
		}

		if isForbiddenResponseCode(resp.StatusCode, rule.ForbiddenResponseCodes) {
			return errForbidden
		}

		if matches(string(body), rule.ForbiddenResponseRE) {
			return errForbidden
		}
	}

	return nil
}

func forbiddenResponse(req *http.Request) *http.Response {
	const body = "Forbidden"

	return &http.Response{
		Status:        "403 Forbidden",
		StatusCode:    http.StatusForbidden,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}

type FirewallTripper struct {
	base  http.RoundTripper
	rules []Rule
}

func (f *FirewallTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	// копируем запрос
	upstreamReq := req.Clone(req.Context())

	if req.Body != nil {
		defer req.Body.Close()
	}

	// проверяем запрос
	if err := checkRequest(upstreamReq, f.rules); err != nil {
		if errors.Is(err, errForbidden) {
			return forbiddenResponse(req), nil
		}
		return nil, err
	}

	// отправляем запрос backend-сервису
	resp, err := f.base.RoundTrip(upstreamReq)
	if err != nil {
		return nil, err
	}

	// проверяем ответ backend-сервиса
	if err := checkResponse(resp, f.rules); err != nil {
		resp.Body.Close()

		if errors.Is(err, errForbidden) {
			return forbiddenResponse(req), nil
		}
		return nil, err
	}

	return resp, nil
}

func main() {
	// `-service-addr` - адрес защищаемого сервиса
	// `-conf` - путь к .yaml конфигу с правилами
	// `-addr` - адрес, на котором будет развёрнут файрвол

	serviceAddr := flag.String("service-addr", "", "address of the service to protect")
	confPath := flag.String("conf", "", "path to the .yaml config file with rules")
	addr := flag.String("addr", "", "address to run the firewall on")
	flag.Parse()

	if *serviceAddr == "" || *confPath == "" || *addr == "" {
		flag.Usage()
		return
	}

	confData, err := os.ReadFile(*confPath)
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	if err := yaml.Unmarshal(confData, &config); err != nil {
		log.Fatal(err)
	}

	target, err := url.Parse(*serviceAddr)
	if err != nil || target.Scheme == "" || target.Host == "" {
		log.Fatal("invalid service address")
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = &FirewallTripper{
		base:  http.DefaultTransport,
		rules: config.Rules,
	}

	if err := http.ListenAndServe(*addr, proxy); err != nil {
		log.Fatal(err)
	}
}
