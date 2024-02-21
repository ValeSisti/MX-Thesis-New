package scenjsonparse

import (
	"errors"
	"fmt"

	mj "github.com/multiversx/mx-chain-vm-v1_2-go/scenarios/json/model"
	oj "github.com/multiversx/mx-chain-vm-v1_2-go/scenarios/orderedjson"
)

func (p *Parser) processTxESDT(txEsdtRaw oj.OJsonObject) (*mj.ESDTTxData, error) {
	fieldMap, isMap := txEsdtRaw.(*oj.OJsonMap)
	if !isMap {
		return nil, errors.New("unmarshalled account object is not a map")
	}

	esdtData := mj.ESDTTxData{}
	var err error

	for _, kvp := range fieldMap.OrderedKV {
		switch kvp.Key {
		case "tokenIdentifier":
			esdtData.TokenIdentifier, err = p.processStringAsByteArray(kvp.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid ESDT token name: %w", err)
			}
		case "nonce":
			esdtData.Nonce, err = p.processUint64(kvp.Value)
			if err != nil {
				return nil, errors.New("invalid account nonce")
			}
		case "value":
			esdtData.Value, err = p.processBigInt(kvp.Value, bigIntUnsignedBytes)
			if err != nil {
				return nil, fmt.Errorf("invalid ESDT balance: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown transaction ESDT data field: %s", kvp.Key)
		}
	}

	return &esdtData, nil
}
