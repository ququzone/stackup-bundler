package simulation

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stackup-wallet/stackup-bundler/pkg/entrypoint"
	"github.com/stackup-wallet/stackup-bundler/pkg/entrypoint/reverts"
	"github.com/stackup-wallet/stackup-bundler/pkg/errors"
	"github.com/stackup-wallet/stackup-bundler/pkg/userop"
)

type revertError struct {
	reason string // revert reason hex encoded
}

func (e *revertError) Error() string {
	return "execution reverted"
}

func (e *revertError) ErrorData() interface{} {
	return e.reason
}

// SimulateValidation makes a static call to Entrypoint.simulateValidation(userop) and returns the
// results without any state changes.
func SimulateValidation(
	rpc *rpc.Client,
	entryPoint common.Address,
	op *userop.UserOperation,
) (*reverts.ValidationResultRevert, error) {
	parsedABI, err := abi.JSON(strings.NewReader(entrypoint.EntrypointABI))
	if err != nil {
		return nil, err
	}
	input, err := parsedABI.Pack("simulateValidation", entrypoint.UserOperation(*op))
	if err != nil {
		return nil, err
	}

	client := ethclient.NewClient(rpc)
	data, err := client.CallContract(
		context.Background(),
		ethereum.CallMsg{
			From: common.BigToAddress(common.Big0),
			To:   &entryPoint,
			Data: input,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}
	err = &revertError{
		reason: hexutil.Encode(data),
	}

	sim, simErr := reverts.NewValidationResult(err)
	if simErr != nil {
		fo, foErr := reverts.NewFailedOp(err)
		if foErr != nil {
			return nil, fmt.Errorf("%s, %s", simErr, foErr)
		}
		return nil, errors.NewRPCError(errors.REJECTED_BY_EP_OR_ACCOUNT, fo.Reason, fo)
	}

	return sim, nil
}
