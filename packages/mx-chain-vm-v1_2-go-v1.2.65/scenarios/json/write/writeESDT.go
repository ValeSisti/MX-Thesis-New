package scenjsonwrite

import (
	mj "github.com/multiversx/mx-chain-vm-v1_2-go/scenarios/json/model"
	oj "github.com/multiversx/mx-chain-vm-v1_2-go/scenarios/orderedjson"
)

func esdtTxDataToOJ(esdtItem *mj.ESDTTxData) *oj.OJsonMap {
	esdtItemOJ := oj.NewMap()
	if len(esdtItem.TokenIdentifier.Original) > 0 {
		esdtItemOJ.Put("tokenIdentifier", bytesFromStringToOJ(esdtItem.TokenIdentifier))
	}
	if len(esdtItem.Nonce.Original) > 0 {
		esdtItemOJ.Put("nonce", uint64ToOJ(esdtItem.Nonce))
	}
	if len(esdtItem.Value.Original) > 0 {
		esdtItemOJ.Put("value", bigIntToOJ(esdtItem.Value))
	}
	return esdtItemOJ
}

func esdtDataToOJ(esdtItems []*mj.ESDTData) *oj.OJsonMap {
	esdtItemsOJ := oj.NewMap()
	for _, esdtItem := range esdtItems {
		esdtItemsOJ.Put(esdtItem.TokenIdentifier.Original, esdtItemToOJ(esdtItem))
	}
	return esdtItemsOJ
}

func esdtItemToOJ(esdtItem *mj.ESDTData) oj.OJsonObject {
	if isCompactESDT(esdtItem) {
		return bigIntToOJ(esdtItem.Instances[0].Balance)
	}

	esdtItemOJ := oj.NewMap()

	// instances
	if len(esdtItem.Instances) == 1 {
		appendESDTInstanceToOJ(esdtItem.Instances[0], esdtItemOJ)
	} else {
		var convertedList []oj.OJsonObject
		for _, esdtInstance := range esdtItem.Instances {
			esdtInstanceOJ := oj.NewMap()
			appendESDTInstanceToOJ(esdtInstance, esdtInstanceOJ)
			convertedList = append(convertedList, esdtInstanceOJ)
		}
		instancesOJList := oj.OJsonList(convertedList)
		esdtItemOJ.Put("instances", &instancesOJList)
	}

	if len(esdtItem.LastNonce.Original) > 0 {
		esdtItemOJ.Put("lastNonce", uint64ToOJ(esdtItem.LastNonce))
	}

	// roles
	if len(esdtItem.Roles) > 0 {
		var convertedList []oj.OJsonObject
		for _, roleStr := range esdtItem.Roles {
			convertedList = append(convertedList, &oj.OJsonString{Value: roleStr})
		}
		rolesOJList := oj.OJsonList(convertedList)
		esdtItemOJ.Put("roles", &rolesOJList)
	}
	if len(esdtItem.Frozen.Original) > 0 {
		esdtItemOJ.Put("frozen", uint64ToOJ(esdtItem.Frozen))
	}

	return esdtItemOJ
}

func appendESDTInstanceToOJ(esdtInstance *mj.ESDTInstance, targetOj *oj.OJsonMap) {
	if len(esdtInstance.Nonce.Original) > 0 {
		targetOj.Put("nonce", uint64ToOJ(esdtInstance.Nonce))
	}
	if len(esdtInstance.Balance.Original) > 0 {
		targetOj.Put("balance", bigIntToOJ(esdtInstance.Balance))
	}
}

func isCompactESDT(esdtItem *mj.ESDTData) bool {
	if len(esdtItem.Instances) != 1 {
		return false
	}
	if len(esdtItem.Instances[0].Nonce.Original) > 0 {
		return false
	}
	if len(esdtItem.Roles) > 0 {
		return false
	}
	if len(esdtItem.Frozen.Original) > 0 {
		return false
	}
	return true
}
