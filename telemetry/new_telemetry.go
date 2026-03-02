package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

var(
	logger *slog.Logger
	logChannel chan []byte
	ooURL string
	ooAuth string
	httpClient *http.Client
	hostname string
	appName string
	serviceVer = "0.1.0"
	initialized bool
	wg sync.WaitGroup
)

func init(){
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func Info (msg string, args ...any){
	logger.Info(msg, args...)
}

func Init(url, auth string, host string, app string){
	if url == "" || auth == ""{
		return
	}

	ooURL = url
	ooAuth = auth
	hostname = host
	appName = app
	logChannel = make(chan []byte, 1000)
	httpClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	initialized = true

	wg.Add(1)
	go startSender()
}

func startSender(){
	defer wg.Done()
	for payload := range logChannel{
		sendToOpenObserve(payload)
	}
}

func sendToOpenObserve(payload []byte) {
    // Adicione este Print para ver exatamente o que está saindo da sua lib
    // fmt.Printf("Enviando para OpenObserve: %s\n", string(payload))

    req, err := http.NewRequest("POST", ooURL, bytes.NewReader(payload))
    // ... (configuração de headers)
		req.Header.Set("Content-Type", "application/json")

	authHeader := ooAuth
	if len(authHeader) < 6 || authHeader[:6] != "Basic"{
		authHeader = "Basic " + authHeader
	}
	req.Header.Set("Authorization", authHeader)
	
    resp, err := httpClient.Do(req)
    if err != nil {
        fmt.Printf("Erro de conexão: %v\n", err)
        return
    }
    defer resp.Body.Close()

    // LEIA O CORPO DA RESPOSTA - É aqui que o OpenObserve explica por que ignorou os campos
    buf := new(bytes.Buffer)
    buf.ReadFrom(resp.Body)
    
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
        fmt.Printf("ERRO OPENOBSERVE (Status %d): %s\n", resp.StatusCode, buf.String())
    } else {
        // Se retornar 200 mas o JSON de resposta tiver "error", você verá aqui
        fmt.Printf("Resposta OpenObserve: %s\n", buf.String())
    }
}

type Event struct{
	name string
	fields map[string]any
	start time.Time
}

func NewEvent(name string, timestamp time.Time) *Event{
	return &Event{
		name: name,
		fields:make(map[string]any),
		start: timestamp,
	}
}

func (e *Event) Add(key string, value any){
	e.fields[key] = value
}

func (e *Event) AddError(err error){
	e.fields["error"] = err.Error()
	e.fields["success"] = false
}

func (e *Event) End() {
  duration := time.Since(e.start)
  e.fields["duration_ms"] = duration.Milliseconds()

  if _, ok := e.fields["success"]; !ok {
    _, hasError := e.fields["error"]
    e.fields["success"] = !hasError
  }

  e.fields["host"] = hostname
  e.fields["service"] = appName
  e.fields["version"] = serviceVer
  e.fields["message"] = e.name
  
  e.fields["timestamp"] = e.start.Format(time.RFC3339)

  args := make([]any, 0, len(e.fields)*2)
  for k, v := range e.fields {
    args = append(args, k, v)
  }
  logger.Info(e.name, args...)

  if initialized {
    jsonBytes, err := json.Marshal(e.fields) 
    if err == nil {
      logChannel <- jsonBytes
    }
  }
}

type contextkey struct{}

func FromContext(ctx context.Context) *Event{
	if ctx == nil{
		return nil
	}
	val := ctx.Value(contextkey{})
	if val == nil{
		return nil
	}
	return val.(*Event)
}

func WithContext(ctx context.Context, e *Event) context.Context{
	return context.WithValue(ctx, contextkey{}, e)
}