package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/project-agonyl/agonyl-utils-go/a3presentfile"
	"github.com/project-agonyl/agonyl-utils-go/cashitemfile"
	"github.com/project-agonyl/agonyl-utils-go/derbygiftfile"
	"github.com/project-agonyl/agonyl-utils-go/eventitemrewardfile"
	"github.com/project-agonyl/agonyl-utils-go/lotteryfile"
	"github.com/project-agonyl/agonyl-utils-go/petfile"
	"github.com/project-agonyl/agonyl-utils-go/presentitemsetfile"
	"github.com/project-agonyl/agonyl-utils-go/setitemfile"
	"github.com/project-agonyl/agonyl-utils-go/shuecombinationfile"
)

func applyEconomyZoneData(original []byte, format ZoneDataFormat, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	switch format {
	case ZoneDataFormatCashItem:
		return applyCashItems(original, operations, allowed)
	case ZoneDataFormatSetItem:
		return applySetItems(original, operations, allowed)
	case ZoneDataFormatPresentItemSet:
		return applyPresentItemSets(original, operations, allowed)
	case ZoneDataFormatPet:
		return applyPets(original, operations, allowed)
	case ZoneDataFormatShueCombination:
		return applyShueCombinations(original, operations, allowed)
	case ZoneDataFormatLottery:
		return applyLottery(original, operations, allowed)
	case ZoneDataFormatDerbyGift:
		return applyDerbyGifts(original, operations, allowed)
	case ZoneDataFormatEventItemReward:
		return applyEventItemRewards(original, operations, allowed)
	case ZoneDataFormatA3Present:
		return applyA3Presents(original, operations, allowed)
	default:
		return nil, fmt.Errorf("unsupported economy ZoneData format %q", format)
	}
}

func applyCashItems(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := cashitemfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := validateRowOperation(operation, len(data)); err != nil {
			return nil, err
		}
		value, err := operationUint(operation, 4294967295)
		if err != nil {
			return nil, err
		}
		offset := operation.Row * cashitemfile.RecordSize
		switch operation.Field {
		case "npc_type":
			if value > 65535 {
				return nil, fmt.Errorf("npc_type exceeds 65535")
			}
			data[operation.Row].SetNPCType(uint16(value))
			allowBytes(allowed, offset, 2)
		case "item_code":
			if value > 65535 {
				return nil, fmt.Errorf("item_code exceeds 65535")
			}
			data[operation.Row].SetItemCode(uint16(value))
			allowBytes(allowed, offset+2, 2)
		case "price":
			data[operation.Row].SetPrice(uint32(value))
			allowBytes(allowed, offset+4, 4)
		case "count":
			if value > 65535 {
				return nil, fmt.Errorf("count exceeds 65535")
			}
			data[operation.Row].SetCount(uint16(value))
			allowBytes(allowed, offset+8, 2)
		default:
			return nil, fmt.Errorf("unknown cash item field %q", operation.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return cashitemfile.Write(w, data) })
}

func applySetItems(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := setitemfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := validateRowOperation(operation, len(data)); err != nil {
			return nil, err
		}
		value, err := operationUint(operation, 65535)
		if err != nil {
			return nil, err
		}
		parts := strings.Split(operation.Field, "_")
		recordOffset := operation.Row * setitemfile.RecordSize
		if len(parts) == 4 && parts[0] == "piece" {
			index, parseErr := strconv.Atoi(parts[1])
			if parseErr != nil || index < 0 || index >= setitemfile.PieceCount {
				return nil, fmt.Errorf("unknown set item field %q", operation.Field)
			}
			wearSlot, code, _ := data[operation.Row].Piece(index)
			switch strings.Join(parts[2:], "_") {
			case "wear_slot":
				if value > 255 {
					return nil, fmt.Errorf("wear slot exceeds 255")
				}
				wearSlot = byte(value)
				allowBytes(allowed, recordOffset+index*3, 1)
			case "code":
				code = uint16(value)
				allowBytes(allowed, recordOffset+index*3+1, 2)
			default:
				return nil, fmt.Errorf("unknown set item field %q", operation.Field)
			}
			if err := data[operation.Row].SetPiece(index, wearSlot, code); err != nil {
				return nil, err
			}
			continue
		}
		if len(parts) == 5 && parts[0] == "bonus" {
			pieceCount, firstErr := strconv.Atoi(parts[1])
			index, secondErr := strconv.Atoi(parts[2])
			if firstErr != nil || secondErr != nil {
				return nil, fmt.Errorf("unknown set item field %q", operation.Field)
			}
			optionID, currentValue, ok := data[operation.Row].Bonus(pieceCount, index)
			if !ok {
				return nil, fmt.Errorf("unknown set item field %q", operation.Field)
			}
			offset := recordOffset + 0x1e + (pieceCount-1)*0x1e + index*3
			switch strings.Join(parts[3:], "_") {
			case "option_id":
				if value > 255 {
					return nil, fmt.Errorf("option id exceeds 255")
				}
				optionID = byte(value)
				allowBytes(allowed, offset, 1)
			case "value":
				currentValue = uint16(value)
				allowBytes(allowed, offset+1, 2)
			default:
				return nil, fmt.Errorf("unknown set item field %q", operation.Field)
			}
			if err := data[operation.Row].SetBonus(pieceCount, index, optionID, currentValue); err != nil {
				return nil, err
			}
			continue
		}
		return nil, fmt.Errorf("unknown set item field %q", operation.Field)
	}
	return writeZoneData(func(w io.Writer) error { return setitemfile.Write(w, data) })
}

func applyPresentItemSets(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := presentitemsetfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := validateRowOperation(operation, len(data)); err != nil {
			return nil, err
		}
		value, err := operationUint(operation, 65535)
		if err != nil {
			return nil, err
		}
		offset := operation.Row * presentitemsetfile.RecordSize
		if operation.Field == "present_set_code" {
			data[operation.Row].SetPresentSetCode(uint16(value))
			allowBytes(allowed, offset, 2)
			continue
		}
		index, part, ok := indexedPairField(operation.Field, "reward_", presentitemsetfile.RewardCount)
		if !ok {
			return nil, fmt.Errorf("unknown present item set field %q", operation.Field)
		}
		count, itemCode, _ := data[operation.Row].Reward(index)
		switch part {
		case "count":
			count = uint16(value)
			allowBytes(allowed, offset+2+index*4, 2)
		case "item_code":
			itemCode = uint16(value)
			allowBytes(allowed, offset+4+index*4, 2)
		default:
			return nil, fmt.Errorf("unknown present item set field %q", operation.Field)
		}
		if err := data[operation.Row].SetReward(index, count, itemCode); err != nil {
			return nil, err
		}
	}
	return writeZoneData(func(w io.Writer) error { return presentitemsetfile.Write(w, data) })
}

func applyPets(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := petfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := validateRowOperation(operation, len(data)); err != nil {
			return nil, err
		}
		offset := operation.Row * petfile.RecordSize
		if operation.Field == "name" {
			value, err := operationString(operation)
			if err != nil {
				return nil, err
			}
			if err := data[operation.Row].SetName(value); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+4, petfile.NameSize)
			continue
		}
		value, err := operationUint(operation, 4294967295)
		if err != nil {
			return nil, err
		}
		switch operation.Field {
		case "group":
			if value > 65535 {
				return nil, fmt.Errorf("group exceeds 65535")
			}
			data[operation.Row].SetGroup(uint16(value))
			allowBytes(allowed, offset, 2)
		case "code":
			if value > petfile.CodeMax {
				return nil, fmt.Errorf("code exceeds %d", petfile.CodeMax)
			}
			if err := data[operation.Row].SetCode(uint16(value)); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+2, 2)
		case "price":
			data[operation.Row].SetPrice(uint32(value))
			allowBytes(allowed, offset+36, 4)
		case "auth":
			if value > 65535 {
				return nil, fmt.Errorf("auth exceeds 65535")
			}
			data[operation.Row].SetAuth(uint16(value))
			allowBytes(allowed, offset+40, 2)
		case "limit":
			if value > 65535 {
				return nil, fmt.Errorf("limit exceeds 65535")
			}
			data[operation.Row].SetLimit(uint16(value))
			allowBytes(allowed, offset+42, 2)
		case "data_1":
			if value > 65535 {
				return nil, fmt.Errorf("data_1 exceeds 65535")
			}
			data[operation.Row].SetData1(uint16(value))
			allowBytes(allowed, offset+44, 2)
		case "data_2":
			if value > 65535 {
				return nil, fmt.Errorf("data_2 exceeds 65535")
			}
			data[operation.Row].SetData2(uint16(value))
			allowBytes(allowed, offset+46, 2)
		default:
			return nil, fmt.Errorf("unknown pet field %q", operation.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return petfile.Write(w, data) })
}

func applyShueCombinations(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := shuecombinationfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := validateRowOperation(operation, len(data)); err != nil {
			return nil, err
		}
		index, ok := fieldIndex(operation.Field, "field_", shuecombinationfile.FieldCount)
		if !ok {
			return nil, fmt.Errorf("unknown Shue combination field %q", operation.Field)
		}
		value, err := operationUint(operation, 65535)
		if err != nil {
			return nil, err
		}
		if err := data[operation.Row].SetField(index, uint16(value)); err != nil {
			return nil, err
		}
		allowBytes(allowed, operation.Row*shuecombinationfile.RecordSize+index*2, 2)
	}
	return writeZoneData(func(w io.Writer) error { return shuecombinationfile.Write(w, data) })
}

func applyLottery(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := lotteryfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := validateRowOperation(operation, len(data)); err != nil {
			return nil, err
		}
		offset := operation.Row * lotteryfile.RecordSize
		if index, ok := fieldIndex(operation.Field, "message_", lotteryfile.MessageCount); ok {
			value, err := operationString(operation)
			if err != nil {
				return nil, err
			}
			if err := data[operation.Row].SetMessage(index, value); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+4+index*lotteryfile.MessageSize, lotteryfile.MessageSize)
			continue
		}
		value, err := operationUint(operation, 65535)
		if err != nil {
			return nil, err
		}
		switch operation.Field {
		case "event_item_id":
			if value > 255 {
				return nil, fmt.Errorf("event_item_id exceeds 255")
			}
			data[operation.Row].SetEventItemID(byte(value))
			allowBytes(allowed, offset, 1)
		case "enabled":
			if value > 255 {
				return nil, fmt.Errorf("enabled exceeds 255")
			}
			data[operation.Row].SetEnabled(byte(value))
			allowBytes(allowed, offset+1, 1)
		case "reward_item_code":
			data[operation.Row].SetRewardItemCode(uint16(value))
			allowBytes(allowed, offset+2, 2)
		default:
			return nil, fmt.Errorf("unknown lottery field %q", operation.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return lotteryfile.Write(w, data) })
}

func applyDerbyGifts(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := derbygiftfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := validateRowOperation(operation, len(data)); err != nil {
			return nil, err
		}
		value, err := operationUint(operation, 4294967295)
		if err != nil {
			return nil, err
		}
		offset := operation.Row * derbygiftfile.RecordSize
		switch operation.Field {
		case "item_code":
			data[operation.Row].SetItemCode(uint32(value))
			allowBytes(allowed, offset, 4)
		case "quantity":
			data[operation.Row].SetQuantity(uint32(value))
			allowBytes(allowed, offset+4, 4)
		case "weight":
			if value > derbygiftfile.MaxProbability {
				return nil, fmt.Errorf("weight exceeds %d", derbygiftfile.MaxProbability)
			}
			data[operation.Row].SetWeight(uint32(value))
			allowBytes(allowed, offset+8, 4)
		default:
			return nil, fmt.Errorf("unknown derby gift field %q", operation.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return derbygiftfile.Write(w, data) })
}

func applyEventItemRewards(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := eventitemrewardfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := validateRowOperation(operation, len(data)); err != nil {
			return nil, err
		}
		offset := operation.Row * eventitemrewardfile.RecordSize
		if operation.Field == "message" {
			value, err := operationString(operation)
			if err != nil {
				return nil, err
			}
			if err := data[operation.Row].SetMessage(value); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+4, eventitemrewardfile.MessageSize)
			continue
		}
		value, err := operationUint(operation, 65535)
		if err != nil {
			return nil, err
		}
		switch operation.Field {
		case "item_code":
			data[operation.Row].SetItemCode(uint16(value))
			allowBytes(allowed, offset, 2)
		case "weight":
			data[operation.Row].SetWeight(uint16(value))
			allowBytes(allowed, offset+2, 2)
		default:
			return nil, fmt.Errorf("unknown event item reward field %q", operation.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return eventitemrewardfile.Write(w, data) })
}

func applyA3Presents(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := a3presentfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := validateRowOperation(operation, len(data)); err != nil {
			return nil, err
		}
		offset := operation.Row * a3presentfile.RecordSize
		if operation.Field == "name" {
			value, err := operationString(operation)
			if err != nil {
				return nil, err
			}
			if err := data[operation.Row].SetName(value); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset, a3presentfile.NameSize)
			continue
		}
		value, err := operationUint(operation, 4294967295)
		if err != nil {
			return nil, err
		}
		if index, part, ok := indexedPairField(operation.Field, "reward_", a3presentfile.RewardCount); ok {
			if value > 65535 {
				return nil, fmt.Errorf("reward value exceeds 65535")
			}
			count, itemCode, _ := data[operation.Row].Reward(index)
			switch part {
			case "count":
				count = uint16(value)
				allowBytes(allowed, offset+a3presentfile.NameSize+index*4, 2)
			case "item_code":
				itemCode = uint16(value)
				allowBytes(allowed, offset+a3presentfile.NameSize+index*4+2, 2)
			default:
				return nil, fmt.Errorf("unknown A3 present field %q", operation.Field)
			}
			if err := data[operation.Row].SetReward(index, count, itemCode); err != nil {
				return nil, err
			}
			continue
		}
		switch operation.Field {
		case "money":
			data[operation.Row].SetMoney(uint32(value))
			allowBytes(allowed, offset+0x21, 4)
		case "lore":
			data[operation.Row].SetLore(uint32(value))
			allowBytes(allowed, offset+0x25, 4)
		case "experience":
			data[operation.Row].SetExperience(uint32(value))
			allowBytes(allowed, offset+0x29, 4)
		case "offered":
			if value > 65535 {
				return nil, fmt.Errorf("offered exceeds 65535")
			}
			data[operation.Row].SetOffered(uint16(value))
			allowBytes(allowed, offset+0x2d, 2)
		default:
			return nil, fmt.Errorf("unknown A3 present field %q", operation.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return a3presentfile.Write(w, data) })
}

func validateRowOperation(operation ZoneDataOperation, length int) error {
	if operation.Scope != "row" {
		return fmt.Errorf("invalid operation scope %q", operation.Scope)
	}
	return validateRow(operation.Row, length)
}

func operationString(operation ZoneDataOperation) (string, error) {
	var value string
	if err := json.Unmarshal(operation.Value, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", operation.Field)
	}
	return value, nil
}

func indexedPairField(field string, prefix string, count int) (int, string, bool) {
	if !strings.HasPrefix(field, prefix) {
		return 0, "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(field, prefix), "_", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	index, err := strconv.Atoi(parts[0])
	return index, parts[1], err == nil && index >= 0 && index < count
}
