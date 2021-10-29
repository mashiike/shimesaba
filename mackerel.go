package shimesaba

import (
	"context"
	"fmt"
	"log"
	"time"

	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

type MackerelClient interface {
	FindHosts(param *mackerel.FindHostsParam) ([]*mackerel.Host, error)
	FetchHostMetricValues(hostID string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error)
	FetchServiceMetricValues(serviceName string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error)
}

type Repository struct {
	client MackerelClient
}

func NewRepository(client MackerelClient) *Repository {
	return &Repository{
		client: client,
	}
}

const (
	fetchMetricMetricmit = 6 * time.Hour
)

func (repo *Repository) FetchMetric(ctx context.Context, cfg *MetricConfig, startAt time.Time, endAt time.Time) (*Metric, error) {
	iter := timeutils.NewIterator(startAt, endAt, fetchMetricMetricmit)
	m := NewMetric(cfg)

	var fetchMetricValues func(int64, int64) ([]mackerel.MetricValue, error)
	switch cfg.Type {
	case HostMetric:
		hosts, err := repo.client.FindHosts(&mackerel.FindHostsParam{
			Service: cfg.ServiceName,
			Roles:   cfg.Roles,
			Name:    cfg.HostName,
		})
		if err != nil {
			return nil, err
		}
		fetchMetricValues = func(from, to int64) ([]mackerel.MetricValue, error) {
			values := make([]mackerel.MetricValue, 0)
			for _, host := range hosts {
				log.Printf("[debug] call MackerelClient.FetchHostMetricValues(%s,%s,%s,%s)", host.ID, cfg.Name, time.Unix(from, 0), time.Unix(to, 0))
				v, err := repo.client.FetchHostMetricValues(host.ID, cfg.Name, from, to)
				if err != nil {
					return nil, err
				}
				values = append(values, v...)
			}
			return values, nil
		}
	case ServiceMetric:
		fetchMetricValues = func(from, to int64) ([]mackerel.MetricValue, error) {
			log.Printf("[debug] call MackerelClient.FetchServiceMetricValues(%s,%s,%s,%s)", cfg.ServiceName, cfg.Name, time.Unix(from, 0), time.Unix(to, 0))
			return repo.client.FetchServiceMetricValues(cfg.ServiceName, cfg.Name, from, to)
		}
	default:
		return nil, fmt.Errorf("metric type `%s` is unknown", cfg.Type)
	}

	for iter.HasNext() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		curStartat, curEndAt := iter.Next()
		values, err := fetchMetricValues(curStartat.Unix(), curEndAt.Unix())
		if err != nil {
			return nil, fmt.Errorf("metric=%s :%w", m.ID(), err)
		}
		for _, value := range values {
			if err := m.AppendValue(time.Unix(value.Time, 0), value.Value); err != nil {
				return nil, fmt.Errorf("metric=%s :%w", m.ID(), err)
			}
		}
		time.Sleep(500 * time.Microsecond)
	}
	return m, nil
}

func (repo *Repository) FetchMetrics(ctx context.Context, cfgs MetricConfigs, startAt time.Time, endAt time.Time) (Metrics, error) {
	ms := make(Metrics)
	for _, cfg := range cfgs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		log.Printf("[info] start fetch metric_id=%s\n", cfg.ID)
		m, err := repo.FetchMetric(ctx, cfg, startAt, endAt)
		log.Printf("[info] finished fetch metric_id=%s\n", cfg.ID)
		if err != nil {
			return nil, err
		}
		ms.Set(m)
	}
	return ms, nil
}

func (repo *Repository) SaveReports(ctx context.Context, reports []*Report) error {
	log.Println("[warn] SaveReports not implemented yet, now nothing todo")
	return nil
}
