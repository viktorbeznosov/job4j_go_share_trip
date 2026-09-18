package service

//go:generate mockgen -source=contract_client.go -destination=mocks/contract_client.go -package=mocks

import (
	"context"

	"job4j_go_share_trip/internal/business/trip/entity"
	contractclient "job4j_go_share_trip/internal/clients/contract"
)

type ContractClient interface {
	CheckService(
		ctx context.Context,
		companyID string,
		serviceCode entity.ServiceType,
	) (contractclient.CheckResult, error)
}