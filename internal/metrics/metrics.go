package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TotalMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_messages_total_count",
		Help: "The total number of messages sent",
	})

	OnlineUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_online_users_count",
		Help: "The current number of online users",
	})

	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "chat_request_duration_seconds",
		Help:    "Histogram of request durations",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "status"})
)
