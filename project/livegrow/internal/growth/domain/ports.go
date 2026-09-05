package domain

import "context"

type CampaignRepository interface {
	GetPublished(ctx context.Context, campaignID string) (Campaign, error)
}

type MetricsSink interface {
	RecordRevenue(ctx context.Context, event RevenueEvent) error
}

type Campaign struct {
	ID      string
	Version int64
	Status  string
}

type RevenueEvent struct {
	OrderID    string
	CampaignID string
	Amount     int64
}
