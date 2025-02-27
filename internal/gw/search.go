package gw

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/robherley/gw-bot/internal/db/sqlgen"
)

type SearchOption func(map[string]any)

func WithMinPrice(min int64) SearchOption {
	return func(q map[string]any) {
		q["lowPrice"] = strconv.FormatInt(min, 10)
	}
}

func WithMaxPrice(max int64) SearchOption {
	return func(q map[string]any) {
		q["highPrice"] = strconv.FormatInt(max, 10)
	}
}

func WithDescending(descending bool) SearchOption {
	return func(q map[string]any) {
		q["sortDescending"] = strconv.FormatBool(descending)
	}
}

func WithPageSize(size int) SearchOption {
	return func(q map[string]any) {
		q["pageSize"] = strconv.Itoa(size)
	}
}

func SearchOptionsFromSubscription(sub sqlgen.Subscription) []SearchOption {
	opts := make([]SearchOption, 0)
	if sub.MinPrice != nil {
		opts = append(opts, WithMinPrice(*sub.MinPrice))
	}

	if sub.MaxPrice != nil {
		opts = append(opts, WithMaxPrice(*sub.MaxPrice))
	}

	return opts
}

func NewSearchQuery(term string, opts ...SearchOption) ([]byte, error) {
	query := map[string]any{
		"isSize":                          false,
		"isWeddingCatagory":               "false",
		"isMultipleCategoryIds":           false,
		"isFromHeaderMenuTab":             false,
		"layout":                          "",
		"isFromHomePage":                  false,
		"searchText":                      term,
		"selectedGroup":                   "",
		"selectedCategoryIds":             "",
		"selectedSellerIds":               "",
		"lowPrice":                        "0",
		"highPrice":                       "999999",
		"searchBuyNowOnly":                "",
		"searchPickupOnly":                "false",
		"searchNoPickupOnly":              "false",
		"searchOneCentShippingOnly":       "false",
		"searchDescriptions":              "false",
		"searchClosedAuctions":            "false",
		"closedAuctionEndingDate":         time.Now().Format("1/2/2006"),
		"closedAuctionDaysBack":           "7",
		"searchCanadaShipping":            "false",
		"searchInternationalShippingOnly": "false",
		"sortColumn":                      "1",
		"page":                            "1",
		"pageSize":                        "100",
		"sortDescending":                  "true",
		"savedSearchId":                   0,
		"useBuyerPrefs":                   "true",
		"searchUSOnlyShipping":            "true",
		"categoryLevelNo":                 "1",
		"categoryLevel":                   1,
		"categoryId":                      0,
		"partNumber":                      "",
		"catIds":                          "",
	}

	for _, opt := range opts {
		opt(query)
	}

	return json.Marshal(&query)
}
