package hostCore

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	worldmock "github.com/multiversx/mx-chain-vm-v1_2-go/mock/world"
	"github.com/multiversx/mx-chain-vm-v1_2-go/vmhost"
	"github.com/stretchr/testify/require"
)

var ESDTTransferGasCost = uint64(1)
var ESDTTestTokenName = []byte("TT")

func TestExecution_ExecuteOnDestContext_ESDTTransferWithoutExecute(t *testing.T) {
	code := GetTestSCCodeModule("exec-dest-ctx-esdt/basic", "basic", "../../")
	scBalance := big.NewInt(1000)
	host, world := defaultTestVMForCallWithWorldMock(t, code, scBalance)

	tokenKey := worldmock.MakeTokenKey(ESDTTestTokenName, 0)
	err := world.BuiltinFuncs.SetTokenData(parentAddress, tokenKey, &esdt.ESDigitalToken{
		Value: big.NewInt(100),
		Type:  uint32(core.Fungible),
	})
	require.Nil(t, err)

	input := DefaultTestContractCallInput()
	input.Function = "basic_transfer"
	input.GasProvided = 100000
	input.ESDTTransfers = make([]*vmcommon.ESDTTransfer, 1)
	input.ESDTTransfers[0] = &vmcommon.ESDTTransfer{}
	input.ESDTTransfers[0].ESDTValue = big.NewInt(16)
	input.ESDTTransfers[0].ESDTTokenName = ESDTTestTokenName

	vmOutput, err := host.RunSmartContractCall(input)
	require.Nil(t, err)

	require.NotNil(t, vmOutput)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	require.Equal(t, "", vmOutput.ReturnMessage)
}

func TestExecution_ExecuteOnDestContext_MockBuiltinFunctions_Claim(t *testing.T) {
	code := GetTestSCCode("exec-dest-ctx-builtin", "../../")
	scBalance := big.NewInt(1000)

	host, stubBlockchainHook := defaultTestVMForCall(t, code, scBalance)
	stubBlockchainHook.ProcessBuiltInFunctionCalled = dummyProcessBuiltInFunction
	host.protocolBuiltinFunctions = getDummyBuiltinFunctionNames()

	input := DefaultTestContractCallInput()
	input.RecipientAddr = parentAddress

	input.Function = "callBuiltinClaim"
	input.GasProvided = 100000

	vmOutput, err := host.RunSmartContractCall(input)
	require.Nil(t, err)

	require.NotNil(t, vmOutput)
	expectedVMOutput := expectedVMOutputDestCtxBuiltinClaim(input, code)
	require.Equal(t, expectedVMOutput, vmOutput)
}

func TestExecution_ExecuteOnDestContext_MockBuiltinFunctions_DoSomething(t *testing.T) {
	code := GetTestSCCode("exec-dest-ctx-builtin", "../../")
	scBalance := big.NewInt(1000)

	host, stubBlockchainHook := defaultTestVMForCall(t, code, scBalance)
	stubBlockchainHook.ProcessBuiltInFunctionCalled = dummyProcessBuiltInFunction
	host.protocolBuiltinFunctions = getDummyBuiltinFunctionNames()

	input := DefaultTestContractCallInput()
	input.RecipientAddr = parentAddress

	input.Function = "callBuiltinDoSomething"
	input.GasProvided = 100000

	vmOutput, err := host.RunSmartContractCall(input)
	require.Nil(t, err)

	require.NotNil(t, vmOutput)
	expectedVMOutput := expectedVMOutputDestCtxBuiltinDoSomething(input, code)
	require.Equal(t, expectedVMOutput, vmOutput)
}

func TestExecution_ExecuteOnDestContext_MockBuiltinFunctions_Nonexistent(t *testing.T) {
	code := GetTestSCCode("exec-dest-ctx-builtin", "../../")
	scBalance := big.NewInt(1000)

	host, stubBlockchainHook := defaultTestVMForCall(t, code, scBalance)
	stubBlockchainHook.ProcessBuiltInFunctionCalled = dummyProcessBuiltInFunction
	host.protocolBuiltinFunctions = getDummyBuiltinFunctionNames()

	input := DefaultTestContractCallInput()
	input.RecipientAddr = parentAddress

	input.Function = "callNonexistingBuiltin"
	input.GasProvided = 100000

	vmOutput, err := host.RunSmartContractCall(input)
	require.Nil(t, err)

	require.NotNil(t, vmOutput)
	expectedVMOutput := expectedVMOutputDestCtxBuiltinNonexistent(input, code)
	require.Equal(t, expectedVMOutput, vmOutput)
}

func TestExecution_ExecuteOnDestContext_MockBuiltinFunctions_Fail(t *testing.T) {
	code := GetTestSCCode("exec-dest-ctx-builtin", "../../")
	scBalance := big.NewInt(1000)

	host, stubBlockchainHook := defaultTestVMForCall(t, code, scBalance)
	stubBlockchainHook.ProcessBuiltInFunctionCalled = dummyProcessBuiltInFunction
	host.protocolBuiltinFunctions = getDummyBuiltinFunctionNames()

	input := DefaultTestContractCallInput()
	input.RecipientAddr = parentAddress

	input.Function = "callBuiltinFail"
	input.GasProvided = 100000

	vmOutput, err := host.RunSmartContractCall(input)
	require.Nil(t, err)

	require.NotNil(t, vmOutput)
	expectedVMOutput := expectedVMOutputDestCtxBuiltinFail(input, code)
	require.Equal(t, expectedVMOutput, vmOutput)
}

func TestExecution_AsyncCall_MockBuiltinFails(t *testing.T) {
	code := GetTestSCCode("async-call-builtin", "../../")
	scBalance := big.NewInt(1000)

	host, stubBlockchainHook := defaultTestVMForCall(t, code, scBalance)
	stubBlockchainHook.ProcessBuiltInFunctionCalled = dummyProcessBuiltInFunction
	host.protocolBuiltinFunctions = getDummyBuiltinFunctionNames()

	input := DefaultTestContractCallInput()
	input.RecipientAddr = parentAddress
	input.Function = "performAsyncCallToBuiltin"
	input.Arguments = [][]byte{{1}}
	input.GasProvided = 1000000

	vmOutput, err := host.RunSmartContractCall(input)
	require.Nil(t, err)

	require.NotNil(t, vmOutput)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	require.Equal(t, [][]byte{[]byte("hello"), {10}}, vmOutput.ReturnData)
}

func TestESDT_GettersAPI(t *testing.T) {
	code := GetTestSCCode("exchange", "../../")
	scBalance := big.NewInt(1000)

	host, _ := defaultTestVMForCall(t, code, scBalance)

	input := DefaultTestContractCallInput()
	input.RecipientAddr = parentAddress
	input.Function = "validateGetters"
	input.GasProvided = 1000000
	input.ESDTTransfers = make([]*vmcommon.ESDTTransfer, 1)
	input.ESDTTransfers[0] = &vmcommon.ESDTTransfer{}
	input.ESDTTransfers[0].ESDTValue = big.NewInt(5)
	input.ESDTTransfers[0].ESDTTokenName = ESDTTestTokenName

	vmOutput, err := host.RunSmartContractCall(input)
	require.Nil(t, err)

	require.NotNil(t, vmOutput)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
}

func TestESDT_GettersAPI_ExecuteAfterBuiltinCall(t *testing.T) {
	dummyCode := GetTestSCCode("init-simple", "../../")
	exchangeCode := GetTestSCCode("exchange", "../../")
	scBalance := big.NewInt(1000)
	esdtValue := int64(5)

	host, stubBlockchainHook := defaultTestVMForCall(t, exchangeCode, scBalance)
	stubBlockchainHook.ProcessBuiltInFunctionCalled = dummyProcessBuiltInFunction
	host.protocolBuiltinFunctions = getDummyBuiltinFunctionNames()

	input := DefaultTestContractCallInput()
	err := host.Output().TransferValueOnly(input.RecipientAddr, input.CallerAddr, input.CallValue, false)
	require.Nil(t, err)

	input.RecipientAddr = parentAddress
	input.Function = core.BuiltInFunctionESDTTransfer
	input.GasProvided = 1000000
	input.Arguments = [][]byte{
		ESDTTestTokenName,
		big.NewInt(esdtValue).Bytes(),
		[]byte("validateGetters"),
	}

	host.InitState()

	_ = host.Runtime().StartWasmerInstance(dummyCode, input.GasProvided, true)
	vmOutput, asyncInfo, _, err := host.ExecuteOnDestContext(input)

	require.Nil(t, err)
	require.NotNil(t, vmOutput)

	require.Zero(t, len(asyncInfo.AsyncContextMap))

	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	host.Clean()
}

func dummyProcessBuiltInFunction(input *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	outPutAccounts := make(map[string]*vmcommon.OutputAccount)
	outPutAccounts[string(parentAddress)] = &vmcommon.OutputAccount{
		BalanceDelta: big.NewInt(0),
		Address:      parentAddress}

	if input.Function == "builtinClaim" {
		outPutAccounts[string(parentAddress)].BalanceDelta = big.NewInt(42)
		return &vmcommon.VMOutput{
			GasRemaining:   400 + input.GasLocked,
			OutputAccounts: outPutAccounts,
		}, nil
	}
	if input.Function == "builtinDoSomething" {
		return &vmcommon.VMOutput{
			GasRemaining:   400 + input.GasLocked,
			OutputAccounts: outPutAccounts,
		}, nil
	}
	if input.Function == "builtinFail" {
		return nil, errors.New("whatdidyoudo")
	}
	if input.Function == core.BuiltInFunctionESDTTransfer {
		vmOutput := &vmcommon.VMOutput{
			GasRemaining: 0,
		}
		function := string(input.Arguments[2])
		esdtTransferTxData := function
		for _, arg := range input.Arguments[3:] {
			esdtTransferTxData += "@" + hex.EncodeToString(arg)
		}
		outTransfer := vmcommon.OutputTransfer{
			Value:         big.NewInt(0),
			GasLimit:      input.GasProvided - ESDTTransferGasCost + input.GasLocked,
			Data:          []byte(esdtTransferTxData),
			CallType:      vm.AsynchronousCall,
			SenderAddress: input.CallerAddr,
		}
		vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
		vmOutput.OutputAccounts[string(input.RecipientAddr)] = &vmcommon.OutputAccount{
			Address:         input.RecipientAddr,
			OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
		}
		// TODO when ESDT token balance querying is implemented, ensure the
		// transfers that happen here are persisted in the mock accounts
		return vmOutput, nil
	}

	return nil, vmhost.ErrFuncNotFound
}

func getDummyBuiltinFunctionNames() vmcommon.FunctionNames {
	names := make(vmcommon.FunctionNames)

	var empty struct{}
	names["builtinClaim"] = empty
	names["builtinDoSomething"] = empty
	names["builtinFail"] = empty
	names[core.BuiltInFunctionESDTTransfer] = empty

	return names
}
