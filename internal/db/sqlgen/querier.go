// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package sqlgen

import (
	"context"
)

type Querier interface {
	CreateItem(ctx context.Context, arg CreateItemParams) (Item, error)
	CreateSubscription(ctx context.Context, arg CreateSubscriptionParams) (Subscription, error)
	DeleteExpiredItems(ctx context.Context) error
	DeleteItemsInSubscriptions(ctx context.Context, ids []string) error
	DeleteUserSubscriptions(ctx context.Context, arg DeleteUserSubscriptionsParams) error
	FindItemsEndingSoon(ctx context.Context) ([]Item, error)
	FindSubscription(ctx context.Context, id string) (Subscription, error)
	FindSubscriptionsToNotify(ctx context.Context) ([]Subscription, error)
	FindUserSubscriptions(ctx context.Context, userID string) ([]Subscription, error)
	IsItemTracked(ctx context.Context, arg IsItemTrackedParams) (int64, error)
	SetItemSentFinal(ctx context.Context, ids []string) error
	SetSubscriptionLastNotifiedAt(ctx context.Context, id string) error
}

var _ Querier = (*Queries)(nil)
