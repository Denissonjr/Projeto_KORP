package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// projetoKorpResponse representa o corpo de resposta do endpoint /projeto-korp
type projetoKorpResponse struct {
	Nome    string `json:"nome"`
	Horario string `json:"horario"`
}

var (
	// requestsTotal contabiliza o volume de requisições recebidas pelo serviço,
	// segmentado por método, caminho e código de status HTTP.
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_server_projeto_korp_requests_total",
			Help: "Número total de requisições HTTP recebidas pelo http-server-projeto-korp",
		},
		[]string{"method", "path", "status"},
	)

	// requestDuration mede a latência das requisições, útil para observabilidade
	// além da simples contagem de disponibilidade/volume exigidas no desafio.
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_server_projeto_korp_request_duration_seconds",
			Help:    "Duração das requisições HTTP em segundos",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// up expõe a disponibilidade do serviço. Enquanto o processo estiver rodando
	// e respondendo, o valor permanece em 1. Prometheus também usa a métrica
	// nativa "up" resultante do scrape, mas esta métrica de aplicação reforça
	// a disponibilidade a nível de negócio.
	up = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_server_projeto_korp_up",
		Help: "Indica se o http-server-projeto-korp está disponível (1 = disponível)",
	})
)

// instrument envolve um handler HTTP registrando métricas de contagem e duração.
func instrument(path string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		handler(rec, r)

		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues(r.Method, path).Observe(duration)
		requestsTotal.WithLabelValues(r.Method, path, http.StatusText(rec.status)).Inc()
	}
}

// statusRecorder captura o status code efetivamente escrito na resposta.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func projetoKorpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}

	resp := projetoKorpResponse{
		Nome:    "Projeto Korp",
		Horario: time.Now().UTC().Format(time.RFC3339Nano),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// healthHandler é um endpoint dedicado de disponibilidade (liveness),
// usado por health checks externos (ex: docker healthcheck, load balancers).
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	up.Set(1)

	mux := http.NewServeMux()
	mux.HandleFunc("/projeto-korp", instrument("/projeto-korp", projetoKorpHandler))
	mux.HandleFunc("/health", instrument("/health", healthHandler))
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":8080"
	log.Printf("http-server-projeto-korp escutando em %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
