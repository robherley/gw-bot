package gw

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/robherley/gw-bot/internal/db"
	"github.com/robherley/gw-bot/internal/db/sqlgen"
)

type Item struct {
	ItemID           int64     `json:"itemId"`
	CategoryID       int64     `json:"categoryId"`
	CategoryName     string    `json:"categoryName"`
	CategoryFullName string    `json:"catFullName"`
	Title            string    `json:"title"`
	CurrentPrice     float64   `json:"currentPrice"`
	NumBids          int64     `json:"numBids"`
	BuyNowPrice      float64   `json:"buyNowPrice"`
	ImageURL         string    `json:"imageUrl"`
	ListingType      int       `json:"listingType"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
}

func (i *Item) URL() string {
	return fmt.Sprintf("https://www.shopgoodwill.com/item/%d", i.ItemID)
}

func (i *Item) HasAuction() bool {
	return i.ListingType == 0 || i.ListingType == 2
}

func (i *Item) HasBuyNow() bool {
	return i.ListingType == 1 || i.ListingType == 2
}

func (i *Item) Kind() string {
	kinds := make([]string, 0)

	if i.HasAuction() {
		kinds = append(kinds, "Auction")
	}

	if i.HasBuyNow() {
		kinds = append(kinds, "Buy It Now")
	}

	return strings.Join(kinds, "/")
}

func (i *Item) RelativeEndTime() string {
	duration := time.Until(i.EndTime)
	if duration <= 0 {
		return "Ended"
	}

	b := strings.Builder{}

	if duration.Hours() > 24 {
		days := int(duration.Hours() / 24)
		b.WriteString(fmt.Sprintf("%dd ", days))
	}

	hours := int(duration.Hours()) % 24
	if hours > 0 {
		b.WriteString(fmt.Sprintf("%dh ", hours))
	}

	minutes := int(duration.Minutes()) % 60
	if minutes > 0 {
		b.WriteString(fmt.Sprintf("%dm ", minutes))
	}

	seconds := int(duration.Seconds()) % 60
	if seconds > 0 {
		b.WriteString(fmt.Sprintf("%ds", seconds))
	}

	return strings.TrimSpace(b.String())
}

func (i *Item) UnmarshalJSON(data []byte) error {
	type Alias Item
	tmp := &struct {
		StartTimeRaw       string `json:"startTime"`
		EndTimeRaw         string `json:"endTime"`
		ImageServer        string `json:"imageServer"`
		ImageURLString     string `json:"imageUrlString"`
		CategoryParentList string `json:"categoryParentList"`
		*Alias
	}{
		Alias: (*Alias)(i),
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	i.StartTime = inferTime(tmp.StartTimeRaw)
	i.EndTime = inferTime(tmp.EndTimeRaw)

	if i.ImageURL == "" && tmp.ImageServer != "" {
		images := strings.Split(tmp.ImageURLString, ";")
		if len(images) > 0 {
			i.ImageURL = tmp.ImageServer + images[0]
		}
	}

	if i.CategoryName == "" && tmp.CategoryParentList != "" {
		split := strings.Split(tmp.CategoryParentList, "|")
		categories := make([]string, 0, len(split)/2)
		for i := 1; i < len(split); i += 2 {
			categories = append(categories, split[i])
		}

		i.CategoryName = categories[len(categories)-1]
		i.CategoryFullName = strings.Join(categories, " > ")
	}

	return nil
}

func (i *Item) NewCreateItemParams(sub sqlgen.Subscription) sqlgen.CreateItemParams {
	return sqlgen.CreateItemParams{
		ID:             db.NewID(),
		GoodwillID:     i.ItemID,
		SubscriptionID: sub.ID,
		StartedAt:      i.StartTime,
		EndsAt:         i.EndTime,
	}
}

func inferTime(raw string) time.Time {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		slog.Error("failed to load location", "err", err)
		return time.Time{}
	}

	ts, err := time.ParseInLocation("2006-01-02T15:04:05", raw, loc)
	if err != nil {
		slog.Error("failed to parse time", "raw", raw, "err", err)
		return time.Time{}
	}

	return ts.UTC()
}
