package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/geoff-phan/indexer/api/generated/v2"
	"github.com/geoff-phan/indexer/idb"
	"github.com/geoff-phan/indexer/idb/mocks"
)

func TestTransactionParamToTransactionFilter(t *testing.T) {
	tests := []struct {
		name          string
		params        generated.SearchForTransactionsParams
		filter        idb.TransactionFilter
		errorContains []string
	}{
		{
			"Default",
			generated.SearchForTransactionsParams{},
			idb.TransactionFilter{Limit: defaultTransactionsLimit},
			nil,
		},
		{
			"Limit",
			generated.SearchForTransactionsParams{Limit: uint64Ptr(defaultTransactionsLimit + 10)},
			idb.TransactionFilter{Limit: defaultTransactionsLimit + 10},
			nil,
		},
		{
			"Limit Max",
			generated.SearchForTransactionsParams{Limit: uint64Ptr(maxTransactionsLimit + 10)},
			idb.TransactionFilter{Limit: maxTransactionsLimit},
			nil,
		},
		{
			"Int field",
			generated.SearchForTransactionsParams{AssetId: uint64Ptr(1234)},
			idb.TransactionFilter{AssetID: 1234, Limit: defaultTransactionsLimit},
			nil,
		},
		{
			"Pointer field",
			generated.SearchForTransactionsParams{Round: uint64Ptr(1234)},
			idb.TransactionFilter{Round: uint64Ptr(1234), Limit: defaultTransactionsLimit},
			nil,
		},
		{
			"Base64 field",
			generated.SearchForTransactionsParams{NotePrefix: strPtr(base64.StdEncoding.EncodeToString([]byte("SomeData")))},
			idb.TransactionFilter{NotePrefix: []byte("SomeData"), Limit: defaultTransactionsLimit},
			nil,
		},
		{
			"Enum fields",
			generated.SearchForTransactionsParams{TxType: strPtr("pay"), SigType: strPtr("lsig")},
			idb.TransactionFilter{TypeEnum: 1, SigType: "lsig", Limit: defaultTransactionsLimit},
			nil,
		},
		{
			"Date time fields",
			generated.SearchForTransactionsParams{AfterTime: timePtr(time.Date(2020, 3, 4, 12, 0, 0, 0, time.FixedZone("UTC", 0)))},
			idb.TransactionFilter{AfterTime: time.Date(2020, 3, 4, 12, 0, 0, 0, time.FixedZone("UTC", 0)), Limit: defaultTransactionsLimit},
			nil,
		},
		{
			"Invalid Enum fields",
			generated.SearchForTransactionsParams{TxType: strPtr("micro"), SigType: strPtr("handshake")},
			idb.TransactionFilter{},
			[]string{errUnknownSigType, errUnknownTxType},
		},
		{
			"As many fields as possible",
			generated.SearchForTransactionsParams{
				Limit:               uint64Ptr(defaultTransactionsLimit + 1),
				Next:                strPtr("next-token"),
				NotePrefix:          strPtr(base64.StdEncoding.EncodeToString([]byte("custom-note"))),
				TxType:              strPtr("pay"),
				SigType:             strPtr("sig"),
				Txid:                strPtr("YXGBWVBK764KGYPX6ENIADKXPWLBNAZ7MTXDZULZWGOBO2W6IAR622VSLA"),
				Round:               nil,
				MinRound:            uint64Ptr(2),
				MaxRound:            uint64Ptr(3),
				AssetId:             uint64Ptr(4),
				BeforeTime:          timePtr(time.Date(2021, 1, 1, 1, 0, 0, 0, time.FixedZone("UTC", 0))),
				AfterTime:           timePtr(time.Date(2022, 2, 2, 2, 0, 0, 0, time.FixedZone("UTC", 0))),
				CurrencyGreaterThan: uint64Ptr(5),
				CurrencyLessThan:    uint64Ptr(6),
				Address:             strPtr("YXGBWVBK764KGYPX6ENIADKXPWLBNAZ7MTXDZULZWGOBO2W6IAR622VSLA"),
				AddressRole:         strPtr("sender"),
				ExcludeCloseTo:      boolPtr(true),
				ApplicationId:       uint64Ptr(7),
			},
			idb.TransactionFilter{
				Limit:             defaultTransactionsLimit + 1,
				NextToken:         "next-token",
				NotePrefix:        []byte("custom-note"),
				TypeEnum:          1,
				SigType:           "sig",
				Txid:              "YXGBWVBK764KGYPX6ENIADKXPWLBNAZ7MTXDZULZWGOBO2W6IAR622VSLA",
				Round:             nil,
				MinRound:          2,
				MaxRound:          3,
				AssetID:           4,
				BeforeTime:        time.Date(2021, 1, 1, 1, 0, 0, 0, time.FixedZone("UTC", 0)),
				AfterTime:         time.Date(2022, 2, 2, 2, 0, 0, 0, time.FixedZone("UTC", 0)),
				AlgosGT:           nil,
				AlgosLT:           nil,
				AssetAmountGT:     uint64Ptr(5),
				AssetAmountLT:     uint64Ptr(6),
				EffectiveAmountGT: nil,
				EffectiveAmountLT: nil,
				Address:           []byte{197, 204, 27, 84, 42, 255, 184, 163, 97, 247, 241, 26, 128, 13, 87, 125, 150, 22, 131, 63, 100, 238, 60, 209, 121, 177, 156, 23, 106, 222, 64, 35},
				AddressRole:       9,
				Offset:            nil,
				OffsetLT:          nil,
				OffsetGT:          nil,
				ApplicationID:     7,
			},
			nil,
		},
		{
			name: "Round + Min/Max Error",
			params: generated.SearchForTransactionsParams{
				Round:    uint64Ptr(10),
				MinRound: uint64Ptr(5),
				MaxRound: uint64Ptr(15),
			},
			filter:        idb.TransactionFilter{},
			errorContains: []string{errInvalidRoundAndMinMax},
		},
		{
			name:          "Swapped Min/Max Round",
			params:        generated.SearchForTransactionsParams{MinRound: uint64Ptr(20), MaxRound: uint64Ptr(10)},
			filter:        idb.TransactionFilter{},
			errorContains: []string{errInvalidRoundMinMax},
		},
		{
			name:          "Illegal Address",
			params:        generated.SearchForTransactionsParams{Address: strPtr("Not-our-base32-thing")},
			filter:        idb.TransactionFilter{},
			errorContains: []string{errUnableToParseAddress},
		},
		{
			name:          "Unknown address role error",
			params:        generated.SearchForTransactionsParams{AddressRole: strPtr("unknown")},
			filter:        idb.TransactionFilter{},
			errorContains: []string{errUnknownAddressRole},
		},
		{
			name:          "Bitmask sender + closeTo(true)",
			params:        generated.SearchForTransactionsParams{AddressRole: strPtr("sender"), ExcludeCloseTo: boolPtr(true)},
			filter:        idb.TransactionFilter{AddressRole: 9, Limit: defaultTransactionsLimit},
			errorContains: nil,
		},
		{
			name:          "Bitmask sender + closeTo(false)",
			params:        generated.SearchForTransactionsParams{AddressRole: strPtr("sender"), ExcludeCloseTo: boolPtr(false)},
			filter:        idb.TransactionFilter{AddressRole: 9, Limit: defaultTransactionsLimit},
			errorContains: nil,
		},
		{
			name:          "Bitmask receiver + closeTo(true)",
			params:        generated.SearchForTransactionsParams{AddressRole: strPtr("receiver"), ExcludeCloseTo: boolPtr(true)},
			filter:        idb.TransactionFilter{AddressRole: 18, Limit: defaultTransactionsLimit},
			errorContains: nil,
		},
		{
			name:          "Bitmask receiver + closeTo(false)",
			params:        generated.SearchForTransactionsParams{AddressRole: strPtr("receiver"), ExcludeCloseTo: boolPtr(false)},
			filter:        idb.TransactionFilter{AddressRole: 54, Limit: defaultTransactionsLimit},
			errorContains: nil,
		},
		{
			name:          "Bitmask receiver + implicit closeTo (false)",
			params:        generated.SearchForTransactionsParams{AddressRole: strPtr("receiver")},
			filter:        idb.TransactionFilter{AddressRole: 54, Limit: defaultTransactionsLimit},
			errorContains: nil,
		},
		{
			name:          "Bitmask freeze-target",
			params:        generated.SearchForTransactionsParams{AddressRole: strPtr("freeze-target")},
			filter:        idb.TransactionFilter{AddressRole: 64, Limit: defaultTransactionsLimit},
			errorContains: nil,
		},
		{
			name:          "Currency to Algos when no asset-id",
			params:        generated.SearchForTransactionsParams{CurrencyGreaterThan: uint64Ptr(10), CurrencyLessThan: uint64Ptr(20)},
			filter:        idb.TransactionFilter{AlgosGT: uint64Ptr(10), AlgosLT: uint64Ptr(20), Limit: defaultTransactionsLimit},
			errorContains: nil,
		},
		{
			name:          "Searching by application-id",
			params:        generated.SearchForTransactionsParams{ApplicationId: uint64Ptr(1234)},
			filter:        idb.TransactionFilter{ApplicationID: 1234, Limit: defaultTransactionsLimit},
			errorContains: nil,
		},
	}

	for _, test := range tests {
		//test := test
		t.Run(test.name, func(t *testing.T) {
			//t.Parallel()
			filter, err := transactionParamsToTransactionFilter(test.params)
			if test.errorContains != nil {
				for _, msg := range test.errorContains {
					assert.Contains(t, err.Error(), msg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.errorContains != nil, err != nil)
				assert.Equal(t, test.filter, filter)
			}
		})
	}
}

func loadResourceFileOrPanic(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to load resource file: '%s'", path))
	}
	var ret idb.TxnRow
	_ = msgpack.Decode(data, &ret)
	return data
}

func loadTransactionFromFile(path string) generated.Transaction {
	data := loadResourceFileOrPanic(path)
	var ret generated.Transaction
	if err := json.Unmarshal(data, &ret); err != nil {
		panic(fmt.Sprintf("Failed to build transaction from file: %s", path))
	}
	return ret
}

func TestFetchTransactions(t *testing.T) {
	// Add in txnRows (with TxnBytes to parse), verify that they are properly serialized to generated.TransactionResponse
	tests := []struct {
		name     string
		txnBytes [][]byte
		response []generated.Transaction
		created  uint64
	}{
		{
			name: "Payment",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/payment.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/payment.response"),
			},
		},
		{
			name: "Key Registration",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/keyreg.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/keyreg.response"),
			},
		},
		{
			name: "Asset Configuration",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/asset_config.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/asset_config.response"),
			},
			created: 100,
		},
		{
			name: "Asset Transfer",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/asset_transfer.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/asset_transfer.response"),
			},
		},
		{
			name: "Asset Freeze",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/asset_freeze.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/asset_freeze.response"),
			},
		},
		{
			name: "Multisig Transaction",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/multisig.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/multisig.response"),
			},
		},
		{
			name: "Rekey Transaction",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/rekey.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/rekey.response"),
			},
		},
		{
			name: "Application Call (1)",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_call_1.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_call_1.response"),
			},
			created: 10,
		},
		{
			name: "Application Call (2)",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_call_2.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_call_2.response"),
			},
			created: 10,
		},
		{
			name: "Application Call (3)",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_call_3.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_call_3.response"),
			},
			created: 10,
		},
		{
			name: "Application Clear",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_clear.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_clear.response"),
			},
		},
		{
			name: "Application Close",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_close.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_close.response"),
			},
		},
		{
			name: "Application Update",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_update.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_update.response"),
			},
		},
		{
			name: "Application Delete",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_delete.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_delete.response"),
			},
		},
		{
			name: "Application Non ASCII Key",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_nonascii.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_nonascii.response"),
			},
		},
		{
			name: "Application Optin",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_optin.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_optin.response"),
			},
		},
		{
			name: "Application With Foreign App",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_foreign.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_foreign.response"),
			},
		},
		{
			name: "Application With Foreign Assets",
			txnBytes: [][]byte{
				loadResourceFileOrPanic("test_resources/app_foreign_assets.txn"),
			},
			response: []generated.Transaction{
				loadTransactionFromFile("test_resources/app_foreign_assets.response"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup the mocked responses

			mockIndexer := &mocks.IndexerDb{}
			si := ServerImplementation{
				EnableAddressSearchRoundRewind: true,
				db:                             mockIndexer,
			}

			roundTime := time.Now()
			roundTime64 := uint64(roundTime.Unix())

			ch := make(chan idb.TxnRow, len(test.txnBytes))
			for _, bytes := range test.txnBytes {
				txnRow := idb.TxnRow{
					Round:     1,
					Intra:     2,
					RoundTime: roundTime,
					TxnBytes:  bytes,
					AssetID:   test.created,
					Extra: idb.TxnExtra{
						AssetCloseAmount: 0,
					},
					Error: nil,
				}
				ch <- txnRow
			}

			close(ch)
			var outCh <-chan idb.TxnRow = ch
			var round uint64 = 1
			mockIndexer.On("Transactions", mock.Anything, mock.Anything).Return(outCh, round)

			// Call the function
			results, _, _, err := si.fetchTransactions(context.Background(), idb.TransactionFilter{})
			assert.NoError(t, err)

			// Automatically print it out when writing the test.
			printIt := len(test.response) == 0
			if printIt {
				fmt.Printf("Test: %s\n", test.name)
				for _, result := range results {
					fmt.Println("-------------------")
					str, _ := json.Marshal(result)
					fmt.Printf("%s\n", str)
				}
				fmt.Println("-------------------")
			}

			// Verify the results
			assert.Equal(t, len(test.response), len(results))
			for i, expected := range test.response {
				actual := results[i]
				// This is set in the mock above, so override it in the expected value.
				expected.RoundTime = &roundTime64
				fmt.Println(roundTime64)
				assert.EqualValues(t, expected, actual)
			}
		})
	}
}

func TestFetchAccountsRewindRoundTooLarge(t *testing.T) {
	ch := make(chan idb.AccountRow)
	close(ch)
	var outCh <-chan idb.AccountRow = ch

	db := &mocks.IndexerDb{}
	db.On("GetAccounts", mock.Anything, mock.Anything).Return(outCh, uint64(7)).Once()

	si := ServerImplementation{
		EnableAddressSearchRoundRewind: true,
		db:                             db,
	}
	atRound := uint64(8)
	_, _, err := si.fetchAccounts(context.Background(), idb.AccountQueryOptions{}, &atRound)
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), errRewindingAccount), err.Error())
}
