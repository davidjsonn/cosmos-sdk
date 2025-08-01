package params_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstestutil "github.com/cosmos/cosmos-sdk/x/params/testutil"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// StakingKeeper defines the expected staking keeper
type StakingKeeper interface {
	MaxValidators(ctx context.Context) (res uint32, err error)
}

type HandlerTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	govHandler    govv1beta1.Handler
	stakingKeeper StakingKeeper
}

func (suite *HandlerTestSuite) SetupTest() {
	encodingCfg := moduletestutil.MakeTestEncodingConfig(params.AppModuleBasic{})
	key := storetypes.NewKVStoreKey(paramtypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey("params_transient_test")

	ctx := testutil.DefaultContext(key, tkey)
	paramsKeeper := keeper.NewKeeper(encodingCfg.Codec, encodingCfg.Amino, key, tkey)
	paramsKeeper.Subspace("staking").WithKeyTable(stakingtypes.ParamKeyTable()) //nolint:staticcheck // TODO: deprecate this test case
	ctrl := gomock.NewController(suite.T())
	stakingKeeper := paramstestutil.NewMockStakingKeeper(ctrl)
	stakingKeeper.EXPECT().MaxValidators(ctx).Return(uint32(1), nil)

	suite.govHandler = params.NewParamChangeProposalHandler(paramsKeeper)
	suite.stakingKeeper = stakingKeeper
	suite.ctx = ctx
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func testProposal(changes ...proposal.ParamChange) *proposal.ParameterChangeProposal {
	return proposal.NewParameterChangeProposal("title", "description", changes)
}

func (suite *HandlerTestSuite) TestProposalHandler() {
	testCases := []struct {
		name     string
		proposal *proposal.ParameterChangeProposal
		onHandle func()
		expErr   bool
	}{
		{
			"all fields",
			testProposal(proposal.NewParamChange(stakingtypes.ModuleName, string(stakingtypes.KeyMaxValidators), "1")),
			func() {
				maxVals, err := suite.stakingKeeper.MaxValidators(suite.ctx)
				suite.Require().NoError(err)
				suite.Require().Equal(uint32(1), maxVals)
			},
			false,
		},
		{
			"invalid type",
			testProposal(proposal.NewParamChange(stakingtypes.ModuleName, string(stakingtypes.KeyMaxValidators), "-")),
			func() {},
			true,
		},
		//{
		//	"omit empty fields",
		//	testProposal(proposal.ParamChange{
		//		Subspace: govtypes.ModuleName,
		//		Key:      string(govv1.ParamStoreKeyDepositParams),
		//		Value:    `{"min_deposit": [{"denom": "uatom","amount": "64000000"}], "max_deposit_period": "172800000000000"}`,
		//	}),
		//	func() {
		//		depositParams := suite.app.GovKeeper.GetDepositParams(suite.ctx)
		//		defaultPeriod := govv1.DefaultPeriod
		//		suite.Require().Equal(govv1.DepositParams{
		//			MinDeposit:       sdk.NewCoins(sdk.NewCoin("uatom", sdkmath.NewInt(64000000))),
		//			MaxDepositPeriod: &defaultPeriod,
		//		}, depositParams)
		//	},
		//	false,
		// },
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := suite.govHandler(suite.ctx, tc.proposal)
			if tc.expErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.onHandle()
			}
		})
	}
}
